package amesh

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

	"github.com/api7/amesh/e2e/framework"
)

type ResponseRewriteConfig struct {
	Body    string            `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type PluginConfigResponseRewriteTester struct {
	f *framework.Framework

	Body    string
	Headers map[string]string

	deletedHeaders map[string]struct{}

	nginxTunnel *httpexpect.Expect
	curlName    string
}

func NewTester(f *framework.Framework, conf *ResponseRewriteConfig) *PluginConfigResponseRewriteTester {
	return &PluginConfigResponseRewriteTester{
		f: f,

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
	assert.Nil(ginkgo.GinkgoT(), err, "marshal AmeshPluginConfig config")

	err = t.f.CreateResourceFromString(fmt.Sprintf(ampc, conf))
	assert.Nil(ginkgo.GinkgoT(), err, "create AmeshPluginConfig")

	time.Sleep(time.Second * 4)

	log.Infow("update config",
		zap.Any("body", t.Body),
		zap.Any("headers", t.Headers),
		zap.Any("deleted_headers", t.deletedHeaders),
	)
}

func (t *PluginConfigResponseRewriteTester) initNginxTunnel() {
	f := t.f

	if t.nginxTunnel == nil {
		name := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN())
		t.nginxTunnel = f.NewHTTPClientToNginx(name)

		time.Sleep(time.Second * 8)
	}
}

func (t *PluginConfigResponseRewriteTester) initCurl() {
	f := t.f

	if t.curlName == "" {
		f.CreateCurl()
		time.Sleep(time.Second * 8)
		t.curlName = "consumer"
	}
}

func (t *PluginConfigResponseRewriteTester) Create() {
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
	assert.Nil(ginkgo.GinkgoT(), err, "delete AmeshPluginConfig")

	log.Infow("delete config",
		zap.Any("deleted_headers", t.deletedHeaders),
	)

	time.Sleep(time.Second * 4)
}

func (t *PluginConfigResponseRewriteTester) ValidateInMeshNginxProxyAccess(withoutHeaders ...string) {
	f := t.f
	t.initNginxTunnel()

	resp := t.nginxTunnel.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

	log.Debugw("resp", zap.Any("headers", resp.Raw().Header), zap.Any("body", resp.Body().Raw()))
	if resp.Raw().StatusCode != http.StatusOK {
		log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
		assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
	}
	resp.Status(http.StatusOK)
	resp.Headers().Value("Via").Array().Contains("APISIX")

	for headerKey, headerValue := range t.Headers {
		log.Infof("validating header exists: " + headerKey)
		resp.Headers().Value(headerKey).Array().Contains(headerValue)
	}

	for header, _ := range t.deletedHeaders {
		log.Infof("validating header doesn't exist: " + header)
		resp.Headers().NotContainsKey(header)
	}

	for _, header := range withoutHeaders {
		log.Infof("validating header doesn't exist: " + header)
		resp.Headers().NotContainsKey(header)
	}

	if t.Body != "" {
		log.Infof("validating body")
		resp.Body().Equal(t.Body)
	}
}

func (t *PluginConfigResponseRewriteTester) ValidateInMeshCurlAccess(withoutHeaders ...string) {
	f := t.f
	t.initCurl()

	output := f.Curl("httpbin/ip")

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

var _ = ginkgo.Describe("[amesh-controller functions]", func() {
	f := framework.NewDefaultFramework()

	framework.Case("should be able to inject plugins", func() {
		t := NewTester(f, &ResponseRewriteConfig{
			Body: "BODY_REWRITE",
			Headers: map[string]string{
				"X-Header": "Rewrite",
			},
		})

		t.Create()
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()
	})

	framework.Case("should be able to update plugins", func() {
		t := NewTester(f, &ResponseRewriteConfig{
			Headers: map[string]string{
				"X-Header": "Rewrite",
			},
		})

		t.Create()
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()

		t.UpdateConfig(&ResponseRewriteConfig{
			Body: "BODY_REWRITE",
			Headers: map[string]string{
				"X-Header": "RewriteChanged",
			},
		})
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()

		t.UpdateConfig(&ResponseRewriteConfig{
			Body: "BODY_REWRITE",
			Headers: map[string]string{
				"X-Header-Changed": "HeaderChanged",
			},
		})
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()
	})
	framework.Case("should be able to delete plugins", func() {
		t := NewTester(f, &ResponseRewriteConfig{
			Headers: map[string]string{
				"X-Header": "Rewrite",
			},
			Body: "REWRITE",
		})

		t.Create()
		t.ValidateInMeshNginxProxyAccess()

		t.DeleteAmeshPluginConfig()
		t.ValidateInMeshNginxProxyAccess()

	})
})
