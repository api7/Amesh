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

func (t *PluginConfigResponseRewriteTester) ValidateInMeshNginxProxyAccess(withoutHeaders ...string) {
	f := t.f
	t.initNginxTunnel()

	resp := t.nginxTunnel.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

	if resp.Raw().StatusCode != http.StatusOK {
		log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
		assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
	}
	resp.Status(http.StatusOK)
	resp.Headers().Value("Via").Array().Contains("APISIX")

	for headerKey, headerValue := range t.Headers {
		resp.Headers().Value(headerKey).Array().Contains(headerValue)
	}

	for header, _ := range t.deletedHeaders {
		resp.Headers().NotContainsKey(header)
	}

	for _, header := range withoutHeaders {
		resp.Headers().NotContainsKey(header)
	}

	if t.Body != "" {
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

	ginkgo.It("should be able to inject plugins", func() {
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

	ginkgo.It("should be able to update plugins", func() {
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
})
