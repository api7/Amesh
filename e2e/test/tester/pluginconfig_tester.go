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

package tester

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gavv/httpexpect/v2"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	controllerutils "github.com/api7/amesh/controller/utils"
	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

type ResponseRewriteConfig struct {
	Body    string            `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type PluginConfigResponseRewriteTester struct {
	f      *framework.Framework
	logger *log.Logger

	Body    string
	Headers map[string]string

	deletedHeaders map[string]struct{}

	nginxName   string
	nginxClient *httpexpect.Expect
	curlName    string

	nginxReady bool
	curlReady  bool
}

func NewPluginConfigTester(f *framework.Framework, conf *ResponseRewriteConfig) *PluginConfigResponseRewriteTester {
	logger, err := log.NewLogger(
		log.WithLogLevel("info"),
		log.WithSkipFrames(3),
	)
	utils.AssertNil(err, "create logger")
	return &PluginConfigResponseRewriteTester{
		f:      f,
		logger: logger,

		Body:    conf.Body,
		Headers: conf.Headers,
	}
}

func (t *PluginConfigResponseRewriteTester) applyAmeshPluginConfig() {
	ampc := `
apiVersion: apisix.apache.org/v1alpha1
kind: AmeshPluginConfig
metadata:
  name: ampc-sample
spec:
  plugins:
    - name: response-rewrite
      type: ""
      config: '%s'
`
	conf, err := json.Marshal(ResponseRewriteConfig{
		Body:    t.Body,
		Headers: t.Headers,
	})
	utils.AssertNil(err, "marshal AmeshPluginConfig config")

	err = t.f.ApplyResourceFromString(fmt.Sprintf(ampc, conf))
	utils.AssertNil(err, "create AmeshPluginConfig")

	// TODO: Check generation?
	err = t.f.WaitForAmeshPluginConfigEvents("ampc-sample", controllerutils.ConditionSync, metav1.ConditionTrue)
	utils.AssertNil(err, "wait for AmeshPluginConfig synchronization")

	time.Sleep(time.Second * 10)

	t.logger.SkipFramesOnce(1).Infow("update config",
		zap.Any("body", t.Body),
		zap.Any("headers", t.Headers),
		zap.Any("deleted_headers", t.deletedHeaders),
	)
}

func (t *PluginConfigResponseRewriteTester) initNginxClient() {
	if t.nginxClient == nil {
		utils.ParallelRunAndWait(func() {
			t.f.CreateHttpbinInMesh()
		}, func() {
			t.nginxName = t.f.CreateNginxInMeshTo(t.f.GetHttpBinServiceFQDN(), false)
		})
	}
}

func (t *PluginConfigResponseRewriteTester) initCurl() {
	f := t.f

	if t.curlName == "" {
		t.curlName = f.CreateCurl()
	}
}

func (t *PluginConfigResponseRewriteTester) waitNginxClient() {
	if !t.nginxReady && t.nginxName != "" {
		utils.ParallelRunAndWait(func() {
			t.f.WaitForHttpbinReady()
			t.f.WaitForNginxReady(t.nginxName)
		})

		t.nginxClient, _ = t.f.NewHTTPClientToNginx(t.nginxName)
		t.nginxReady = true
	}
}

func (t *PluginConfigResponseRewriteTester) waitCurl() {
	if !t.curlReady {
		t.f.WaitForCurlReady()
		t.curlReady = true
	}
}

func (t *PluginConfigResponseRewriteTester) Create() {
	go func() {
		defer ginkgo.GinkgoRecover()
		t.initNginxClient()
	}()
	go func() {
		defer ginkgo.GinkgoRecover()
		t.initCurl()
	}()
	//t.initNginxClient()
	//t.initCurl()

	t.applyAmeshPluginConfig()
}

func (t *PluginConfigResponseRewriteTester) UpdateConfig(conf *ResponseRewriteConfig) {
	t.Body = conf.Body

	t.deletedHeaders = map[string]struct{}{}
	for key, _ := range t.Headers {
		if _, ok := conf.Headers[key]; !ok {
			t.deletedHeaders[key] = struct{}{}
		}
	}

	t.Headers = conf.Headers

	t.applyAmeshPluginConfig()
}

func (t *PluginConfigResponseRewriteTester) DeleteAmeshPluginConfig() {
	t.Body = ""
	t.deletedHeaders = map[string]struct{}{}
	for key, _ := range t.Headers {
		t.deletedHeaders[key] = struct{}{}
	}
	t.Headers = map[string]string{}

	err := t.f.DeleteResourceFromString("ampc", "ampc-sample")
	utils.AssertNil(err, "delete AmeshPluginConfig")

	t.logger.Infow("delete config",
		zap.Any("deleted_headers", t.deletedHeaders),
	)

	time.Sleep(time.Second * 4)
}

func (t *PluginConfigResponseRewriteTester) ValidateInMeshNginxProxyAccess(withoutHeaders ...string) {
	f := t.f
	t.waitNginxClient()

	resp := t.nginxClient.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

	t.logger.Debugw("resp", zap.Any("headers", resp.Raw().Header), zap.Any("body", resp.Body().Raw()))
	if resp.Raw().StatusCode != http.StatusOK {
		log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
		assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
	}
	resp.Status(http.StatusOK)
	resp.Headers().Value("Via").Array().Contains("APISIX")

	for headerKey, headerValue := range t.Headers {
		t.logger.Infof("validating header exists: " + headerKey)
		resp.Headers().Value(headerKey).Array().Contains(headerValue)
	}

	for header, _ := range t.deletedHeaders {
		t.logger.Infof("validating header doesn't exist: " + header)
		resp.Headers().NotContainsKey(header)
	}

	for _, header := range withoutHeaders {
		t.logger.Infof("validating header doesn't exist: " + header)
		resp.Headers().NotContainsKey(header)
	}

	if t.Body != "" {
		t.logger.Infof("validating body")
		resp.Body().Equal(t.Body)
	}
}

func (t *PluginConfigResponseRewriteTester) ValidateInMeshCurlAccess(withoutHeaders ...string) {
	f := t.f
	t.waitCurl()

	output := f.CurlInPod(t.curlName, "httpbin/ip")

	assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")

	for headerKey, headerValue := range t.Headers {
		assert.Contains(ginkgo.GinkgoT(), output, headerKey+": "+headerValue, "check header "+headerKey)
	}
	for header, _ := range t.deletedHeaders {
		assert.NotContains(ginkgo.GinkgoT(), output, header+": ", "check header "+header)
	}
	for _, header := range withoutHeaders {
		assert.NotContains(ginkgo.GinkgoT(), output, header+": ", "check header "+header)
	}

	if t.Body != "" {
		assert.Contains(ginkgo.GinkgoT(), output, t.Body, "check body")
	}
}
