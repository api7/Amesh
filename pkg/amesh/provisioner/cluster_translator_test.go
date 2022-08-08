package provisioner

import (
	"testing"

	"github.com/api7/gopkg/pkg/log"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/pkg/apisix"
)

func TestTranslateClusterLbPolicy(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}
	c := &clusterv3.Cluster{
		Name:     "test",
		LbPolicy: clusterv3.Cluster_ROUND_ROBIN,
	}
	var ups apisix.Upstream
	assert.Nil(t, a.translateClusterLbPolicy(c, &ups))
	assert.Equal(t, ups.Type, apisix.LoadBalanceType("roundrobin"))
	c.LbPolicy = clusterv3.Cluster_LEAST_REQUEST
	assert.Nil(t, a.translateClusterLbPolicy(c, &ups))
	assert.Equal(t, ups.Type, apisix.LoadBalanceType("least_conn"))

	// Unsupported
	c.LbPolicy = clusterv3.Cluster_RING_HASH
	assert.Equal(t, a.translateClusterLbPolicy(c, &ups), nil)
}

func TestTranslateClusterTimeoutSettings(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}
	c := &clusterv3.Cluster{
		Name: "test",
		ConnectTimeout: &duration.Duration{
			Seconds: 10,
		},
	}
	var ups apisix.Upstream
	assert.Nil(t, a.translateClusterTimeoutSettings(c, &ups))
	assert.Equal(t, ups.Timeout.Connect, float64(10))
}
