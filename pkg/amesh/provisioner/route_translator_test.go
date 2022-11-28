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

package provisioner

import (
	"sort"
	"strings"
	"testing"

	"github.com/api7/gopkg/pkg/id"
	"github.com/api7/gopkg/pkg/log"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	faultv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/fault/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	xdswellknown "github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/api7/amesh/pkg/apisix"
	"github.com/api7/amesh/pkg/apisix/utils"
)

func TestGetStringMatchValue(t *testing.T) {
	getStringMatchValue := func(matcher *matcherv3.StringMatcher) string {
		val, _ := getStringMatchValue(matcher)
		return val
	}

	matcher := &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Exact{
			Exact: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "^Hangzhou$", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Contains{
			Contains: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "Hangzhou", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Prefix{
			Prefix: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "^Hangzhou", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_Suffix{
			Suffix: "Hangzhou",
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), "Hangzhou$", "translating exact string match")

	matcher = &matcherv3.StringMatcher{
		MatchPattern: &matcherv3.StringMatcher_SafeRegex{
			SafeRegex: &matcherv3.RegexMatcher{
				Regex: ".*\\d+Hangzhou",
			},
		},
	}
	assert.Equal(t, getStringMatchValue(matcher), ".*\\d+Hangzhou", "translating exact string match")
}

func TestGetHeadersMatchVars(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}

	route := &routev3.Route{
		Match: &routev3.RouteMatch{
			Headers: []*routev3.HeaderMatcher{
				{
					Name: ":method",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
						ContainsMatch: "POST",
					},
				},
				{
					Name: ":authority",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
						ExactMatch: "apisix.apache.org",
					},
				},
				{
					Name: "Accept-Ranges",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_PrefixMatch{
						PrefixMatch: "bytes",
					},
					InvertMatch: true,
				},
				{
					Name: "Content-Type",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_SafeRegexMatch{
						SafeRegexMatch: &matcherv3.RegexMatcher{
							Regex: `\.(jpg|png|gif)`,
						},
					},
				},
				{
					Name: "Content-Encoding",
					HeaderMatchSpecifier: &routev3.HeaderMatcher_SuffixMatch{
						SuffixMatch: "zip",
					},
				},
			},
		},
	}
	vars, err := a.getHeadersMatchVars(route)
	assert.Equal(t, err, nil)
	assert.Len(t, vars, len(route.Match.Headers))
	assert.Equal(t, vars[0], &apisix.Var{"request_method", "~~", "POST"})
	assert.Equal(t, vars[1], &apisix.Var{"http_host", "~~", "^apisix.apache.org$"})
	assert.Equal(t, vars[2], &apisix.Var{"http_accept_ranges", "!", "~~", "^bytes"})
	assert.Equal(t, vars[3], &apisix.Var{"http_content_type", "~~", `\.(jpg|png|gif)`})
	assert.Equal(t, vars[4], &apisix.Var{"http_content_encoding", "~~", "zip$"})
}

func TestGetParametersMatchVars(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}

	route := &routev3.Route{
		Match: &routev3.RouteMatch{
			QueryParameters: []*routev3.QueryParameterMatcher{
				{
					Name: "man",
					QueryParameterMatchSpecifier: &routev3.QueryParameterMatcher_PresentMatch{
						PresentMatch: true,
					},
				},
				{
					Name: "id",
					QueryParameterMatchSpecifier: &routev3.QueryParameterMatcher_StringMatch{
						StringMatch: &matcherv3.StringMatcher{
							MatchPattern: &matcherv3.StringMatcher_Exact{
								Exact: "123456",
							},
						},
					},
				},
				{
					Name: "name",
					QueryParameterMatchSpecifier: &routev3.QueryParameterMatcher_StringMatch{
						StringMatch: &matcherv3.StringMatcher{
							MatchPattern: &matcherv3.StringMatcher_Contains{
								Contains: "alex",
							},
							IgnoreCase: true,
						},
					},
				},
			},
		},
	}

	vars, err := a.getParametersMatchVars(route)
	assert.Equal(t, err, nil)
	assert.Len(t, vars, 3)
	assert.Equal(t, vars[0], &apisix.Var{"arg_man", "!", "~~", "^$"})
	assert.Equal(t, vars[1], &apisix.Var{"arg_id", "~~", "^123456$"})
	assert.Equal(t, vars[2], &apisix.Var{"arg_name", "~*", "alex"})
}

