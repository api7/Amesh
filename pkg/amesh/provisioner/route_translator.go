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
	"fmt"
	"reflect"
	"strings"

	"github.com/api7/gopkg/pkg/id"
	"github.com/api7/gopkg/pkg/log"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	commonfaultv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/common/fault/v3"
	faultv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	xdswellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/fatih/color"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	ameshv1alpha1 "github.com/api7/amesh/controller/apis/amesh/v1alpha1"
	"github.com/api7/amesh/pkg/amesh/types"
	"github.com/api7/amesh/pkg/amesh/util"
	"github.com/api7/amesh/pkg/apisix"
)

const (
	_httpConnectManagerV3 = "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"

	_httpFaultV3 = "type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault"

	_defaultRoutePriority = 999
)

func (p *xdsProvisioner) TranslateRouteConfiguration(r *routev3.RouteConfiguration, routeOriginalDest map[string]string) ([]*apisix.Route, error) {
	var routes []*apisix.Route
	for _, vhost := range r.GetVirtualHosts() {
		p.logger.Debugw(color.GreenString("process virtual host"),
			zap.Any("vhost", vhost),
		)

		partial, err := p.translateVirtualHost(r.Name, vhost)
		if err != nil {
			p.logger.Errorw("failed to translate VirtualHost",
				zap.Error(err),
			)
			return nil, err
		}
		routes = append(routes, partial...)
	}
	if routeOriginalDest != nil {
		origDst, ok := routeOriginalDest[r.Name]
		if ok {
			patchRoutesWithOriginalDestination(routes, origDst)
		}
	}

	p.logger.Debugw("got routes after parsing route config",
		zap.Any("routes", routes),
	)

	// TODO support Vhds.
	return routes, nil
}

// TODO: how to deal with partial failure and ACK response?
func (p *xdsProvisioner) translateVirtualHost(routeName string, vhost *routev3.VirtualHost) ([]*apisix.Route, error) {
	if routeName == "" {
		routeName = "anon"
	}

	hostSet := util.StringSet{}
	for _, domain := range vhost.Domains {
		if domain == "*" {
			// If this route allows any domain to use, just don't set hosts
			// in APISIX routes.
			hostSet = util.StringSet{}
			break
		} else {
			if pos := strings.Index(domain, ":"); pos != -1 {
				domain = domain[:pos]
			}
			hostSet.Add(domain)
		}
	}
	// avoid unstable array for diff
	hosts := hostSet.OrderedStrings()

	var errors error = nil
	var routes []*apisix.Route
	for _, route := range vhost.GetRoutes() {
		p.logger.Debugw(color.GreenString("process route"),
			zap.Any("route", route),
		)

		match := route.GetMatch()
		// TODO CaseSensitive field.
		sensitive := match.CaseSensitive
		if sensitive != nil && !sensitive.GetValue() {
			// Apache APISIX doesn't support case-insensitive URI match,
			// so these routes should be neglected.
			p.logger.Warnw("ignore route with case insensitive match",
				zap.Any("route", route),
			)
			continue
		}

		cluster, err := p.getClusterName(route)
		if err != nil {
			errors = multierror.Append(err)
			p.logger.Warnw("failed to get cluster name, skipped",
				zap.Error(err),
				zap.Any("route", route),
			)
			continue
		}
		uri, err := p.getURL(route)
		if err != nil {
			errors = multierror.Append(err)
			p.logger.Warnw("failed to get url from path specifier, skipped",
				zap.Error(err),
				zap.Any("route", route),
			)
			continue
		}

		name := route.Name
		if name == "" {
			name = "anon"
		}
		priority := _defaultRoutePriority
		if len(hosts) == 0 {
			priority = 0
		}
		// This is for istio.
		// use the default and lowest priority for the "allow_any" route.
		if name == "allow_any" {
			priority = 0
		}

		queryVars, err := p.getParametersMatchVars(route)
		if err != nil {
			errors = multierror.Append(err)
			p.logger.Warnw("failed to get parameter match variable, skipped",
				zap.Error(err),
				zap.Any("route", route),
			)
			continue
		}
		vars, err := p.getHeadersMatchVars(route)
		if err != nil {
			errors = multierror.Append(err)
			p.logger.Warnw("failed to get header match variable, skipped",
				zap.Error(err),
				zap.Any("route", route),
			)
			continue
		}
		vars = append(vars, queryVars...)
		name = fmt.Sprintf("%s#%s#%s", name, vhost.GetName(), routeName)
		name = strings.Replace(name, ".svc.cluster.local", "", -1) // avoid name too long

		set := util.StringSet{}
		for _, v := range vars {
			set.Add(v.ToComparableString())
		}
		condStr := strings.Join(set.OrderedStrings(), "-")

		// all matching conditions should be considered in name generation, since routes may have same name
		matchConditions := fmt.Sprintf("%v-%v-%v", uri, sensitive, condStr)

		r := &apisix.Route{
			Name:       name,
			Priority:   int32(priority),
			Status:     1,
			Id:         id.GenID(name) + "_" + id.GenID(matchConditions),
			Hosts:      hosts,
			Uris:       []string{uri},
			UpstreamId: id.GenID(cluster),
			Vars:       vars,
			Plugins:    map[string]interface{}{},
			Desc:       fmt.Sprintf("GENERATED_BY_AMESH: VIRTUAL_HOST: %v, NAME: %v, CLUSTER: %v", vhost.Name, route.Name, cluster),
		}

		timeoutSettings := route.GetRoute().GetTimeout()
		if timeoutSettings != nil && timeoutSettings.Seconds > 0 {
			timeout := timeoutSettings.Seconds
			r.Timeout = &apisix.Timeout{
				Connect: float64(timeout),
				Send:    float64(timeout),
				Read:    float64(timeout),
			}
		}
		r = p.patchAmeshPlugins(r)

		//p.logger.Warnw("pre filter route",
		//	zap.Any("route", route),
		//	zap.Any("apisix_route", r),
		//)
		err = p.translateRouteFilters(route, r)
		if err != nil {
			errors = multierror.Append(err)
			p.logger.Warnw("failed to translate route filters, skipped",
				zap.Error(err),
				zap.Any("route", route),
			)
			continue
		}

		r.Plugins[apisix.PluginPrometheus] = &apisix.Prometheus{
			PreferName: true,
		}

		err = p.translateRoutePlugins(route, r)
		if err != nil {
			errors = multierror.Append(err)
			p.logger.Warnw("failed to translate route plugins, skipped",
				zap.Error(err),
				zap.Any("route", route),
			)
			continue
		}

		routes = append(routes, r)
	}
	return routes, errors
}

