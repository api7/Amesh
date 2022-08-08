package provisioner

import (
	"testing"

	"github.com/api7/gopkg/pkg/log"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
)

func TestTranslateClusterLoadAssignment(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}
	la := &endpointv3.ClusterLoadAssignment{
		ClusterName: "test",
		Endpoints: []*endpointv3.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpointv3.LbEndpoint{
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_TCP,
											Address:  "10.0.3.1",
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: 8000,
											},
										},
									},
								},
							},
						},

						// Will override outer load balancing weight
						LoadBalancingWeight: &wrappers.UInt32Value{
							Value: 100,
						},
					},
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_TCP,
											Address:  "10.0.3.2",
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: 8000,
											},
										},
									},
								},
							},
						},
					},
					{
						// Only support LbEndpoint_Endpoint, will be ignored.
						HostIdentifier: &endpointv3.LbEndpoint_EndpointName{},
					},
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									// Only support Address_SocketAddress, will be ignored.
									Address: &corev3.Address_Pipe{},
								},
							},
						},
					},
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											// Only support TCP, will be ignored.
											Protocol: corev3.SocketAddress_UDP,
											Address:  "10.0.3.11",
											PortSpecifier: &corev3.SocketAddress_PortValue{
												PortValue: 8000,
											},
										},
									},
								},
							},
						},
					},
					{
						HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
							Endpoint: &endpointv3.Endpoint{
								Address: &corev3.Address{
									Address: &corev3.Address_SocketAddress{
										SocketAddress: &corev3.SocketAddress{
											Protocol: corev3.SocketAddress_TCP,
											Address:  "10.0.3.12",
											// Only support SocketAddress_PortValue, will be ignored.
											PortSpecifier: &corev3.SocketAddress_NamedPort{
												NamedPort: "http",
											},
										},
									},
								},
							},
						},
					},
				},
				LoadBalancingWeight: &wrappers.UInt32Value{
					Value: 50,
				},
			},
		},
	}
	nodes, err := a.TranslateClusterLoadAssignment(la)
	assert.Nil(t, err)
	assert.Len(t, nodes, 2)
	assert.Equal(t, nodes[0].Port, int32(8000))
	assert.Equal(t, nodes[0].Weight, int32(100))
	assert.Equal(t, nodes[0].Host, "10.0.3.1")
	assert.Equal(t, nodes[1].Port, int32(8000))
	assert.Equal(t, nodes[1].Weight, int32(50))
	assert.Equal(t, nodes[1].Host, "10.0.3.2")
}
