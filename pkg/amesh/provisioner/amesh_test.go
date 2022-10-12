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
	"strings"
	"testing"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ameshapi "github.com/api7/amesh/api/proto/v1"
)

// func TestAmeshGrpcServer(t *testing.T) {
func AmeshGrpcServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	svc := "amesh-controller.istio-system.svc.cluster.local:15810"
	svc = "localhost:15810"
	conn, err := grpc.DialContext(ctx, svc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		cancel()
		log.Errorw("failed to conn amesh source",
			zap.Error(err),
		)
	}
	client, err := ameshapi.NewAmeshServiceClient(conn).StreamPlugins(context.Background(), &ameshapi.PluginsRequest{
		Instance: &ameshapi.Instance{
			Key: "test/consumer",
		},
	})

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
		log.Errorw("got discovery response",
			zap.Any("body", dr),
		)
	}
}
