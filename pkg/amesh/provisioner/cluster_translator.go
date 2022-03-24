package provisioner

import (
	"errors"
	"github.com/api7/amesh/pkg/apisix"
	"github.com/api7/gopkg/pkg/id"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"go.uber.org/zap"
)

func (p *xdsProvisioner) TranslateCluster(c *clusterv3.Cluster) (*apisix.Upstream, error) {
	ups := &apisix.Upstream{
		Name:  c.Name,
		Id:    id.GenID(c.Name),
		Nodes: []*apisix.Node{},
	}
	if err := p.translateClusterLbPolicy(c, ups); err != nil {
		return nil, err
	}
	if err := p.translateClusterTimeoutSettings(c, ups); err != nil {
		return nil, err
	}
	if err := p.translateClusterLoadAssignments(c, ups); err != nil {
		//if err == ErrRequireFurtherEDS {
		//	return ups, err
		//}
		return nil, err
	}

	p.logger.Debugw("got upstream after parsing cluster",
		zap.Any("cluster", c),
	)

	return ups, nil
}

func (p *xdsProvisioner) translateClusterLbPolicy(c *clusterv3.Cluster, ups *apisix.Upstream) error {
	switch c.GetLbPolicy() {
	case clusterv3.Cluster_ROUND_ROBIN:
		ups.Type = "roundrobin"
	case clusterv3.Cluster_LEAST_REQUEST:
		// Apache APISIX's lease_conn policy is same to lease request.
		// But is doesn't expose configuration items. So LbConfig field
		// is ignored.
		ups.Type = "least_conn"
	default:
		// Apache APISIX doesn't support Random, Manglev. In addition,
		// also RinghHash (Consistent Hash) is available but the configurations
		// like key is in RouteConfiguration, so we cannot use it either.
		p.logger.Warnw("ignore cluster with unsupported load balancer",
			zap.String("cluster_name", c.Name),
			zap.String("lb_policy", c.GetLbPolicy().String()),
		)
		return errors.New("ErrFeatureNotSupportedYet")
	}
	return nil
}

func (p *xdsProvisioner) translateClusterTimeoutSettings(c *clusterv3.Cluster, ups *apisix.Upstream) error {
	if c.GetConnectTimeout() != nil {
		ups.Timeout = &apisix.Timeout{
			Connect: float64((*c.GetConnectTimeout()).Seconds),
			Read:    60,
			Send:    60,
		}
	}
	return nil
}

func (p *xdsProvisioner) translateClusterLoadAssignments(c *clusterv3.Cluster, ups *apisix.Upstream) error {
	if c.GetClusterType() != nil {
		return nil
	}
	switch c.GetType() {
	case clusterv3.Cluster_EDS:
		// TODO:
		return nil
	default:
		nodes, err := p.TranslateClusterLoadAssignment(c.GetLoadAssignment())
		if err != nil {
			return err
		}
		ups.Nodes = nodes
		return nil
	}
}

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