func (p *xdsProvisioner) patchAmeshPlugins(route *apisix.Route) *apisix.Route {
	//route.Plugins = map[string]interface{}{}
	ameshPlugins := p.amesh.GetPlugins()

	for pluginName, _ := range route.Plugins {
		if _, ok := ameshPlugins[pluginName]; !ok {
			// Delete event
			delete(route.Plugins, pluginName)
		}
	}

	preReq := types.ApisixExtPluginConfig{}
	postReq := types.ApisixExtPluginConfig{}
	// TODO: reuse plugin configs?
	for _, plugin := range ameshPlugins {
		switch plugin.Type {
		case "":
			route.Plugins[plugin.Name] = plugin.Config
			p.logger.Infow("patched plugin", zap.Any("config", plugin.Config))
		case ameshv1alpha1.AmeshPluginConfigTypePreRequest:
			preReq.Conf = append(preReq.Conf, &types.ApisixExtPlugin{
				Name:  plugin.Name,
				Value: plugin.Config,
			})
		case ameshv1alpha1.AmeshPluginConfigTypePostRequest:
			postReq.Conf = append(postReq.Conf, &types.ApisixExtPlugin{
				Name:  plugin.Name,
				Value: plugin.Config,
			})
		}
	}
	if len(preReq.Conf) > 0 {
		route.Plugins["ext-plugin-pre-req"] = preReq
	}
	if len(postReq.Conf) > 0 {
		route.Plugins["ext-plugin-post-req"] = postReq
	}
	return route
}

func (p *xdsProvisioner) getClusterName(xdsRoute *routev3.Route) (string, error) {
	route := xdsRoute.GetRoute()
	if route == nil {
		return "", fmt.Errorf("unsupported route action type %T", xdsRoute.GetAction())
	}

	clusterSpecifier := route.GetClusterSpecifier()
	switch v := clusterSpecifier.(type) {
	case *routev3.RouteAction_Cluster:
		return v.Cluster, nil
	case *routev3.RouteAction_WeightedClusters:
		clusters := v.WeightedClusters.Clusters
		// pick last cluster as default upstream
		return clusters[len(clusters)-1].Name, nil
	default:
		return "", fmt.Errorf("unsupported cluster specifier type %T", route.GetClusterSpecifier())
	}
}

