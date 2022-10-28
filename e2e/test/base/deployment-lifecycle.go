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

package base

import (
	"github.com/api7/amesh/pkg/apisix"
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[deployment lifecycle]", func() {
	f := framework.NewDefaultFramework()

	utils.Case("inside mesh curl should be able to access inside mesh after pod deletion", func() {
		// Inside Curl -> (Inside NGINX -> Outside HTTPBIN)
		// Delete Inside NGINX and make it restart

		ngxName := ""

		httpbinName := "httpbin-outside"
		f.CreateHttpbinOutsideMesh(httpbinName)

		svc := f.GetHttpBinServiceFQDN(httpbinName)
		ngxName = f.CreateNginxInMeshTo(svc, false)

		curl := f.CreateCurl()

		utils.ParallelRunAndWait(func() {
			f.WaitForNginxReady(ngxName)
		}, func() {
			f.WaitForHttpbinReady(httpbinName)
		}, func() {
			f.WaitForCurlReady(curl)
		})
		time.Sleep(time.Second * 5)

		output := f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

		//time.Sleep(time.Hour)
		// Delete the pod
		log.Infof(color.BlueString("Delete " + ngxName + " pods"))
		utils.AssertNil(f.DeletePodByLabel(f.AppNamespace(), "app="+ngxName), "delete nginx pods")
		f.WaitForNginxReady(ngxName)
		time.Sleep(time.Second * 5)
		output = f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
	})

	utils.Case("inside mesh curl should be able to access outside->inside mesh", func() {
		// Inside Curl -> (Outside NGINX -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX -> Outside HTTPBIN)

		ngxName := ""

		httpbinName := "httpbin-outside"
		f.CreateHttpbinOutsideMesh(httpbinName)

		svc := f.GetHttpBinServiceFQDN(httpbinName)
		ngxName = f.CreateNginxOutsideMeshTo(svc, false)

		curl := f.CreateCurl()

		utils.ParallelRunAndWait(func() {
			f.WaitForNginxReady(ngxName)
		}, func() {
			f.WaitForHttpbinReady(httpbinName)
		}, func() {
			f.WaitForCurlReady(curl)
		})
		time.Sleep(time.Second * 5)

		output := f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

		// Make the ngx inside mesh
		log.Infof(color.BlueString("Make " + ngxName + " in mesh"))
		f.MakeNginxInsideMesh(ngxName, true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)
		output = f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
	})

	utils.Case("inside mesh curl should be able to access inside->outside mesh", func() {
		// Inside Curl -> (Inside NGINX -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Outside NGINX -> Outside HTTPBIN)

		ngxName := ""

		httpbinName := "httpbin-outside"
		f.CreateHttpbinOutsideMesh(httpbinName)

		svc := f.GetHttpBinServiceFQDN(httpbinName)
		ngxName = f.CreateNginxInMeshTo(svc, false)

		curl := f.CreateCurl()

		utils.ParallelRunAndWait(func() {
			f.WaitForNginxReady(ngxName)
		}, func() {
			f.WaitForHttpbinReady(httpbinName)
		}, func() {
			f.WaitForCurlReady(curl)
		})
		time.Sleep(time.Second * 5)

		output := f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

		// Make the ngx outside mesh
		log.Infof(color.BlueString("Make " + ngxName + " in mesh"))
		f.MakeNginxOutsideMesh(ngxName, true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)
		output = f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
	})

	utils.Case("should be able to handle replicas", func() {
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Replica: 2) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)

		ngxName := ""

		httpbinName := "httpbin-outside"
		f.CreateHttpbinOutsideMesh(httpbinName)

		svc := f.GetHttpBinServiceFQDN(httpbinName)
		ngxName = f.CreateNginxInMeshTo(svc, false)

		curl := f.CreateCurl()

		utils.ParallelRunAndWait(func() {
			f.WaitForNginxReady(ngxName)
		}, func() {
			f.WaitForHttpbinReady(httpbinName)
		}, func() {
			f.WaitForCurlReady(curl)
		})
		time.Sleep(time.Second * 5)

		output := f.CurlInPod(curl, ngxName+"/ip")
		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

		// Verify upstream.nodes length
		upstreams := f.GetSidecarUpstreams(curl)
		var httpbinUpstream *apisix.Upstream
		for _, upstream := range upstreams {
			if strings.Contains(upstream.Name, ngxName) {
				httpbinUpstream = upstream
			}
		}
		assert.NotNil(ginkgo.GinkgoT(), httpbinUpstream)
		assert.Equal(ginkgo.GinkgoT(), 1, len(httpbinUpstream.Nodes), "httpbin nodes count")

		// Scale to 2 and verify upstream.nodes length
		f.ScaleNginx(ngxName, 2, true)
		upstreams = f.GetSidecarUpstreams(curl)
		httpbinUpstream = nil
		for _, upstream := range upstreams {
			if strings.Contains(upstream.Name, ngxName) {
				httpbinUpstream = upstream
			}
		}
		assert.NotNil(ginkgo.GinkgoT(), httpbinUpstream)
		assert.Equal(ginkgo.GinkgoT(), 2, len(httpbinUpstream.Nodes), "httpbin nodes count")

		output = f.CurlInPod(curl, ngxName+"/ip")
		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

		// Scale to 1 and verify upstream.nodes length
		f.ScaleNginx(ngxName, 1, true)
		upstreams = f.GetSidecarUpstreams(curl)
		httpbinUpstream = nil
		for _, upstream := range upstreams {
			if strings.Contains(upstream.Name, ngxName) {
				httpbinUpstream = upstream
			}
		}
		assert.NotNil(ginkgo.GinkgoT(), httpbinUpstream)
		assert.Equal(ginkgo.GinkgoT(), 1, len(httpbinUpstream.Nodes), "httpbin nodes count")

		output = f.CurlInPod(curl, ngxName+"/ip")
		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")

	})
})
