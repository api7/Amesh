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
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[basic proxy functions]", func() {
	f := framework.NewDefaultFramework()

	utils.Case("should be able to proxy outside mesh", func() {
		_ = f

		name := f.CreateNginxOutsideMeshTo(f.GetHttpBinServiceFQDN(), true)
		tunnel := f.NewHTTPClientToNginx(name)

		time.Sleep(time.Second * 8)
		resp := tunnel.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			//time.Sleep(time.Hour * 1000)
			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")
	})

	utils.Case("should be able to proxy inside mesh", func() {
		_ = f

		name := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(), true)

		tunnel := f.NewHTTPClientToNginx(name)

		time.Sleep(time.Second * 8)
		resp := tunnel.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			//time.Sleep(time.Hour * 1000)
			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")
	})
})
