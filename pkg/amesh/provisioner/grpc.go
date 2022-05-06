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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/api7/amesh/pkg/amesh/types"
	"github.com/api7/amesh/pkg/amesh/util"
	"github.com/api7/amesh/pkg/apisix"
	"github.com/api7/amesh/pkg/version"
	"github.com/api7/amesh/pkg/xds"
)

type xdsProvisioner struct {
	src    string
	node   *corev3.Node
	logger *log.Logger

	sendCh  chan *discoveryv3.DiscoveryRequest
	recvCh  chan *discoveryv3.DiscoveryResponse
	evChan  chan []types.Event
	resetCh chan error

	// route name -> addr
	// find the listener (address) owner, an extra match
	// condition will be patched to the APISIX route.
	// "connection_original_dst == <ip>:<port>"
	routeOwnership map[string]string
	// static route configuration from listeners.
	staticRouteConfigurations []*routev3.RouteConfiguration
	// last state of routes.
	routes []*apisix.Route
	// last state of upstreams.
	// map is necessary since EDS requires the original cluster
	// by the name.
	upstreams map[string]*apisix.Upstream
	// this map enrolls all clusters that require further EDS requests.
	edsRequiredClusters util.StringSet
}

type Config struct {
	// Running Id of this instance, it will be filled by
	// a random string when the instance started.
	RunId string
	// The minimum log level that will be printed.
	LogLevel string `json:"log_level" yaml:"log_level"`
	// The destination of logs.
	LogOutput string `json:"log_output" yaml:"log_output"`
	// The xds source
	XDSConfigSource string `json:"xds_config_source" yaml:"xds_config_source"`

	Namespace string
	IpAddress string
}

func NewXDSProvisioner(cfg *Config) (types.Provisioner, error) {
	logger, err := log.NewLogger(
		log.WithOutputFile(cfg.LogOutput),
		log.WithLogLevel(cfg.LogLevel),
		log.WithContext("xds-grpc-provisioner"),
	)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(cfg.XDSConfigSource, "grpc://") {
		return nil, errors.New("bad xds config source")
	}
	src := strings.TrimPrefix(cfg.XDSConfigSource, "grpc://")

	// TODO make domain suffix configurable
	dnsDomain := cfg.Namespace + ".svc.cluster.local"
	node := &corev3.Node{
		Id:            xds.GenNodeId(cfg.RunId, cfg.IpAddress, dnsDomain),
		UserAgentName: fmt.Sprintf("amesh/%s", version.Short()),
	}

	return &xdsProvisioner{
		src:    src,
		node:   node,
		logger: logger,

		sendCh:  make(chan *discoveryv3.DiscoveryRequest),
		recvCh:  make(chan *discoveryv3.DiscoveryResponse),
		evChan:  make(chan []types.Event),
		resetCh: make(chan error),
	}, nil
}

func (p *xdsProvisioner) EventsChannel() <-chan []types.Event {
	return p.evChan
}

func (p *xdsProvisioner) Run(stop <-chan struct{}) error {
	p.logger.Infow("provisioner started")
	defer p.logger.Info("provisioner exited")
	defer close(p.evChan)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		conn, err := grpc.DialContext(ctx, p.src,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			cancel()
			p.logger.Errorw("failed to conn xds source",
				zap.Error(err),
				zap.String("xds_source", p.src),
			)
			return err
		}
		cleanup := func() {
			cancel()
			if err := conn.Close(); err != nil {
				p.logger.Errorw("failed to close gRPC connection to XDS config source",
					zap.Error(err),
					zap.String("src", p.src),
				)
			}
		}
		p.logger.Info("xds connected")

		if err := p.run(ctx, conn); err != nil {
			cleanup()
			return err
		}

		select {
		case <-stop:
			cleanup()
			return nil
		case err = <-p.resetCh:
			p.logger.Errorw("grpc client reset, closing",
				zap.Error(err),
			)
			cleanup()
			p.logger.Errorw("grpc client reset, try reconnect",
				zap.Error(err),
			)
			continue
		}
	}
}

