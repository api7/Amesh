// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioner

import (
	"context"
	"encoding/json"
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
	"github.com/api7/amesh/pkg/amesh/types"
)

var (
	_ types.AmeshPluginProvider = (*ameshProvisioner)(nil)
)

type ameshProvisioner struct {
	src string

	namespace string
	name      string

	logger *log.Logger

	configLock     sync.RWMutex
	config         []*types.ApisixPlugin
	configRevision map[string]string

	evChan  chan struct{}
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

		evChan:  make(chan struct{}),
		resetCh: make(chan error),
	}, nil
}

func (p *ameshProvisioner) Run(stop <-chan struct{}) error {
	p.logger.Infow("amesh provisioner started")
	defer p.logger.Info("amesh provisioner exited")

	for {
		dialCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		conn, err := grpc.DialContext(dialCtx, p.src,
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

		if err := p.run(stop, conn); err != nil {
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

func (p *ameshProvisioner) run(stop <-chan struct{}, conn *grpc.ClientConn) error {
	client, err := ameshapi.NewAmeshServiceClient(conn).StreamPlugins(context.Background(), &ameshapi.PluginsRequest{
		Instance: &ameshapi.Instance{
			Key: p.namespace + "/" + p.name,
		},
	})
	if err != nil {
		return err
	}

	go p.recvLoop(stop, client)

	return nil
}

// recvLoop receives DiscoveryResponse objects from the wire stream and sends them
// to the recvCh channel.
func (p *ameshProvisioner) recvLoop(stop <-chan struct{}, client ameshapi.AmeshService_StreamPluginsClient) {

	// TODO: DELET EVENT

	for {
		dr, err := client.Recv()
		if err != nil {
			select {
			case <-stop:
				return
			default:
				p.logger.Errorw("failed to receive amesh plugin response",
					zap.Error(err),
				)
				errMsg := err.Error()
				if strings.Contains(errMsg, "transport is closing") ||
					strings.Contains(errMsg, "DeadlineExceeded") ||
					strings.Contains(errMsg, "EOF") {
					p.logger.Errorw("trigger amesh grpc client reset",
						zap.Error(err),
					)
					p.resetCh <- err
					return
				}
				continue
			}
		}
		p.logger.Debugw("got amesh plugin config",
			zap.Any("body", dr),
		)
		go func(dr *ameshapi.PluginsResponse) {
			select {
			case <-stop:
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

	var plugins []*types.ApisixPlugin
	p.configLock.Lock()
	defer p.configLock.Unlock()

	revisionMap := p.configRevision
	p.configRevision = map[string]string{}
	for _, plugin := range resp.Plugins {
		if revision, ok := revisionMap[plugin.Name]; ok && plugin.Version <= revision {
			continue
		}
		p.configRevision[plugin.Name] = plugin.Version

		var apisixPlugins []*types.ApisixPlugin
		pluginValues := map[string]map[string]interface{}{}
		for _, plugin := range plugin.Plugins {
			var anyValue map[string]interface{}
			err := json.Unmarshal([]byte(plugin.Config), &anyValue)
			if err != nil {
				p.logger.Errorw("failed to unmarshal plugin config",
					zap.Error(err),
					zap.Any("config", plugin.Config),
				)
				continue
			}
			apisixPlugins = append(apisixPlugins, &types.ApisixPlugin{
				Type:   plugin.Type,
				Name:   plugin.Name,
				Config: anyValue,
			})
			pluginValues[plugin.Name] = anyValue
		}

		plugins = append(plugins, apisixPlugins...)
	}
	p.config = plugins

	p.evChan <- struct{}{}
}

func (p *ameshProvisioner) GetPlugins() []*types.ApisixPlugin {
	p.configLock.RLock()
	defer p.configLock.RUnlock()
	return p.config
}

func (p *ameshProvisioner) EventsChannel() <-chan struct{} {
	return p.evChan
}
