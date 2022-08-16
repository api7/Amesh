package base

import (
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework"
)

var _ = ginkgo.Describe("[basic proxy functions]", func() {
	f := framework.NewDefaultFramework()

	ginkgo.It("should be able to proxy", func() {
		_ = f

		name := f.CreateNginxTo(f.GetHttpBinServiceFQDN())
		tunnel := f.NewHTTPClientToNginx(name)

		time.Sleep(time.Second * 8)
		resp := tunnel.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN()).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			ginkgo.GinkgoT().Logf("status code is %v, please check logs", resp.Raw().StatusCode)
			time.Sleep(time.Hour * 1000)
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")

		ginkgo.GinkgoT().Logf("status code is 200, please check logs")
	})
})
