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
	"net/http"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to config timeout", func() {
		tester := NewVirtualServiceTester(f, []string{"v1"})

		tester.Create()

		// Case 1, timeout
		// apply routes
		tester.AddRouteToWithDelayFault("httpbin-kind", "v1", "User", "fault-delay", 10)
		tester.AddRouteTo("httpbin-kind", "v1")
		tester.AddRouteToWithTimeout("nginx-kind", "v1", 5)
		tester.ApplyRoute()

		// validate access with timeout
		tester.ValidateTimeout(0, time.Second*5, func() {
			tester.ValidateAccessible("origin", "nginx-kind/ip")
		})
		// validate longer than 5s (toleration timeout) but shorter than 10s (actual service timeout)
		tester.ValidateTimeout(time.Second*5, time.Second*8, func() {
			tester.ValidateInaccessible(http.StatusGatewayTimeout, "origin", "nginx-kind/ip", "-H", "User: fault-delay")
		})

		// Case 2, tolerate timeout
		// apply routes
		tester.ClearRoute("nginx-kind")
		tester.AddRouteToWithTimeout("nginx-kind", "v1", 12)
		tester.ApplyRoute()

		// validate accessible
		tester.ValidateTimeout(0, time.Second*5, func() {
			tester.ValidateAccessible("origin", "nginx-kind/ip")
		})
		// validate longer than 10s (actual service timeout) but shorter than 12s (toleration timeout)
		tester.ValidateTimeout(time.Second*10, time.Second*12, func() {
			tester.ValidateAccessible("origin", "nginx-kind/ip", "-H", "User: fault-delay")
		})
	})
})