func (p *xdsProvisioner) run(ctx context.Context, conn *grpc.ClientConn) error {
	client, err := discoveryv3.NewAggregatedDiscoveryServiceClient(conn).StreamAggregatedResources(ctx)
	if err != nil {
		return err
	}

	go p.sendLoop(ctx, client)
	go p.recvLoop(ctx, client)
	go p.translateLoop(ctx)
	go p.firstSend()

	return nil
}

func (p *xdsProvisioner) firstSend() {
	dr1 := &discoveryv3.DiscoveryRequest{
		Node:    p.node,
		TypeUrl: types.ListenerUrl,
	}
	dr2 := &discoveryv3.DiscoveryRequest{
		Node:    p.node,
		TypeUrl: types.ClusterUrl,
	}
	//dr3 := &discoveryv3.DiscoveryRequest{
	//	Node:    p.node,
	//	TypeUrl: types.RouteConfigurationUrl,
	//}

	p.sendCh <- dr1
	p.sendCh <- dr2
	p.logger.Debugw("sent initial discovery requests for listeners and clusters")
}

// sendLoop receives pending DiscoveryRequest objects and sends them to client.
// Send operation will be retried continuously until successful or the context is
// cancelled.
func (p *xdsProvisioner) sendLoop(ctx context.Context, client discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case dr := <-p.sendCh:
			p.logger.Debugw("sending discovery request",
				zap.String("type", dr.TypeUrl),
				zap.Any("body", dr),
			)
			go func(dr *discoveryv3.DiscoveryRequest) {
				if err := client.Send(dr); err != nil {
					p.logger.Errorw("failed to send discovery request",
						zap.Error(err),
						zap.String("xds_source", p.src),
					)
				}
			}(dr)
		}
	}
}

// recvLoop receives DiscoveryResponse objects from the wire stream and sends them
// to the recvCh channel.
func (p *xdsProvisioner) recvLoop(ctx context.Context, client discoveryv3.AggregatedDiscoveryService_StreamAggregatedResourcesClient) {
	for {
		dr, err := client.Recv()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				p.logger.Errorw("failed to receive discovery response",
					zap.Error(err),
				)
				errMsg := err.Error()
				if strings.Contains(errMsg, "transport is closing") ||
					strings.Contains(errMsg, "DeadlineExceeded") {
					p.logger.Errorw("trigger grpc client reset",
						zap.Error(err),
					)
					p.resetCh <- err
					return
				}
				continue
			}
		}
		//p.logger.Debugw("got discovery response",
		//	zap.String("type", dr.TypeUrl),
		//	zap.Any("body", dr),
		//)
		go func(dr *discoveryv3.DiscoveryResponse) {
			select {
			case <-ctx.Done():
			case p.recvCh <- dr:
			}
		}(dr)
	}
}

// translateLoop mediates the input DiscoveryResponse objects, translating
// them APISIX resources, and generating an ACK request ultimately.
func (p *xdsProvisioner) translateLoop(ctx context.Context) {
	var verInfo string
	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-p.recvCh:
			ackReq := &discoveryv3.DiscoveryRequest{
				Node:          p.node,
				TypeUrl:       resp.TypeUrl,
				ResponseNonce: resp.Nonce,
			}
			if err := p.translate(resp); err != nil {
				ackReq.ErrorDetail = &status.Status{
					Code:    int32(code.Code_INVALID_ARGUMENT),
					Message: err.Error(),
				}
			} else {
				verInfo = resp.VersionInfo
			}
			ackReq.VersionInfo = verInfo
			p.sendCh <- ackReq
		}
	}
}

