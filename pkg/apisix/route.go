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
package apisix

// RouteStatus Enumerations.
type RouteStatus int32

const (
	RouteDisable RouteStatus = 0
	RouteEnable  RouteStatus = 1
)

// A Route contains multiple parts but basically can be grouped
// into three:
// 1). Route match, fields like uris, hosts, remote_addrs are the
// predicates to indicate whether a request can hit the route.
// 2). Route action, upstream_id specifies the backend upstream
// object, which guides Apache APISIX how to route request.
// 3). Plugins, plugins will run before/after the route action,
// some plugins are "terminated" so may be requests will be returned
// on the APISIX side (like authentication failures).
type Route struct {
	// URI array used to do the route match.
	// At least one item should be configured and each of them cannot be
	// duplicated.
	Uris []string `json:"uris,omitempty"`
	// The route name, it's useful for the logging but it's not required.
	Name string `json:"name,omitempty"`
	// The route id.
	Id string `json:"id,omitempty"`
	// Textual descriptions used to describe the route use.
	Desc string `json:"desc,omitempty"`
	// Priority of this route, used to decide which route should be used when
	// multiple routes contains same URI.
	// Larger value means higher priority. The default value is 0.
	Priority int32 `json:"priority,omitempty"`
	// HTTP Methods used to do the route match.
	Methods []string `json:"methods,omitempty"`
	// Host array used to do the route match.
	Hosts []string `json:"hosts,omitempty"`
	// Remote address array used to do the route match.
	RemoteAddrs []string `json:"remote_addrs,omitempty"`
	// Nginx vars used to do the route match.
	Vars []*Var `json:"vars,omitempty"`
	// Embedded plugins.
	Plugins map[string]interface{} `json:"plugins,omitempty"`
	// The referred service id.
	ServiceId string `json:"service_id,omitempty"`
	// The referred upstream id.
	UpstreamId string `json:"upstream_id,omitempty"`
	// The route status.
	Status RouteStatus `json:"status,omitempty"`
	// Timeout sets the I/O operations timeouts on the route level.
	Timeout *Timeout `json:"timeout,omitempty"`
	// enable_websocket indicates whether the websocket proxy is enabled.
	EnableWebsocket bool `json:"enable_websocket,omitempty"`
	// Labels contains some labels for the sake of management.
	Labels map[string]string `json:"labels,omitempty"  `
	// create_time indicate the create timestamp of this route.
	CreateTime int64 `json:"create_time,omitempty"`
	// update_time indicate the last update timestamp of this route.
	UpdateTime int64 `json:"update_time,omitempty"`
}

func (route *Route) Copy() *Route {
	r := &Route{
		Uris:            nil,
		Name:            route.Name,
		Id:              route.Id,
		Desc:            route.Desc,
		Priority:        route.Priority,
		Methods:         nil,
		Hosts:           nil,
		RemoteAddrs:     route.RemoteAddrs,
		Vars:            route.Vars,
		Plugins:         map[string]interface{}{},
		ServiceId:       route.ServiceId,
		UpstreamId:      route.UpstreamId,
		Status:          route.Status,
		Timeout:         route.Timeout,
		EnableWebsocket: route.EnableWebsocket,
		Labels:          map[string]string{},
		CreateTime:      route.CreateTime,
		UpdateTime:      route.UpdateTime,
	}
	for _, val := range route.Uris {
		r.Uris = append(r.Uris, val)
	}
	for _, val := range route.Methods {
		r.Methods = append(r.Methods, val)
	}
	for _, val := range route.Hosts {
		r.Hosts = append(r.Hosts, val)
	}
	for k, v := range route.Plugins {
		r.Plugins[k] = v
	}
	for k, v := range route.Labels {
		r.Labels[k] = v
	}

	return r
}
