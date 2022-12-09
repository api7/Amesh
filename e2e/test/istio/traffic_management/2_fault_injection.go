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

var _ = ginkgo.Describe("[istio functions] Fault Injection:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to abort", func() {
		name := "abort-httpbin"

		f.CreateHttpbinInMesh(name)
		f.WaitForHttpbinReady(name)

		err := f.ApplyVirtualService(&framework.VirtualServiceConfig{
			Host: name,
			Destinations: map[string]struct{}{
				"v1": {},
			},
			Routes: []*framework.RouteConfig{
				{
					Match: &framework.RouteMatchRule{
						Headers: map[string]string{
							"User": "fault-abort",
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

		ngxName := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(name), true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)

		client, _ := f.NewHTTPClientToNginx(ngxName)

		// Validate normal access
		resp := client.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN(name)).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			utils.DebugSleep(time.Hour)

			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")

		// Validate User:fault-abort access
		resp = client.GET("/ip").
			WithHeader("Host", f.GetServiceFQDN(ngxName)).
			WithHeader("User", "fault-abort").
			Expect()

		if resp.Raw().StatusCode != 555 {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			utils.DebugSleep(time.Hour)

			assert.Equal(ginkgo.GinkgoT(), 555, resp.Raw().StatusCode, "status code")
		}
		resp.Status(555)
	})

	utils.Case("should be able to delay", func() {
		name := "delay-httpbin"

		f.CreateHttpbinInMesh(name)
		f.WaitForHttpbinReady(name)

		err := f.ApplyVirtualService(&framework.VirtualServiceConfig{
			Host: name,
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

		ngxName := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(name), true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)

		client, _ := f.NewHTTPClientToNginx(ngxName)

		// Validate normal access
		resp := client.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN(name)).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			utils.DebugSleep(time.Hour)

			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")

		// Validate User:fault-delay access
		start := time.Now()
		resp = client.GET("/ip").
			WithHeader("Host", f.GetServiceFQDN(ngxName)).
			WithHeader("User", "fault-delay").
			Expect()
		duration := time.Since(start)

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			utils.DebugSleep(time.Hour)

			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)

		log.Infof("The timeout is: %v", duration)
		assert.True(ginkgo.GinkgoT(), duration > time.Second*10)
		assert.True(ginkgo.GinkgoT(), duration < time.Second*12)
	})
})
