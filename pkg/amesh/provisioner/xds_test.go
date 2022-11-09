package provisioner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/api7/gopkg/pkg/log"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/api7/amesh/pkg/amesh/types"
	"github.com/api7/amesh/pkg/version"
	"github.com/api7/amesh/pkg/xds"
)

func ConnXdsGrpcServer(t *testing.T) {
	l, err := log.NewLogger(
		log.WithOutputFile("stderr"),
		log.WithLogLevel("debug"),
	)
	if err != nil {
		panic(err)
	}
	log.DefaultLogger = l
	//func XdsGrpcServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	log.Info("testing")
	svc := "istiod.istio-system.svc.cluster.local:15010"
	svc = "localhost:15010"
	conn, err := grpc.DialContext(ctx, svc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		cancel()
		log.Errorw("failed to conn xds source",
			zap.Error(err),
		)
	}
	log.Info("grpc connected")

	client, err := discoveryv3.NewAggregatedDiscoveryServiceClient(conn).StreamAggregatedResources(context.Background())
	if err != nil {
		log.Errorw("failed to conn xds client",
			zap.Error(err),
		)
		return
	}
	log.Info("xds client started")

	node := &corev3.Node{
		Id:            xds.GenNodeId(uuid.NewString(), "127.0.0.1", "default.svc.cluster.local"),
		UserAgentName: fmt.Sprintf("amesh/%s", version.Short()),
	}

	go func() {
		drs := []*discoveryv3.DiscoveryRequest{
			{
				Node:    node,
				TypeUrl: types.ListenerUrl,
			},
			{
				Node:    node,
				TypeUrl: types.ClusterUrl,
			},
			//{
			//	Node:    p.node,
			//	TypeUrl: types.RouteConfigurationUrl,
			//},
			//{
			//	Node:    p.node,
			//	TypeUrl: types.ClusterLoadAssignmentUrl,
			//},
		}

		for _, dr := range drs {
			log.Infow("sending initial discovery request",
				zap.String("type_url", dr.TypeUrl),
			)
			if err := client.Send(dr); err != nil {
				log.Errorw("failed to send discovery request",
					zap.Error(err),
				)
			}
		}
	}()

	for {
		dr, err := client.Recv()
		if err != nil {
			select {
			default:
				log.Errorw("failed to receive discovery response",
					zap.Error(err),
				)
				errMsg := err.Error()
				if strings.Contains(errMsg, "transport is closing") ||
					strings.Contains(errMsg, "DeadlineExceeded") {
					log.Errorw("trigger grpc client reset",
						zap.Error(err),
					)
					return
				}
				continue
			}
		}

		data, err := json.Marshal(dr)
		_ = data
		test := proto.MarshalTextString(dr)
		_ = test
		if err != nil {
			log.Errorw("failed to marshal data",
				zap.Error(err),
			)
		} else {
			log.Info("got discovery response",
				zap.Any("body", test),
			)
		}
	}
}
