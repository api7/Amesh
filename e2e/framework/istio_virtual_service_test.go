package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVirtualServiceManifest(t *testing.T) {
	vs := &VirtualServiceConfig{
		Host: "nginx-kind",
		Destinations: map[string]struct{}{
			"v1": {},
			"v2": {},
		},
		Routes: []*RouteConfig{
			// this key is route name, not dest key
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
  - name: route-3
    route:
    - destination:
        host: nginx-kind
        subset: v1
      weight: 100
    timeout: 10s
`, yaml)
}
