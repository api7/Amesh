package provisioner

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ameshapi "github.com/api7/amesh/api/proto/v1"
)

type ameshProvisioner struct {
	src string

	namespace string
	name      string

	logger *log.Logger

	configLock     sync.RWMutex
	config         []*ameshapi.AmeshPlugin
	configRevision map[string]string

	resetCh chan error
}

func NewAmeshProvisioner(src, logLevel, logOutput string) (*ameshProvisioner, error) {
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		return nil, errors.New("env variable NAMESPACE not found")
	}
	name := os.Getenv("POD_NAME")
	if name == "" {
		return nil, errors.New("env variable POD_NAME not found")
	}

	logger, err := log.NewLogger(
		log.WithOutputFile(logOutput),
		log.WithLogLevel(logLevel),
		log.WithContext("amesh-grpc-provisioner"),
	)
	if err != nil {
		return nil, err
	}

	return &ameshProvisioner{
		src:            src,
		namespace:      namespace,
		name:           name,
		logger:         logger,
		configRevision: map[string]string{},
		resetCh:        make(chan error),
	}, nil
}

func (p *ameshProvisioner) Run(stop <-chan struct{}) error {
	p.logger.Infow("amesh provisioner started")
	defer p.logger.Info("amesh provisioner exited")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		conn, err := grpc.DialContext(ctx, p.src,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			cancel()
			p.logger.Errorw("failed to conn amesh source",
				zap.Error(err),
				zap.String("amesh_source", p.src),
			)
			return err
		}
		cleanup := func() {
			cancel()
			if err := conn.Close(); err != nil {
				p.logger.Errorw("failed to close gRPC connection to XDS config source",
					zap.Error(err),
					zap.String("src", p.src),
				)
			}
		}
		p.logger.Info("amesh connected")

		if err := p.run(ctx, conn); err != nil {
			cleanup()
			return err
		}

		select {
		case <-stop:
			cleanup()
			return nil
		case err = <-p.resetCh:
			p.logger.Errorw("amesh grpc client reset, closing",
				zap.Error(err),
			)
			cleanup()
			p.logger.Errorw("amesh grpc client reset, try reconnect",
				zap.Error(err),
			)
			continue
		}
	}
}

func (p *ameshProvisioner) run(ctx context.Context, conn *grpc.ClientConn) error {
	client, err := ameshapi.NewAmeshServiceClient(conn).StreamPlugins(ctx, &ameshapi.PluginsRequest{
		Instance: &ameshapi.Instance{
			Key: p.namespace + "/" + p.name,
		},
	})
	if err != nil {
		return err
	}

	go p.recvLoop(ctx, client)

	return nil
}

// recvLoop receives DiscoveryResponse objects from the wire stream and sends them
// to the recvCh channel.
func (p *ameshProvisioner) recvLoop(ctx context.Context, client ameshapi.AmeshService_StreamPluginsClient) {
	for {
		dr, err := client.Recv()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				p.logger.Errorw("failed to receive discovery response",
					zap.Error(err),
				)
				errMsg := err.Error()
				if strings.Contains(errMsg, "transport is closing") ||
					strings.Contains(errMsg, "DeadlineExceeded") {
					p.logger.Errorw("trigger grpc client reset",
						zap.Error(err),
					)
					p.resetCh <- err
					return
				}
				continue
			}
		}
		//p.logger.Debugw("got discovery response",
		//	zap.String("type", dr.TypeUrl),
		//	zap.Any("body", dr),
		//)
		go func(dr *ameshapi.PluginsResponse) {
			select {
			case <-ctx.Done():
			default:
				p.updatePlugins(dr)
			}
		}(dr)
	}
}

func (p *ameshProvisioner) updatePlugins(resp *ameshapi.PluginsResponse) {
	if resp.ErrorMessage != nil {
		log.Errorw("received response with error", zap.Any("error_message", resp.ErrorMessage))
		return
	}

	var plugins []*ameshapi.AmeshPlugin
	p.configLock.Lock()
	defer p.configLock.Unlock()

	revisionMap := p.configRevision
	p.configRevision = map[string]string{}
	for _, plugin := range resp.Plugins {
		if revision, ok := revisionMap[plugin.Name]; ok && plugin.Version <= revision {
			continue
		}
		p.configRevision[plugin.Name] = plugin.Version
		plugins = append(plugins, plugin.Plugins...)
	}
	p.config = plugins

	// TODO: TRIGGER ROUTES CHANGE
}