func TestGetURL(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}
	route := &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: "/foo/baz",
			},
		},
	}
	uri, err := a.getURL(route)
	assert.Nil(t, err)
	assert.Equal(t, uri, "/foo/baz*")

	route = &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Path{
				Path: "/foo/baz",
			},
		},
	}
	uri, err = a.getURL(route)
	assert.Nil(t, err)
	assert.Equal(t, uri, "/foo/baz")

	route = &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_SafeRegex{
				SafeRegex: &matcherv3.RegexMatcher{
					Regex: "/foo/.*?",
				},
			},
		},
	}
	_, err = a.getURL(route)
	assert.NotNil(t, err)
}

func TestGetClusterName(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,
	}
	route := &routev3.Route{
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_Cluster{
					Cluster: "kubernetes.default.svc.cluster.local",
				},
			},
		},
	}
	clusterName, err := a.getClusterName(route)
	assert.Nil(t, err)
	assert.Equal(t, clusterName, "kubernetes.default.svc.cluster.local")

	route = &routev3.Route{
		Action: &routev3.Route_Redirect{},
	}
	_, err = a.getClusterName(route)
	assert.NotNil(t, err)

	route = &routev3.Route{
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_ClusterHeader{},
			},
		},
	}
	_, err = a.getClusterName(route)
	assert.NotNil(t, err)
}

func toAnyPb(msg proto.Message) *anypb.Any {
	var a anypb.Any
	err := anypb.MarshalFrom(&a, msg, proto.MarshalOptions{})
	if err != nil {
		panic(err)
	}
	return &a
}

func TestTranslateVirtualHost(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,

		amesh: &ameshProvisioner{
			config: nil,
		},
	}
	vhost := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"apisix.apache.org",
			"*.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route1",
				Match: &routev3.RouteMatch{
					CaseSensitive: &wrappers.BoolValue{
						Value: true,
					},
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Prefix{
						Prefix: "/foo/baz",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
			},
			{
				Name: "route2",
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: "User",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
								StringMatch: &matcherv3.StringMatcher{
									MatchPattern: &matcherv3.StringMatcher_Contains{
										Contains: "abort",
									},
								},
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						// No cluster name, skipped
						ClusterSpecifier: &routev3.RouteAction_ClusterHeader{},
					},
				},
			},
			{
				Name: "route3",
				Match: &routev3.RouteMatch{
					CaseSensitive: &wrappers.BoolValue{
						Value: false,
					},
				},
			},
		},
	}
	routes, err := a.translateVirtualHost("test", vhost)
	assert.Nil(t, err)
	assert.Len(t, routes, 1)
	assert.Equal(t, "route1#test#test", routes[0].Name)
	assert.Equal(t, apisix.RouteEnable, routes[0].Status)
	assert.True(t, strings.Contains(routes[0].Id, id.GenID(routes[0].Name)))

	sort.Strings(routes[0].Hosts)
	assert.Equal(t, []string{
		"*.apache.org",
		"apisix.apache.org",
	}, routes[0].Hosts)
	assert.Equal(t, []string{
		"/foo/baz*",
	}, routes[0].Uris)
	assert.Equal(t, id.GenID("kubernetes.default.svc.cluster.local"), routes[0].UpstreamId)
	assert.Equal(t, []*apisix.Var{
		{"request_method", "~~", "POST"},
	}, routes[0].Vars)
}