func (p *xdsProvisioner) getURL(route *routev3.Route) (string, error) {
	var uri string
	path := route.GetMatch().GetPathSpecifier()
	switch path.(type) {
	case *routev3.RouteMatch_Path:
		uri = path.(*routev3.RouteMatch_Path).Path
	case *routev3.RouteMatch_Prefix:
		uri = path.(*routev3.RouteMatch_Prefix).Prefix + "*"
	default:
		return "", fmt.Errorf("unsupported path specifier type %T", path)
	}
	return uri, nil
}

func (p *xdsProvisioner) getParametersMatchVars(route *routev3.Route) ([]*apisix.Var, error) {
	// See https://github.com/api7/lua-resty-expr
	// for the translation details.
	var vars []*apisix.Var
	for _, param := range route.GetMatch().GetQueryParameters() {
		var expr apisix.Var
		name := "arg_" + param.GetName()
		switch param.GetQueryParameterMatchSpecifier().(type) {
		case *routev3.QueryParameterMatcher_PresentMatch:
			expr = apisix.Var{name, "!", "~~", "^$"}
		case *routev3.QueryParameterMatcher_StringMatch:
			matcher := param.GetQueryParameterMatchSpecifier().(*routev3.QueryParameterMatcher_StringMatch)
			value, err := getStringMatchValue(matcher.StringMatch)
			if err != nil {
				return nil, err
			}
			op := "~~"
			if matcher.StringMatch.IgnoreCase {
				op = "~*"
			}
			expr = apisix.Var{name, op, value}
		}
		vars = append(vars, &expr)
	}
	return vars, nil
}

func (p *xdsProvisioner) getHeadersMatchVars(route *routev3.Route) ([]*apisix.Var, error) {
	// See https://github.com/api7/lua-resty-expr
	// for the translation details.
	var vars []*apisix.Var
	for _, header := range route.GetMatch().GetHeaders() {
		var (
			expr  apisix.Var
			name  string
			value string
		)
		switch header.GetName() {
		case ":method":
			name = "request_method"
		case ":authority":
			name = "http_host"
		default:
			name = strings.ToLower(header.Name)
			name = "http_" + strings.ReplaceAll(name, "-", "_")
		}

		switch v := header.HeaderMatchSpecifier.(type) {
		case *routev3.HeaderMatcher_StringMatch:
			var err error
			value, err = getStringMatchValue(v.StringMatch)
			if err != nil {
				return nil, err
			}
		case *routev3.HeaderMatcher_ContainsMatch:
			value = v.ContainsMatch
		case *routev3.HeaderMatcher_ExactMatch:
			value = "^" + v.ExactMatch + "$"
		case *routev3.HeaderMatcher_PrefixMatch:
			value = "^" + v.PrefixMatch
		case *routev3.HeaderMatcher_PresentMatch:
		case *routev3.HeaderMatcher_SafeRegexMatch:
			value = v.SafeRegexMatch.Regex
		case *routev3.HeaderMatcher_SuffixMatch:
			value = v.SuffixMatch + "$"
		default:
			// TODO Some other HeaderMatchers can be implemented else.
			p.logger.Warnw("ignore route with unexpected header matcher",
				zap.Any("matcher", v),
				zap.Any("route", route),
			)
			return nil, fmt.Errorf("unexpected header matcher type %T", v)
		}

		if header.InvertMatch {
			expr = apisix.Var{name, "!", "~~", value}
		} else {
			expr = apisix.Var{name, "~~", value}
		}
		vars = append(vars, &expr)
	}
	return vars, nil
}

func getStringMatchValue(matcher *matcherv3.StringMatcher) (string, error) {
	// TODO support case sensitive options
	//insensitive := matcher.IgnoreCase

	pattern := matcher.MatchPattern
	switch pat := pattern.(type) {
	case *matcherv3.StringMatcher_Exact:
		return "^" + pat.Exact + "$", nil
	case *matcherv3.StringMatcher_Contains:
		return pat.Contains, nil
	case *matcherv3.StringMatcher_Prefix:
		return "^" + pat.Prefix, nil
	case *matcherv3.StringMatcher_Suffix:
		return pat.Suffix + "$", nil
	case *matcherv3.StringMatcher_SafeRegex:
		// TODO Regex Engine detection.
		return pat.SafeRegex.Regex, nil
	default:
		return "", fmt.Errorf("unknown StringMatcher type %T", pattern)
	}
}

