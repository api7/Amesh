package amesh

import (
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
)

const (
	ampc = `
apiVersion: apisix.apache.org/v1alpha1
kind: AmeshPluginConfig
metadata:
  name: ameshpluginconfig-sample
spec:
  plugins:
    - name: response-rewrite
      type: ""
      config: '{"body": "BODY_REWRITE", "headers": {"X-Header":"Rewrite"}}'
`
)

var _ = ginkgo.Describe("[amesh-controller functions]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("should be able to inject plugins", func() {
		_ = f

		err := f.CreateResourceFromString(ampc)
		assert.Nil(ginkgo.GinkgoT(), err, "create AmeshPluginConfig")

		f.CreateCurl()
		time.Sleep(time.Second * 8)
		output := f.Curl("httpbin/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
		assert.Contains(ginkgo.GinkgoT(), output, "X-Header: Rewrite", "check header")
		assert.Contains(ginkgo.GinkgoT(), output, "BODY_REWRITE", "check body")

		// Nginx access
		name := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN())
		time.Sleep(time.Second * 8)

		tunnel := f.NewHTTPClientToNginx(name)
		resp := tunnel.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			ginkgo.GinkgoT().Logf("status code is %v, please check logs", resp.Raw().StatusCode)
			//time.Sleep(time.Hour * 1000)
			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Headers().Value("X-Header").Array().Contains("Rewrite")
		resp.Body().Contains("BODY_REWRITE")
	})
})
