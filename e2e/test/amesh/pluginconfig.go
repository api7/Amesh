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

package amesh

import (
	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
	"github.com/api7/amesh/e2e/test/tester"
)

var _ = ginkgo.Describe("[amesh-controller functions] AmeshPluginConfig:", func() {
	f := framework.NewDefaultFramework()

	utils.Case("should be able to inject plugins", func() {
		t := tester.NewPluginConfigTester(f, &tester.ResponseRewriteConfig{
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
		t := tester.NewPluginConfigTester(f, &tester.ResponseRewriteConfig{
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
		t := tester.NewPluginConfigTester(f, &tester.ResponseRewriteConfig{
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
