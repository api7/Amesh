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
	"github.com/stretchr/testify/assert"
	"time"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
	"github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to config timeout", func() {
		httpbinName := "delay-httpbin"

		f.CreateHttpbinInMesh(httpbinName)
		f.WaitForHttpbinReady(httpbinName)

		err := f.ApplyVirtualService(&framework.VirtualServiceConfig{
			Host: httpbinName,
			Destinations: map[string]struct{}{
				"v1": {},
			},
			Routes: []*framework.RouteConfig{
				{
					Match: &framework.RouteMatchRule{
						Headers: map[string]string{
							"User": "fault-delay",
						},
					},
					Fault: &framework.RouteFaultRule{
						Delay: &framework.RouteFaultDelayRule{
							Duration:   10,
							Percentage: 100,
						},
						Abort: nil,
					},
					Destinations: map[string]*framework.RouteDestinationConfig{
						"v1": {
							Weight: 100,
						},
					},
				},
				{
					Destinations: map[string]*framework.RouteDestinationConfig{
						"v1": {
							Weight: 100,
						},
					},
				},
			},
		})
		utils.AssertNil(err)
		time.Sleep(time.Second * 5)

		ngxName := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(httpbinName), true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)

		err = f.ApplyVirtualService(&framework.VirtualServiceConfig{
			Host: ngxName,
			Destinations: map[string]struct{}{
				"v1": {},
			},
			Routes: []*framework.RouteConfig{
				{
					Destinations: map[string]*framework.RouteDestinationConfig{
						"v1": {
							Weight: 100,
						},
					},
					Timeout: 5,
				},
			},
		})
		utils.AssertNil(err)
		time.Sleep(time.Second * 5)

		curl := f.CreateCurl()
		f.WaitForCurlReady(curl)

		// Make sure the route works fine
		output := f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

		// Make sure the timeout works
		output = f.CurlInPod(curl, ngxName+"/ip", "-H", "User: fault-delay")
		assert.Contains(ginkgo.GinkgoT(), output, "504 Gateway Time-out", "make sure it works properly")

		// Increase timeout toleration, should work again
		err = f.ApplyVirtualService(&framework.VirtualServiceConfig{
			Host: ngxName,
			Destinations: map[string]struct{}{
				"v1": {},
			},
			Routes: []*framework.RouteConfig{
				{
					Destinations: map[string]*framework.RouteDestinationConfig{
						"v1": {
							Weight: 100,
						},
					},
					Timeout: 12,
				},
			},
		})
		utils.AssertNil(err)
		time.Sleep(time.Second * 5)

		output = f.CurlInPod(curl, ngxName+"/ip", "-H", "User: fault-delay")
		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
	})
})
