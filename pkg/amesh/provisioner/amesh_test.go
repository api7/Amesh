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

func TestAmeshGrpcServer(t *testing.T) {
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
	client, err := ameshapi.NewAmeshServiceClient(conn).StreamPlugins(ctx, &ameshapi.PluginsRequest{
		Instance: &ameshapi.Instance{
			Key: "test/consumer",
		},
	})

	for {
		dr, err := client.Recv()
		if err != nil {
			select {
			case <-ctx.Done():
				return
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
		log.Debugw("got discovery response",
			zap.Any("body", dr),
		)
	}
}