func TestTranslateVirtualHostWithFilter(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,

		amesh: &ameshProvisioner{
			config: nil,
		},
	}
	vhost := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"apisix.apache.org",
			"*.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route",
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: "User",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_StringMatch{
								StringMatch: &matcherv3.StringMatcher{
									MatchPattern: &matcherv3.StringMatcher_Contains{
										Contains: "abort",
									},
								},
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
			},
			{
				Name: "route", // Same name, but has different matching conditions
				Match: &routev3.RouteMatch{
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
				TypedPerFilterConfig: map[string]*any.Any{
					xdswellknown.Fault: toAnyPb(&faultv3.HTTPFault{
						Abort: &faultv3.FaultAbort{
							ErrorType: &faultv3.FaultAbort_HttpStatus{
								HttpStatus: 555,
							},
							Percentage: &typev3.FractionalPercent{
								Numerator:   100,
								Denominator: 1,
							},
						},
					}),
				},
			},
		},
	}
	routes, err := a.translateVirtualHost("test", vhost)
	assert.Nil(t, err)
	assert.Len(t, routes, 2)
	assert.Equal(t, "route#test#test", routes[0].Name)
	assert.Equal(t, apisix.RouteEnable, routes[0].Status)

	assert.True(t, strings.Contains(routes[0].Id, id.GenID(routes[0].Name)))
	assert.True(t, strings.Contains(routes[1].Id, id.GenID(routes[0].Name)))
	assert.True(t, routes[0].Id != routes[1].Id)

	sort.Strings(routes[0].Hosts)
	assert.Equal(t, []string{
		"*.apache.org",
		"apisix.apache.org",
	}, routes[0].Hosts)
	assert.Equal(t, []string{
		"/foo/baz",
	}, routes[0].Uris)
	assert.Equal(t, id.GenID("kubernetes.default.svc.cluster.local"), routes[0].UpstreamId)
	assert.Equal(t, []*apisix.Var{
		{"http_user", "~~", "abort"},
	}, routes[0].Vars)
	assert.Equal(t, 0, len(routes[1].Vars))

	assert.Equal(t, 0, len(routes[0].Plugins))
	assert.Equal(t, 1, len(routes[1].Plugins))
	assert.NotNil(t, routes[1].Plugins["fault-injection"].(*apisix.FaultInjection))
	assert.NotNil(t, routes[1].Plugins["fault-injection"].(*apisix.FaultInjection).Abort)
	assert.Equal(t, uint32(100), routes[1].Plugins["fault-injection"].(*apisix.FaultInjection).Abort.Percentage)
	assert.Equal(t, uint32(555), routes[1].Plugins["fault-injection"].(*apisix.FaultInjection).Abort.HttpStatus)

	routes[0].Id = ""
	routes[1].Id = ""
	routes[0].Plugins = nil
	routes[1].Plugins = nil
	routes[0].Vars = nil
	routes[1].Vars = nil
	assert.Equal(t, routes[0], routes[1])
}

func TestPatchRoutesWithOriginalDestination(t *testing.T) {
	routes := []*apisix.Route{
		{
			Name: "1",
			Id:   "1",
		},
	}
	patchRoutesWithOriginalDestination(routes, "10.0.5.4:8080")
	// []*apisix.Var{
	//		{
	//			"connection_original_dst",
	//			"==",
	//			"10.0.5.4:8080",
	//		},
	//	}
	assert.Equal(t, routes[0].Vars, []*apisix.Var(nil))
}

func TestUnstableHostsRouteDiff(t *testing.T) {
	a := &xdsProvisioner{
		logger: log.DefaultLogger,

		amesh: &ameshProvisioner{
			config: nil,
		},
	}
	vhost1 := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"a.apisix.apache.org",
			"b.apisix.apache.org",
			"c.apisix.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route",
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
			},
		},
	}
	vhost2 := &routev3.VirtualHost{
		Name: "test",
		Domains: []string{
			"c.apisix.apache.org",
			"a.apisix.apache.org",
			"b.apisix.apache.org",
		},
		Routes: []*routev3.Route{
			{
				Name: "route",
				Action: &routev3.Route_Route{
					Route: &routev3.RouteAction{
						ClusterSpecifier: &routev3.RouteAction_Cluster{
							Cluster: "kubernetes.default.svc.cluster.local",
						},
					},
				},
				Match: &routev3.RouteMatch{
					Headers: []*routev3.HeaderMatcher{
						{
							Name: ":method",
							HeaderMatchSpecifier: &routev3.HeaderMatcher_ContainsMatch{
								ContainsMatch: "POST",
							},
						},
					},
					PathSpecifier: &routev3.RouteMatch_Path{
						Path: "/foo/baz",
					},
				},
			},
		},
	}
	routes1, err := a.translateVirtualHost("test", vhost1)
	assert.Nil(t, err)
	routes2, err := a.translateVirtualHost("test", vhost2)
	assert.Nil(t, err)

	assert.NotNil(t, routes1)
	assert.NotNil(t, routes2)

	added, deleted, updated := utils.CompareRoutes(routes1, routes2)
	assert.Nil(t, added)
	assert.Nil(t, deleted)
	assert.Nil(t, updated)
}
