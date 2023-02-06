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

package traffic_management

import (
	"net/http"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

type VirtualServiceTester struct {
	f *framework.Framework

	// Host -> config
	Configs map[string]*framework.VirtualServiceConfig
	// Valid versions is: v1, v2, v3
	Versions []string

	needNginx bool
	curlName  string
}

// opts: needNginx
func NewVirtualServiceTester(f *framework.Framework, versions []string, opts ...bool) *VirtualServiceTester {
	needNgx := true
	if len(opts) > 0 {
		needNgx = opts[0]
	}
	return &VirtualServiceTester{
		f:        f,
		Configs:  map[string]*framework.VirtualServiceConfig{},
		Versions: versions,

		needNginx: needNgx,
	}
}

func (t *VirtualServiceTester) Create() {
	f := t.f

	funcs := []func(){
		func() {
			t.curlName = "curl"
			f.CreateCurl(t.curlName)
			f.WaitForCurlReady(t.curlName)
		},
	}
	for _, version := range t.Versions {
		version := version
		funcs = append(funcs, func() {
			httpbin := "httpbin-" + version
			ngx := "ngx-" + version
			f.CreateHttpbinInMesh(httpbin)
			if t.needNginx {
				f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN("httpbin-kind"), true, ngx)
			}

			f.WaitForHttpbinReady(httpbin)
			if t.needNginx {
				f.WaitForNginxReady(ngx)
			}
		})
	}

	f.CreateNginxService()
	f.CreateHttpbinService()
	utils.ParallelRunAndWait(funcs...)

	time.Sleep(time.Second * 5)
}

func (t *VirtualServiceTester) AddRouteTo(host string, dest string) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}
	t.Configs[host].Routes = append(t.Configs[host].Routes, &framework.RouteConfig{
		Destinations: map[string]*framework.RouteDestinationConfig{
			dest: {
				Weight: 100,
			},
		},
	})
}

func (t *VirtualServiceTester) AddRouteToWithTimeout(host string, dest string, timeout float32) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}
	t.Configs[host].Routes = append(t.Configs[host].Routes, &framework.RouteConfig{
		Destinations: map[string]*framework.RouteDestinationConfig{
			dest: {
				Weight: 100,
			},
		},
		Timeout: timeout,
	})
}

func (t *VirtualServiceTester) AddWeightedRoutes(host string, dests []string) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}

	weight := 100 / len(dests)
	lastWeight := 100 - weight*(len(dests)-1)
	cfg := &framework.RouteConfig{
		Destinations: map[string]*framework.RouteDestinationConfig{},
	}
	for i, dest := range dests {
		cfg.Destinations[dest] = &framework.RouteDestinationConfig{
			Weight: weight,
		}
		if i == len(dests)-1 {
			cfg.Destinations[dest].Weight = lastWeight
		}
	}

	t.Configs[host].Routes = append(t.Configs[host].Routes, cfg)
}

func (t *VirtualServiceTester) AddRouteToIfHeaderIs(host string, dest, header, value string) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}
	t.Configs[host].Routes = append(t.Configs[host].Routes, &framework.RouteConfig{
		Match: &framework.RouteMatchRule{
			Headers: map[string]string{
				header: value,
			},
		},
		Destinations: map[string]*framework.RouteDestinationConfig{
			dest: {
				Weight: 100,
			},
		},
	})
}

func (t *VirtualServiceTester) AddRouteToWithAbortFault(host string, dest, header, value string) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}
	t.Configs[host].Routes = append(t.Configs[host].Routes, &framework.RouteConfig{
		Match: &framework.RouteMatchRule{
			Headers: map[string]string{
				header: value,
			},
		},
		Fault: &framework.RouteFaultRule{
			Abort: &framework.RouteFaultAbortRule{
				StatusCode: 555,
				Percentage: 100,
			},
			Delay: nil,
		},
		Destinations: map[string]*framework.RouteDestinationConfig{
			dest: {
				Weight: 100,
			},
		},
	})
}

