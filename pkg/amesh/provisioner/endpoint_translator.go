package provisioner

import (
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"go.uber.org/zap"

	"github.com/api7/amesh/pkg/apisix"
)

func (p *xdsProvisioner) TranslateClusterLoadAssignment(la *endpointv3.ClusterLoadAssignment) ([]*apisix.Node, error) {
	var nodes []*apisix.Node
	for _, eps := range la.GetEndpoints() {
		var weight int32
		if eps.GetLoadBalancingWeight() != nil {
			weight = int32(eps.GetLoadBalancingWeight().GetValue())
		} else {
			weight = 100
		}
		for _, ep := range eps.LbEndpoints {
			node := &apisix.Node{
				Weight: weight,
			}
			if ep.GetLoadBalancingWeight() != nil {
				node.Weight = int32(ep.GetLoadBalancingWeight().GetValue())
			}
			switch identifier := ep.GetHostIdentifier().(type) {
			case *endpointv3.LbEndpoint_Endpoint:
				switch addr := identifier.Endpoint.Address.Address.(type) {
				case *corev3.Address_SocketAddress:
					if addr.SocketAddress.GetProtocol() != corev3.SocketAddress_TCP {
						p.logger.Warnw("ignore endpoint with non-tcp protocol",
							zap.Any("endpoint", ep),
						)
						continue
					}
					node.Host = addr.SocketAddress.GetAddress()
					switch port := addr.SocketAddress.GetPortSpecifier().(type) {
					case *corev3.SocketAddress_PortValue:
						node.Port = int32(port.PortValue)
					case *corev3.SocketAddress_NamedPort:
						p.logger.Warnw("ignore endpoint with unsupported named port",
							zap.Any("endpoint", ep),
						)
						continue
					}
				default:
					p.logger.Warnw("ignore endpoint with unsupported address type",
						zap.Any("endpoint", ep),
					)
					continue
				}
			default:
				p.logger.Warnw("ignore endpoint with unknown endpoint type ",
					zap.Any("endpoint", ep),
				)
				continue
			}
			p.logger.Debugw("got node after parsing endpoint",
				zap.Any("node", node),
				zap.Any("endpoint", ep),
			)
			// Currently Apache APISIX doesn't use the metadata field.
			// So we don't pass ep.Metadata.
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}