func (p *xdsProvisioner) translate(resp *discoveryv3.DiscoveryResponse) error {
	var (
		// Since the type url is fixed, only one field is filled in newManifest and oldManifest.
		newManifest util.Manifest
		oldManifest util.Manifest
		events      []types.Event
	)
	// As we use ADS, the TypeUrl field indicates the resource type already.
	switch resp.GetTypeUrl() {
	case types.RouteConfigurationUrl:
		for _, res := range resp.GetResources() {
			partial, err := p.processRouteConfigurationV3(res)
			if err != nil {
				return err
			}
			newManifest.Routes = append(newManifest.Routes, partial...)
		}
		if p.staticRouteConfigurations != nil {
			partial, err := p.processStaticRouteConfigurations(p.staticRouteConfigurations)
			if err != nil {
				return err
			}
			newManifest.Routes = append(newManifest.Routes, partial...)
		}
		oldManifest.Routes = p.routes
		p.routes = newManifest.Routes

	case types.ClusterUrl:
		newUps := make(map[string]*apisix.Upstream)
		oldEdsRequiredClusters := p.edsRequiredClusters
		p.edsRequiredClusters = util.StringSet{}
		for _, res := range resp.GetResources() {
			var cluster clusterv3.Cluster
			err := anypb.UnmarshalTo(res, &cluster, proto.UnmarshalOptions{
				DiscardUnknown: true,
			})
			if err != nil {
				p.logger.Errorw("unmarshal cluster failed",
					zap.Error(err),
					zap.Any("resource", res),
				)
				return err
			}
			p.logger.Debugw("got cluster response",
				zap.Any("cluster", &cluster),
			)

			ups, err := p.TranslateCluster(&cluster)
			if err != nil {
				p.logger.Warnw("failed to translate Cluster to APISIX upstreams",
					zap.Error(err),
					zap.Any("cluster", res),
				)
				continue
			}
			if cluster.GetType() == clusterv3.Cluster_EDS {
				p.edsRequiredClusters.Add(cluster.Name)
			}
			newManifest.Upstreams = append(newManifest.Upstreams, ups)
			newUps[ups.Name] = ups
		}
		// TODO Refactor util.Manifest to just use map.
		for _, ups := range p.upstreams {
			oldManifest.Upstreams = append(oldManifest.Upstreams, ups)
		}
		p.upstreams = newUps
		if !p.edsRequiredClusters.Equals(oldEdsRequiredClusters) {
			p.logger.Infow("new EDS discovery request",
				zap.Any("old_eds_required_clusters", oldEdsRequiredClusters),
				zap.Any("eds_required_clusters", p.edsRequiredClusters),
			)
			p.sendEds(p.edsRequiredClusters)
		}
	case types.ClusterLoadAssignmentUrl:
		requireEds := util.StringSet{}

		for _, res := range resp.GetResources() {
			var cla endpointv3.ClusterLoadAssignment
			err := anypb.UnmarshalTo(res, &cla, proto.UnmarshalOptions{
				DiscardUnknown: true,
			})
			if err != nil {
				p.logger.Errorw("failed to unmarshal ClusterLoadAssignment",
					zap.Error(err),
					zap.Any("resource", res),
				)
				continue
			}

			p.logger.Debugw("got cluster load assignment response",
				zap.Any("cla", &cla),
			)

			ups, err := p.processClusterLoadAssignmentV3(&cla)
			if err == ErrorRequireFurtherEDS {
				// TODO process this
				ignoredClusterName := []string{
					"kubernetes.default.svc.cluster.local",
					"kube-system.svc.cluster.local",
					"istio-system.svc.cluster.local",
				}

				ignored := false
				for _, clusterName := range ignoredClusterName {
					if strings.Contains(cla.ClusterName, clusterName) {
						ignored = true
						break
					}
				}
				if !ignored {
					requireEds.Add(cla.ClusterName)
				}
				continue
			}
			if err != nil {
				return err
			}
			p.upstreams[ups.Name] = ups
			newManifest.Upstreams = append(newManifest.Upstreams, ups)
		}

		if len(requireEds) > 0 {
			p.logger.Infow("empty endpoint, new EDS discovery request",
				zap.Any("eds_required_clusters", requireEds),
			)
			p.sendEds(requireEds)
		}
	case types.ListenerUrl:
		var (
			rdsNames      []string
			staticConfigs []*routev3.RouteConfiguration
		)
		routeOwnership := make(map[string]string)
		for _, res := range resp.GetResources() {
			var listener listenerv3.Listener
			if err := anypb.UnmarshalTo(res, &listener, proto.UnmarshalOptions{}); err != nil {
				p.logger.Errorw("failed to unmarshal listener v3",
					zap.Error(err),
					zap.Any("response", res),
				)
				return err
			}

			p.logger.Debugw("got listener response",
				zap.Any("listener", &listener),
			)

			sockAddr := listener.Address.GetSocketAddress()
			if sockAddr == nil || sockAddr.GetPortValue() == 0 {
				// Only use listener which listens on socket.
				// TODO Support named port.
				continue
			}
			addr := fmt.Sprintf("%s:%d", sockAddr.GetAddress(), sockAddr.GetPortValue())
			names, cfgs, err := p.GetRoutesFromListener(&listener)
			if err != nil {
				return err
			}
			rdsNames = append(rdsNames, names...)
			staticConfigs = append(staticConfigs, cfgs...)
			for _, name := range names {
				routeOwnership[name] = addr
			}
			for _, cfg := range cfgs {
				routeOwnership[cfg.GetName()] = addr
			}
		}
		p.staticRouteConfigurations = staticConfigs
		p.routeOwnership = routeOwnership
		p.sendRds(rdsNames)
	default:
		p.logger.Debugw("got unsupported discovery response type",
			zap.String("type", resp.TypeUrl),
			zap.Any("body", resp),
		)
		return errors.New("UnknownResourceTypeUrl")
	}

	// Always generate update event for EDS.
	if resp.GetTypeUrl() == types.ClusterLoadAssignmentUrl {
		for _, ups := range newManifest.Upstreams {
			events = append(events, types.Event{
				Type:   types.EventUpdate,
				Key:    fmt.Sprintf("/upstreams/%s", ups.Id),
				Object: ups,
			})
		}
	} else {
		//events = newManifest.Events(types.EventAdd)
		events = p.generateIncrementalEvents(&newManifest, &oldManifest)
	}
	go func() {
		p.evChan <- events
	}()
	return nil
}

