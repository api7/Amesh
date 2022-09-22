package amesh

import (
	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
	"github.com/api7/amesh/e2e/test/tester"
)

var _ = ginkgo.Describe("[amesh-controller functions]", func() {
	f := framework.NewDefaultFramework()

	utils.Case("should be able to inject plugins", func() {
		t := tester.NewTester(f, &tester.ResponseRewriteConfig{
			Body: "BODY_REWRITE",
			Headers: map[string]string{
				"X-Header": "Rewrite",
			},
		})

		t.Create()
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()
	})

	utils.Case("should be able to update plugins", func() {
		t := tester.NewTester(f, &tester.ResponseRewriteConfig{
			Headers: map[string]string{
				"X-Header": "Rewrite",
			},
		})

		t.Create()
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()

		t.UpdateConfig(&tester.ResponseRewriteConfig{
			Body: "BODY_REWRITE",
			Headers: map[string]string{
				"X-Header": "RewriteChanged",
			},
		})
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()

		t.UpdateConfig(&tester.ResponseRewriteConfig{
			Body: "BODY_REWRITE",
			Headers: map[string]string{
				"X-Header-Changed": "HeaderChanged",
			},
		})
		t.ValidateInMeshNginxProxyAccess()
		t.ValidateInMeshCurlAccess()
	})

	utils.Case("should be able to delete plugins", func() {
		t := tester.NewTester(f, &tester.ResponseRewriteConfig{
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
