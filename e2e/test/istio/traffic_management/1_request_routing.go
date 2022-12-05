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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to abort", func() {
		apply := func(str string) {
			err := f.ApplyResourceFromString(str)
			utils.AssertNil(err, "apply resource")

			time.Sleep(time.Second * 5)
		}

		// deploy apps
		httpbinV1 := "httpbin-v1"
		httpbinV2 := "httpbin-v2"
		httpbinService := "httpbin-kind" // versioned httpbin service
		nginxService := "nginx-kind"     // versioned nginx service

		ngxV1Name := "ngx-v1"
		ngxV2Name := "ngx-v2"
		utils.ParallelRunAndWait(func() {
			f.CreateHttpbinInMesh(httpbinV1)
			f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(httpbinService), true, ngxV1Name)

			f.WaitForHttpbinReady(httpbinV1)
			f.WaitForNginxReady(ngxV1Name)
		}, func() {
			f.CreateHttpbinInMesh(httpbinV2)
			f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(httpbinService), true, ngxV2Name)

			f.WaitForHttpbinReady(httpbinV2)
			f.WaitForNginxReady(ngxV2Name)
		}, func() {
			f.CreateNginxService()
			f.CreateHttpbinService()
		})

		time.Sleep(time.Second * 5)

		// destination rule
		apply(`
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
`)

		// Basic config to v1
		apply(`
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: nginx-kind
spec:
  hosts:
  - nginx-kind
  http:
  - name: route-v1
    route:
    - destination:
        host: nginx-kind
        subset: v1
`)
		// Validate normal access
		curlName := "curl"
		f.CreateCurl(curlName)
		f.WaitForCurlReady(curlName)

		checkIsV1 := func() {
			output := f.CurlInPod(curlName, nginxService+"/status")
			assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "ngx-v1", "make sure it works properly")
			assert.NotContains(ginkgo.GinkgoT(), output, "ngx-v2", "make sure it works properly")
		}
		checkIsV1()
		checkIsV1()
		checkIsV1()
		checkIsV1()

		// Basic config to v2
		apply(`
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: nginx-kind
spec:
  hosts:
  - nginx-kind
  http:
  - name: route-v2
    route:
    - destination:
        host: nginx-kind
        subset: v2
`)

		//resp = clientV2.GET("/status").WithHeader("Host", f.GetHttpBinServiceFQDN(httpbinV2)).Expect()
		//shouldOK(resp, "v2")
		checkIsV2 := func() {
			output := f.CurlInPod(curlName, nginxService+"/status")
			assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
			assert.NotContains(ginkgo.GinkgoT(), output, "ngx-v1", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "ngx-v2", "make sure it works properly")
		}
		checkIsV2()
		checkIsV2()
		checkIsV2()
		checkIsV2()

		// Match Conditions
		apply(`
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: nginx-kind
spec:
  hosts:
  - nginx-kind
  http:
  - route:
    - destination:
        host: nginx-kind
        subset: v2
    match:
    - headers:
        X-USER:
          exact: v2
  - route:
    - destination:
        host: nginx-kind
        subset: v1
`)
		checkMatchConditions := func() {
			output := f.CurlInPod(curlName, nginxService+"/status")
			assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "ngx-v1", "make sure it works properly")
			assert.NotContains(ginkgo.GinkgoT(), output, "ngx-v2", "make sure it works properly")

			output = f.CurlInPod(curlName, nginxService+"/status", "-H", "X-USER: v2")
			assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
			assert.NotContains(ginkgo.GinkgoT(), output, "ngx-v1", "make sure it works properly")
			assert.Contains(ginkgo.GinkgoT(), output, "ngx-v2", "make sure it works properly")
		}
		checkMatchConditions()
		checkMatchConditions()
		checkMatchConditions()
		checkMatchConditions()
	})
})