func (p *xdsProvisioner) sendEds(edsRequests util.StringSet) {
	// TODO: merge calls to reduce duplicate requests?
	dr := &discoveryv3.DiscoveryRequest{
		Node:          p.node,
		TypeUrl:       types.ClusterLoadAssignmentUrl,
		ResourceNames: edsRequests.Strings(),
	}
	p.logger.Debugw("sending EDS discovery request",
		zap.Any("body", dr),
	)
	p.sendCh <- dr
}

func (p *xdsProvisioner) sendRds(rdsNames []string) {
	if len(rdsNames) == 0 {
		return
	}
	dr := &discoveryv3.DiscoveryRequest{
		Node:          p.node,
		TypeUrl:       types.RouteConfigurationUrl,
		ResourceNames: rdsNames,
	}
	p.logger.Debugw("sending RDS discovery request",
		zap.Any("body", dr),
	)
	p.sendCh <- dr
}

func (p *xdsProvisioner) generateIncrementalEvents(m, o *util.Manifest) []types.Event {
	p.logger.Debugw("comparing old and new manifests",
		zap.Any("old", o),
		zap.Any("new", m),
	)
	var (
		added   *util.Manifest
		deleted *util.Manifest
		updated *util.Manifest
		count   int
	)
	if o == nil {
		added = m
	} else if m == nil {
		deleted = o
	} else {
		added, deleted, updated = o.DiffFrom(m)
	}
	if added != nil {
		count += added.Size()
	}
	if deleted != nil {
		count += deleted.Size()
	}
	if updated != nil {
		count += updated.Size()
	}
	if count == 0 {
		p.logger.Debugw("old and new manifests are exactly same")
		return nil
	}

	p.logger.Debugw("found changes (after converting to APISIX resources)",
		zap.Any("added", added),
		zap.Any("updated", updated),
		zap.Any("deleted", deleted),
	)

	events := make([]types.Event, 0, count)
	if added != nil {
		events = append(events, added.Events(types.EventAdd)...)
	}
	if deleted != nil {
		events = append(events, deleted.Events(types.EventDelete)...)
	}
	if updated != nil {
		events = append(events, updated.Events(types.EventUpdate)...)
	}
	return events
}