func patchRoutesWithOriginalDestination(routes []*apisix.Route, origDst string) {
	// TODO: apply this
	//if strings.HasPrefix(origDst, "0.0.0.0:") {
	//	port := origDst[len("0.0.0.0:"):]
	//	for _, r := range routes {
	//		r.Vars = append(r.Vars, &apisix.Var{"connection_original_dst", "~~", port + "$"})
	//	}
	//} else {
	//	for _, r := range routes {
	//		r.Vars = append(r.Vars, &apisix.Var{"connection_original_dst", "==", origDst})
	//	}
	//}
}

func (p *xdsProvisioner) convertPercent(percent *typev3.FractionalPercent) uint32 {
	if percent == nil {
		return 0
	}

	var percentage uint32
	switch percent.Denominator {
	case typev3.FractionalPercent_MILLION:
		percentage = percent.Numerator / 10000
	case typev3.FractionalPercent_TEN_THOUSAND:
		percentage = percent.Numerator / 100
	case typev3.FractionalPercent_HUNDRED:
		percentage = percent.Numerator
	}
	if percentage > 100 {
		percentage = 100
	}
	return percentage
}

func (p *xdsProvisioner) convertRuntimePercent(percent *corev3.RuntimeFractionalPercent) uint32 {
	if percent == nil {
		return 0
	}

	v := percent.GetDefaultValue()
	return p.convertPercent(v)
}

func (p *xdsProvisioner) translateRouteFilters(xdsRoute *routev3.Route, apisixRoute *apisix.Route) error {
	for filterName, data := range xdsRoute.TypedPerFilterConfig {
		p.logger.Debugw(color.GreenString("process route filter"),
			zap.Any("filter_type", filterName),
			zap.Any("config", data),
		)

		switch filterName {
		case xdswellknown.Fault:
			if data.GetTypeUrl() == _httpFaultV3 {
				log.Infof("got route http.fault filter")

				var fault faultv3.HTTPFault
				if err := anypb.UnmarshalTo(data, &fault, proto.UnmarshalOptions{}); err != nil {
					log.Errorw("failed to unmarshal HTTPFault config",
						zap.Error(err),
						zap.Any("route", xdsRoute.Name),
					)
					continue
				}

				faultPlugin := &apisix.FaultInjection{}

				// abort
				if fault.Abort != nil {
					faultPlugin.Abort = &apisix.FaultInjectionAbort{}
					// status
					switch v := fault.Abort.ErrorType.(type) {
					case *faultv3.FaultAbort_HttpStatus:
						faultPlugin.Abort.HttpStatus = v.HttpStatus
					default:
						// TODO: other types
						p.logger.Warnw("unsupported HTTPFault error type",
							zap.String("typed_url", data.GetTypeUrl()),
							zap.Any("config", data),
							zap.Any("type", reflect.TypeOf(v)),
						)
						continue
					}

					faultPlugin.Abort.Percentage = p.convertPercent(fault.Abort.Percentage)
				}

				// delay
				if fault.Delay != nil {
					faultPlugin.Delay = &apisix.FaultInjectionDelay{}

					switch v := fault.Delay.FaultDelaySecifier.(type) {
					case *commonfaultv3.FaultDelay_FixedDelay:
						faultPlugin.Delay.Duration = v.FixedDelay.Seconds
					default:
						// TODO: other types
						p.logger.Warnw("unsupported HTTPFault error type",
							zap.String("typed_url", data.GetTypeUrl()),
							zap.Any("config", data),
							zap.Any("type", reflect.TypeOf(v)),
						)
						continue
					}

					faultPlugin.Delay.Percentage = p.convertPercent(fault.Delay.Percentage)
				}

				apisixRoute.Plugins[apisix.PluginFaultInjection] = faultPlugin
			} else {
				p.logger.Warnw("unsupported HTTPFault version",
					zap.String("typed_url", data.GetTypeUrl()),
					zap.Any("config", data),
				)
			}
			break
		default:
			p.logger.Warnw("unsupported http filter",
				zap.String("name", filterName),
				zap.Any("config", data),
			)
		}
	}

	return nil
}

