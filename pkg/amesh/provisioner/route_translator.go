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
	"fmt"
	"strings"

	"github.com/api7/gopkg/pkg/id"
	"github.com/api7/gopkg/pkg/log"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	xdswellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/api7/amesh/pkg/amesh/util"
	"github.com/api7/amesh/pkg/apisix"
)

const (
	_httpConnectManagerV3 = "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
	_defaultRoutePriority = 999
)

func (p *xdsProvisioner) TranslateRouteConfiguration(r *routev3.RouteConfiguration, routeOriginalDest map[string]string) ([]*apisix.Route, error) {
	var routes []*apisix.Route
	for _, vhost := range r.GetVirtualHosts() {
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

func (p *xdsProvisioner) translateVirtualHost(prefix string, vhost *routev3.VirtualHost) ([]*apisix.Route, error) {
	if prefix == "" {
		prefix = "<anon>"
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

	var routes []*apisix.Route
	for _, route := range vhost.GetRoutes() {
		// TODO CaseSensitive field.
		sensitive := route.GetMatch().CaseSensitive
		if sensitive != nil && !sensitive.GetValue() {
			// Apache APISIX doesn't support case insensitive URI match,
			// so these routes should be neglected.
			p.logger.Warnw("ignore route with case insensitive match",
				zap.Any("route", route),
			)
			continue
		}

		cluster, skip := p.getClusterName(route)
		if skip {
			continue
		}
		uri, skip := p.getURL(route)
		if skip {
			continue
		}

		name := route.Name
		if name == "" {
			name = "<anon>"
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

		queryVars, skip := p.getParametersMatchVars(route)
		if skip {
			continue
		}
		vars, skip := p.getHeadersMatchVars(route)
		if skip {
			continue
		}
		vars = append(vars, queryVars...)
		name = fmt.Sprintf("%s#%s#%s", name, vhost.GetName(), prefix)
		name = strings.Replace(name, ".svc.cluster.local", "", -1) // avoid name too long
		r := &apisix.Route{
			Name:       name,
			Priority:   int32(priority),
			Status:     1,
			Id:         id.GenID(name),
			Hosts:      hosts,
			Uris:       []string{uri},
			UpstreamId: id.GenID(cluster),
			Vars:       vars,
			Desc:       "GENERATED_BY_AMESH: VIRTUAL_HOST: " + vhost.Name,
		}
		routes = append(routes, r)
	}
	return routes, nil
}

func (p *xdsProvisioner) getClusterName(route *routev3.Route) (string, bool) {
	action, ok := route.GetAction().(*routev3.Route_Route)
	if !ok {
		p.logger.Warnw("ignore route with unexpected action type",
			zap.Any("route", route),
			zap.Any("action", route.GetAction()),
		)
		return "", true
	}
	cluster, ok := action.Route.GetClusterSpecifier().(*routev3.RouteAction_Cluster)
	if !ok {
		p.logger.Warnw("ignore route with unexpected cluster specifier",
			zap.Any("route", route),
		)
		return "", true
	}
	return cluster.Cluster, false
}

func (p *xdsProvisioner) getURL(route *routev3.Route) (string, bool) {
	var uri string
	switch route.GetMatch().GetPathSpecifier().(type) {
	case *routev3.RouteMatch_Path:
		uri = route.GetMatch().GetPathSpecifier().(*routev3.RouteMatch_Path).Path
	case *routev3.RouteMatch_Prefix:
		uri = route.GetMatch().GetPathSpecifier().(*routev3.RouteMatch_Prefix).Prefix + "*"
	default:
		p.logger.Warnw("ignore route with unexpected path specifier",
			zap.Any("route", route),
			zap.Any("type", route.GetMatch().GetPathSpecifier()),
		)
		return "", true
	}
	return uri, false
}

func (p *xdsProvisioner) getParametersMatchVars(route *routev3.Route) ([]*apisix.Var, bool) {
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
			value := getStringMatchValue(matcher.StringMatch)
			op := "~~"
			if matcher.StringMatch.IgnoreCase {
				op = "~*"
			}
			expr = apisix.Var{name, op, value}
		}
		vars = append(vars, &expr)
	}
	return vars, false
}

func (p *xdsProvisioner) getHeadersMatchVars(route *routev3.Route) ([]*apisix.Var, bool) {
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

		switch header.HeaderMatchSpecifier.(type) {
		case *routev3.HeaderMatcher_ContainsMatch:
			value = header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_ContainsMatch).ContainsMatch
		case *routev3.HeaderMatcher_ExactMatch:
			value = "^" + header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_ExactMatch).ExactMatch + "$"
		case *routev3.HeaderMatcher_PrefixMatch:
			value = "^" + header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_PrefixMatch).PrefixMatch
		case *routev3.HeaderMatcher_PresentMatch:
		case *routev3.HeaderMatcher_SafeRegexMatch:
			value = header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_SafeRegexMatch).SafeRegexMatch.Regex
		case *routev3.HeaderMatcher_SuffixMatch:
			value = header.HeaderMatchSpecifier.(*routev3.HeaderMatcher_SuffixMatch).SuffixMatch + "$"
		default:
			// TODO Some other HeaderMatchers can be implemented else.
			p.logger.Warnw("ignore route with unexpected header matcher",
				zap.Any("route", route),
			)
			return nil, true
		}

		if header.InvertMatch {
			expr = apisix.Var{name, "!", "~~", value}
		} else {
			expr = apisix.Var{name, "~~", value}
		}
		vars = append(vars, &expr)
	}
	return vars, false
}

func getStringMatchValue(matcher *matcherv3.StringMatcher) string {
	pattern := matcher.MatchPattern
	switch pat := pattern.(type) {
	case *matcherv3.StringMatcher_Exact:
		return "^" + pat.Exact + "$"
	case *matcherv3.StringMatcher_Contains:
		return pat.Contains
	case *matcherv3.StringMatcher_Prefix:
		return "^" + pat.Prefix
	case *matcherv3.StringMatcher_Suffix:
		return pat.Suffix + "$"
	case *matcherv3.StringMatcher_SafeRegex:
		// TODO Regex Engine detection.
		return pat.SafeRegex.Regex
	default:
		panic("unknown StringMatcher type")
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

func (p *xdsProvisioner) GetRoutesFromListener(l *listenerv3.Listener) ([]string, []*routev3.RouteConfiguration, error) {
	var (
		rdsNames      []string
		staticConfigs []*routev3.RouteConfiguration
	)

	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name == xdswellknown.HTTPConnectionManager {
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
						zap.String("typed", f.GetTypedConfig().GetTypeUrl()),
						zap.Any("config", f.GetTypedConfig()),
					)
				}
			} else {
				switch f.Name {
				case xdswellknown.TCPProxy:
				case xdswellknown.RateLimit:
					p.logger.Debugw("unsupported filter",
						zap.String("name", f.Name),
						zap.Any("config", f.GetTypedConfig()),
					)
				default:
					p.logger.Warnw("unsupported filter",
						zap.String("name", f.Name),
						zap.Any("config", f.GetTypedConfig()),
					)
				}
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
