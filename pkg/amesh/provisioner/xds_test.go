// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/api7/gopkg/pkg/log"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

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

	go func() {
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
					panic(err)
				}
			}

			switch dr.TypeUrl {
			case types.ClusterUrl:
				for _, res := range dr.GetResources() {
					var cluster clusterv3.Cluster
					err := anypb.UnmarshalTo(res, &cluster, proto.UnmarshalOptions{
						DiscardUnknown: false,
					})

					if err != nil {
						log.Errorw("unmarshal cluster failed, skipped",
							zap.Error(err),
							zap.Any("resource", res),
						)
						continue
					}
					log.Infow(color.GreenString("process cluster response"),
						zap.Any("name", cluster.Name),
						zap.Any("cluster", &cluster),
					)
				}
				return
			case types.RouteConfigurationUrl:
				for _, res := range dr.GetResources() {
					var route routev3.RouteConfiguration
					err := anypb.UnmarshalTo(res, &route, proto.UnmarshalOptions{
						DiscardUnknown: true,
					})

					if err != nil {
						log.Errorw("found invalid RouteConfiguration resource",
							zap.Error(err),
							zap.Any("resource", res),
						)
						continue
					}

					log.Debugw(color.GreenString("process route configurations"),
						zap.Any("route", &route),
					)
				}
				return
			case types.ClusterLoadAssignmentUrl:
				for _, res := range dr.GetResources() {
					var cla endpointv3.ClusterLoadAssignment
					err := anypb.UnmarshalTo(res, &cla, proto.UnmarshalOptions{
						DiscardUnknown: true,
					})
					if err != nil {
						log.Errorw("failed to unmarshal ClusterLoadAssignment",
							zap.Error(err),
							zap.Any("resource", res),
						)
						continue
					}

					log.Infow(color.GreenString("process cluster load assignment response"),
						zap.Any("cla", &cla),
					)
				}
				return
			}
			data, err := json.Marshal(dr)
			_ = data
			textString, err := proto.Marshal(dr)
			_ = textString
			if err != nil {
				log.Errorw("failed to marshal data",
					zap.Error(err),
				)
			} else {
				log.Info("got discovery response",
					zap.Any("body", textString),
				)
			}
		}
	}()

	node := &corev3.Node{
		Id:            xds.GenNodeId(uuid.NewString(), "127.0.0.1", "default.svc.cluster.local"),
		UserAgentName: fmt.Sprintf("amesh/%s", version.Short()),
	}

	go func() {
		drs := []*discoveryv3.DiscoveryRequest{
			//{
			//	Node:    node,
			//	TypeUrl: types.ListenerUrl,
			//},
			{
				Node:    node,
				TypeUrl: types.ClusterUrl,
			},
			{
				Node:    node,
				TypeUrl: types.RouteConfigurationUrl,
			},
			//{
			//	Node:          node,
			//	TypeUrl:       types.ClusterLoadAssignmentUrl,
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
	}
}
