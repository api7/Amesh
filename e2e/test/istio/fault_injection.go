package istio

import (
	"net/http"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[istio functions] Fault Injection:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to abort", func() {
		name := "abort-httpbin"

		f.CreateHttpbinInMesh(name)
		f.WaitForHttpbinReady(name)

		f.CreateVirtualServiceWithFaultAbort(&framework.FaultAbortArgs{
			Name:        "virtual-service-" + name,
			Host:        name,
			Header:      "User",
			HeaderValue: "fault-abort",
			Status:      555,
		})

		ngxName := f.CreateNginxInMeshTo(f.GetHttpBinServiceFQDN(name), true)
		f.WaitForNginxReady(ngxName)

		time.Sleep(time.Second * 5)

		client, _ := f.NewHTTPClientToNginx(ngxName)

		// Validate normal access
		resp := client.GET("/ip").WithHeader("Host", f.GetHttpBinServiceFQDN(name)).Expect()

		if resp.Raw().StatusCode != http.StatusOK {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			utils.DebugSleep(time.Hour)

			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(http.StatusOK)
		resp.Headers().Value("Via").Array().Contains("APISIX")
		resp.Body().Contains("origin")

		// Validate User:fault-abort access
		resp = client.GET("/ip").
			WithHeader("Host", f.GetHttpBinServiceFQDN(name)).
			WithHeader("User", "fault-abort").
			Expect()

		if resp.Raw().StatusCode != 555 {
			log.Errorf("status code is %v, please check logs", resp.Raw().StatusCode)
			utils.DebugSleep(time.Hour)

			assert.Equal(ginkgo.GinkgoT(), http.StatusOK, resp.Raw().StatusCode, "status code")
		}
		resp.Status(555)
	})
})
