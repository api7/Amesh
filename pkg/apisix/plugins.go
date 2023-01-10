// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apisix

const (
	PluginFaultInjection = "fault-injection"
	PluginProxyMirror    = "proxy-mirror"
	PluginTrafficSplit   = "traffic-split"
	PluginPrometheus     = "prometheus"
)

type FaultInjectionAbort struct {
	HttpStatus uint32 `json:"http_status,omitempty"`
	Body       string `json:"body,omitempty"`
	Percentage uint32 `json:"percentage,omitempty"`
	Vars       []*Var `json:"vars,omitempty"`
}

type FaultInjectionDelay struct {
	// Duration in seconds
	Duration   int64  `json:"duration,omitempty"`
	Percentage uint32 `json:"percentage,omitempty"`
	Vars       []*Var `json:"vars,omitempty" json:"vars,omitempty"`
}

type FaultInjection struct {
	Abort *FaultInjectionAbort `json:"abort,omitempty"`
	Delay *FaultInjectionDelay `json:"delay,omitempty"`
}

type TrafficSplitWeightedUpstreams struct {
	UpstreamId string `json:"upstream_id,omitempty"`
	Weight     uint32 `json:"weight,omitempty"`
}

type TrafficSplitRule struct {
	Match             []*Var                           `json:"match,omitempty"`
	WeightedUpstreams []*TrafficSplitWeightedUpstreams `json:"weighted_upstreams,omitempty"`
}

type TrafficSplit struct {
	Rules []*TrafficSplitRule `json:"rules,omitempty"`
}

type ProxyMirror struct {
	Host        string  `json:"host,omitempty"`
	Path        string  `json:"path,omitempty"`
	SampleRatio float32 `json:"sample_ratio,omitempty"`
}

type Prometheus struct {
	// When set to true, would print route/service name instead of id in Prometheus metric.
	PreferName bool `json:"prefer_name,omitempty"`
}
