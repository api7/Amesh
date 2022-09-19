package amesh

import (
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

	ginkgo.FIt("should be able to inject plugins", func() {
		_ = f

		// FIXME: access through nginx proxy_pass doesn't log route matching in apisix logs
		err := f.CreateResourceFromString(ampc)
		assert.Nil(ginkgo.GinkgoT(), err, "create AmeshPluginConfig")

		f.CreateCurl()

		time.Sleep(time.Second * 8)

		output := f.Curl("httpbin/ip")

		assert.Contains(ginkgo.GinkgoT(), output, "X-Header: Rewrite", "check header")
		assert.Contains(ginkgo.GinkgoT(), output, "BODY_REWRITE", "check body")
	})
})