func (t *VirtualServiceTester) AddRouteToWithDelayFault(host, dest string, header, value string, timeout float32) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}
	t.Configs[host].Routes = append(t.Configs[host].Routes, &framework.RouteConfig{
		Match: &framework.RouteMatchRule{
			Headers: map[string]string{
				header: value,
			},
		},
		Fault: &framework.RouteFaultRule{
			Delay: &framework.RouteFaultDelayRule{
				Duration:   timeout,
				Percentage: 100,
			},
			Abort: nil,
		},
		Destinations: map[string]*framework.RouteDestinationConfig{
			dest: {
				Weight: 100,
			},
		},
	})
}

func (t *VirtualServiceTester) AddRouteToWithMirror(host, dest string, mirrorDest, mirrorSubset string) {
	if _, ok := t.Configs[host]; !ok {
		t.Configs[host] = &framework.VirtualServiceConfig{
			Host:         host,
			Destinations: t.Versions,
			Routes:       nil,
		}
	}
	t.Configs[host].Routes = append(t.Configs[host].Routes, &framework.RouteConfig{
		Destinations: map[string]*framework.RouteDestinationConfig{
			dest: {
				Weight: 100,
			},
		},
		Mirror: &framework.RouteMirrorRule{
			Host:   mirrorDest,
			Subset: mirrorSubset,
		},
	})
}

func (t *VirtualServiceTester) ClearRoute(host string) {
	t.Configs[host].Routes = nil
}

func (t *VirtualServiceTester) ApplyRoute() {
	for host, config := range t.Configs {
		err := t.f.ApplyVirtualService(config)
		utils.AssertNil(err, "apply route for "+host)
	}
	// TODO: Support delete
	time.Sleep(time.Second * 5)
}

func (t *VirtualServiceTester) ValidateSingleVersionAccess(has, hasNo []string, args ...string) {
	t.DoAccess(http.StatusOK, has, hasNo, args...)
	t.DoAccess(http.StatusOK, has, hasNo, args...)
	t.DoAccess(http.StatusOK, has, hasNo, args...)
	t.DoAccess(http.StatusOK, has, hasNo, args...)
}

func (t *VirtualServiceTester) ValidateAccessible(has string, args ...string) {
	t.DoAccess(http.StatusOK, []string{has}, nil, args...)
}

func (t *VirtualServiceTester) ValidateInaccessible(code int, hasNo string, args ...string) {
	t.DoAccess(code, nil, []string{hasNo}, args...)
}

func (t *VirtualServiceTester) DoAccess(code int, has, hasNo []string, args ...string) string {
	log.SkipFramesOnce(1)
	output := t.f.CurlInPod(t.curlName, args...)
	switch code {
	case 555, http.StatusGatewayTimeout:
		assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure request doesn't via APISIX")
	default:
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure request via APISIX")
	}

	target := ""
	switch code {
	case http.StatusOK:
		target = "200 OK"
	case http.StatusGatewayTimeout:
		target = "504 Gateway Time-out"
	case 555:
		target = "HTTP/1.1 555"
	}

	assert.Contains(ginkgo.GinkgoT(), output, target, "make sure status code")

	for _, str := range has {
		assert.Contains(ginkgo.GinkgoT(), output, str, "make sure it works properly")
	}
	for _, str := range hasNo {
		assert.NotContains(ginkgo.GinkgoT(), output, str, "make sure it works properly")
	}

	return output
}

func (t *VirtualServiceTester) ValidateTimeout(longerThan, shorterThan time.Duration, f func()) {
	start := time.Now()
	f()
	duration := time.Since(start)
	log.SkipFramesOnce(1)
	log.Infof("The duration should: %v < %v < %v", longerThan, duration, shorterThan)
	assert.True(ginkgo.GinkgoT(), duration > longerThan)
	assert.True(ginkgo.GinkgoT(), duration < shorterThan)
}
