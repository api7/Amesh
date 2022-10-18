// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package provisioner

import (
	"encoding/json"
	"errors"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/any"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/api7/amesh/pkg/amesh/types"
	"github.com/api7/amesh/pkg/apisix"
)

func (p *xdsProvisioner) processRouteConfigurationV3(res *any.Any) ([]*apisix.Route, error) {
	var route routev3.RouteConfiguration
	err := anypb.UnmarshalTo(res, &route, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})

	p.logger.Debugw("process route configurations",
		zap.Any("route", &route),
	)

	if err != nil {
		p.logger.Errorw("found invalid RouteConfiguration resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil, err
	}

	routes, err := p.TranslateRouteConfiguration(&route, p.routeOwnership)
	if err != nil {
		p.logger.Errorw("failed to translate RouteConfiguration to APISIX routes",
			zap.Error(err),
			zap.Any("route", &route),
		)
		return nil, err
	}
	return routes, nil
}

func (p *xdsProvisioner) processStaticRouteConfigurations(rcs []*routev3.RouteConfiguration) ([]*apisix.Route, error) {
	var (
		routes []*apisix.Route
	)
	for _, rc := range rcs {
		p.logger.Debugw("process static route configurations",
			zap.Any("static_route", rc),
		)
		route, err := p.TranslateRouteConfiguration(rc, p.routeOwnership)
		if err != nil {
			p.logger.Errorw("failed to translate static RouteConfiguration to APISIX routes",
				zap.Error(err),
				zap.Any("route", &route),
			)
			return nil, err
		}
	}
	return routes, nil
}

func (p *xdsProvisioner) processClusterV3(res *any.Any) (*apisix.Upstream, error) {
	var cluster clusterv3.Cluster
	err := anypb.UnmarshalTo(res, &cluster, proto.UnmarshalOptions{
		DiscardUnknown: true,
	})
	if err != nil {
		p.logger.Errorw("found invalid Cluster resource",
			zap.Error(err),
			zap.Any("resource", res),
		)
		return nil, err
	}
	p.logger.Debugw("process cluster configurations",
		zap.Any("cluster", &cluster),
	)

	ups, err := p.TranslateCluster(&cluster)
	if err != nil {
		return nil, err
	}
	return ups, nil
}

func (p *xdsProvisioner) processClusterLoadAssignmentV3(cla *endpointv3.ClusterLoadAssignment) (*apisix.Upstream, error) {
	ups, ok := p.upstreams[cla.ClusterName]
	if !ok {
		p.logger.Warnw("ClusterLoadAssignment referred cluster not found",
			zap.String("reason", "cluster unknown"),
			zap.String("cluster_name", cla.ClusterName),
		)
		return nil, errors.New("UnknownClusterName")
	}

	nodes, err := p.TranslateClusterLoadAssignment(cla)
	if err == types.ErrorRequireFurtherEDS {
		return nil, err
	}
	if err != nil {
		p.logger.Errorw("failed to translate ClusterLoadAssignment",
			zap.Error(err),
			zap.Any("cla", cla),
		)
		return nil, err
	}

	// Do not set on the original ups to avoid race conditions.
	data, err := json.Marshal(ups)
	if err != nil {
		return nil, err
	}
	var newUps apisix.Upstream
	err = json.Unmarshal(data, &newUps)
	if err != nil {
		return nil, err
	}

	newUps.Nodes = nodes
	return &newUps, nil
}