func (p *xdsProvisioner) translateRoutePlugins(xdsRoute *routev3.Route, apisixRoute *apisix.Route) error {
	route := xdsRoute.GetRoute()
	if route == nil {
		return nil
	}

	err := p.translateRouteWeightedClusters(route, apisixRoute)
	if err != nil {
		return err
	}

	//err = p.translateRouteMirror(route, apisixRoute)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (p *xdsProvisioner) translateRouteWeightedClusters(route *routev3.RouteAction, apisixRoute *apisix.Route) error {
	weightedClusters := route.GetWeightedClusters()
	if weightedClusters == nil {
		return nil
	}

	clusters := weightedClusters.Clusters
	weighted := make([]*apisix.TrafficSplitWeightedUpstreams, len(clusters))
	for i, weightedCluster := range clusters {
		p.logger.Debugw("translating weighted cluster",
			zap.Any("cluster", weightedCluster),
			zap.Any("index", i),
		)
		weighted[i] = &apisix.TrafficSplitWeightedUpstreams{
			Weight: weightedCluster.Weight.Value,
		}
		if i != len(clusters)-1 {
			// last one is default upstream
			weighted[i].UpstreamId = id.GenID(weightedCluster.Name)
		}
	}

	apisixRoute.Plugins[apisix.PluginTrafficSplit] = &apisix.TrafficSplit{
		Rules: []*apisix.TrafficSplitRule{
			{
				WeightedUpstreams: weighted,
			},
		},
	}

	return nil
}

func (p *xdsProvisioner) translateRouteMirror(route *routev3.RouteAction, apisixRoute *apisix.Route) error {
	mirrors := route.GetRequestMirrorPolicies()
	if len(mirrors) == 0 {
		return nil
	}
	// TODO: how proxy-mirror works with traffic-split?
	mirror := mirrors[0]
	cluster := mirror.Cluster
	percent := p.convertRuntimePercent(mirror.RuntimeFraction)
	if percent <= 0 {
		percent = 100 // defaults to 100%
	}
	percentage := float32(percent) / 100

	p.upstreamsLock.RLock()
	upstream, ok := p.upstreams[cluster]
	p.upstreamsLock.RUnlock()

	if !ok || len(upstream.Nodes) == 0 {
		// enqueue retry queue
		apisixRoute.Plugins[apisix.PluginProxyMirror] = &apisix.ProxyMirror{
			Host:        id.GenID(cluster), // patched needed
			Path:        "",                // leave empty
			SampleRatio: percentage,
		}

		return nil
	}

	host := string(upstream.Scheme) + "://" + upstream.Nodes[0].Host

	apisixRoute.Plugins[apisix.PluginProxyMirror] = &apisix.ProxyMirror{
		//Host:        host,
		Host:        host,
		Path:        "", // leave empty
		SampleRatio: percentage,
	}

	return nil
}

func (p *xdsProvisioner) GetRoutesFromListener(l *listenerv3.Listener) ([]string, []*routev3.RouteConfiguration, error) {
	var (
		rdsNames      []string
		staticConfigs []*routev3.RouteConfiguration
	)

	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			// network filters
			switch f.Name {
			case xdswellknown.HTTPConnectionManager:
				if f.GetTypedConfig().GetTypeUrl() == _httpConnectManagerV3 {
					var hcm hcmv3.HttpConnectionManager
					if err := anypb.UnmarshalTo(f.GetTypedConfig(), &hcm, proto.UnmarshalOptions{}); err != nil {
						log.Errorw("failed to unmarshal HttpConnectionManager config",
							zap.Error(err),
							zap.Any("listener", l),
						)
						return nil, nil, err
					}
					if hcm.GetRds() != nil {
						rdsNames = append(rdsNames, hcm.GetRds().GetRouteConfigName())
					} else if hcm.GetRouteConfig() != nil {
						// TODO deep copy?
						staticConfigs = append(staticConfigs, hcm.GetRouteConfig())
					} else if hcm.GetScopedRoutes() != nil {
						p.logger.Warnw("unsupported ScopedRoutes config",
							zap.String("typed", f.GetTypedConfig().GetTypeUrl()),
							zap.Any("config", f.GetTypedConfig()),
						)
					}
				} else {
					p.logger.Warnw("unsupported HTTPConnectManager version",
						zap.String("typed_url", f.GetTypedConfig().GetTypeUrl()),
						zap.Any("config", f.GetTypedConfig()),
					)
				}
				break
			case xdswellknown.TCPProxy:
				p.logger.Warnw("unsupported tcp proxy filter",
					zap.String("name", f.Name),
					zap.Any("config", f.GetTypedConfig()),
				)
				break
			case xdswellknown.RateLimit:
				p.logger.Warnw("unsupported rate limit filter",
					zap.String("name", f.Name),
					zap.Any("config", f.GetTypedConfig()),
				)
				break
			default:
				p.logger.Warnw("unsupported network filter",
					zap.String("name", f.Name),
					zap.Any("config", f.GetTypedConfig()),
				)
			}
		}
	}
	p.logger.Debugw("got route names and config from listener",
		zap.Strings("route_names", rdsNames),
		zap.Any("route_configs", staticConfigs),
		//zap.Any("listener", l),
	)
	return rdsNames, staticConfigs, nil
}
