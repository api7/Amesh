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
//
package provisioner

import (
	"errors"
	"reflect"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"go.uber.org/zap"

	"github.com/api7/amesh/pkg/apisix"
)

var (
	ErrorRequireFurtherEDS = errors.New("require further eds")
)

func (p *xdsProvisioner) TranslateClusterLoadAssignment(la *endpointv3.ClusterLoadAssignment) ([]*apisix.Node, error) {
	var nodes []*apisix.Node
	if len(la.GetEndpoints()) == 0 {
		return nil, ErrorRequireFurtherEDS
	}
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
						p.logger.Warnw("ignore endpoint with unsupported named port type",
							zap.Any("type", reflect.TypeOf(port)),
							zap.Any("endpoint", ep),
						)
						continue
					}
				default:
					p.logger.Warnw("ignore endpoint with unsupported address type",
						zap.Any("type", reflect.TypeOf(addr)),
						zap.Any("endpoint", ep),
					)
					continue
				}
			default:
				p.logger.Warnw("ignore endpoint with unsupported endpoint type ",
					zap.Any("type", reflect.TypeOf(identifier)),
					zap.Any("endpoint", ep),
				)
				continue
			}
			p.logger.Debugw("got node after parsing endpoint",
				zap.Any("node", node),
				zap.Any("endpoint", ep),
			)
			// Currently, Apache APISIX doesn't use the metadata field.
			// So we don't pass ep.Metadata.
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}
