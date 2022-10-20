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
	"net/http"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[basic proxy functions]", func() {
	f := framework.NewDefaultFramework()

	utils.Case("outside mesh should be able to access inside mesh", func() {
		// Outside NGINX -> Inside HTTPBIN

		ngxName := ""

		utils.ParallelRunAndWait(func() {
			f.CreateHttpbinInMesh()
			f.WaitForHttpbinReady()
		}, func() {
			ngxName = f.CreateNginxOutsideMeshTo(f.GetHttpBinServiceFQDN(), true)
			f.WaitForNginxReady(ngxName)
		})

		client, _ := f.NewHTTPClientToNginx(ngxName)
		resp := client.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")
	})

	utils.Case("inside mesh should be able to access inside mesh", func() {
		// Inside NGINX -> Inside HTTPBIN

		ngxName := ""

		utils.ParallelRunAndWait(func() {
			f.CreateHttpbinInMesh()
			f.WaitForHttpbinReady()
		}, func() {
			ngxName = f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(), true)
			f.WaitForNginxReady(ngxName)
		})

		client, _ := f.NewHTTPClientToNginx(ngxName)
		resp := client.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")
	})

	utils.Case("inside mesh curl should be able to access outside mesh", func() {
		// Inside Curl -> (Outside NGINX -> Outside HTTPBIN)

		ngxName := ""

		httpbinName := "httpbin-outside"
		f.CreateHttpbinOutsideMesh(httpbinName)

		ngxName = f.CreateNginxOutsideMeshTo(f.GetHttpBinServiceFQDN(httpbinName), false)
		curl := f.CreateCurl()

		utils.ParallelRunAndWait(func() {
			f.WaitForNginxReady(ngxName)
		}, func() {
			f.WaitForHttpbinReady(httpbinName)
		}, func() {
			f.WaitForCurlReady(curl)
		})

		output := f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
	})

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
		f.MakeNginxInsideMesh(ngxName, svc, true)
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
		f.MakeNginxOutsideMesh(ngxName, svc, true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)
		output = f.CurlInPod(curl, ngxName+"/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
		assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
	})
})
