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

package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVirtualServiceManifest(t *testing.T) {
	vs := &VirtualServiceConfig{
		Host: "nginx-kind",
		Destinations: []string{
			"v1",
			"v2",
		},
		Routes: []*RouteConfig{
			{
				Match: &RouteMatchRule{
					Headers: map[string]string{
						"X-User": "abort",
					},
				},
				Fault: &RouteFaultRule{
					Abort: &RouteFaultAbortRule{
						StatusCode: 555,
						Percentage: 100,
					},
				},
				Destinations: map[string]*RouteDestinationConfig{
					"v1": {
						Weight: 100,
					},
				},
			},
			{
				Match: &RouteMatchRule{
					Headers: map[string]string{
						"X-User": "delay",
					},
				},
				Fault: &RouteFaultRule{
					Delay: &RouteFaultDelayRule{
						Duration:   2,
						Percentage: 100,
					},
				},
				Destinations: map[string]*RouteDestinationConfig{
					"v1": {
						Weight: 100,
					},
				},
			},
			{
				Match: &RouteMatchRule{
					Headers: map[string]string{
						"X-User": "canary",
					},
				},
				Destinations: map[string]*RouteDestinationConfig{
					"v1": {
						Weight: 50,
					},
					"v2": {
						Weight: 50,
					},
				},
				Mirror: &RouteMirrorRule{
					Host:   "mirror-backend",
					Subset: "v1",
				},
			},
			{
				Match: &RouteMatchRule{},
				Destinations: map[string]*RouteDestinationConfig{
					"v1": {
						Weight: 100,
					},
				},
				Timeout: 10,
			},
		},
	}

	yaml := vs.GenerateYAML()
	assert.Equal(t, `
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: nginx-kind
spec:
  host: nginx-kind
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: nginx-kind
spec:
  hosts:
  - nginx-kind
  http:
  - name: route-0
    match:
    - headers:
        X-User:
          exact: abort
    route:
    - destination:
        host: nginx-kind
        subset: v1
      weight: 100
    fault:
      abort:
        httpStatus: 555
        percentage:
          value: 100
  - name: route-1
    match:
    - headers:
        X-User:
          exact: delay
    route:
    - destination:
        host: nginx-kind
        subset: v1
      weight: 100
    fault:
      delay:
        fixedDelay: 2s
        percentage:
          value: 100
  - name: route-2
    match:
    - headers:
        X-User:
          exact: canary
    route:
    - destination:
        host: nginx-kind
        subset: v1
      weight: 50
    - destination:
        host: nginx-kind
        subset: v2
      weight: 50
    mirror:
      host: mirror-backend
      subset: v1
    mirrorPercentage:
      value: 100.0
  - name: route-3
    route:
    - destination:
        host: nginx-kind
        subset: v1
      weight: 100
    timeout: 10s
`, yaml)
}
