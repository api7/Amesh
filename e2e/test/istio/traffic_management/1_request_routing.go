// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package traffic_management

import (
	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to route requests", func() {
		tester := NewVirtualServiceTester(f, []string{"v1", "v2"})

		// deploy apps
		tester.Create()

		// Case 1
		// Basic config to v1
		tester.AddRouteTo("nginx-kind", "v1")
		tester.ApplyRoute()

		// Validate normal access
		tester.ValidateSingleVersionAccess([]string{"ngx-v1"}, []string{"ngx-v2"}, "nginx-kind/status")

		// Case 2
		// Basic config to v2
		tester.ClearRoute("nginx-kind")
		tester.AddRouteTo("nginx-kind", "v2")
		tester.ApplyRoute()

		// Validate normal access
		tester.ValidateSingleVersionAccess([]string{"ngx-v2"}, []string{"ngx-v1"}, "nginx-kind/status")

		// Case 3
		// Match Conditions
		tester.ClearRoute("nginx-kind")
		tester.AddRouteToIfHeaderIs("nginx-kind", "v2", "X-USER", "v2User")
		tester.AddRouteTo("nginx-kind", "v1")
		tester.ApplyRoute()

		// Validate normal access
		tester.ValidateSingleVersionAccess([]string{"ngx-v1"}, []string{"ngx-v2"}, "nginx-kind/status")
		// Validate matched access
		tester.ValidateSingleVersionAccess([]string{"ngx-v2"}, []string{"ngx-v1"}, "nginx-kind/status", "-H", "X-USER: v2User")
	})
})
