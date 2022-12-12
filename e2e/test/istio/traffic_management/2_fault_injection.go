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
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to inject fault: abort", func() {
		tester := NewVirtualServiceTester(f, []string{"v1"})

		tester.Create()

		// create fault routes
		tester.AddRouteToWithAbortFault("nginx-kind", "v1", "User", "fault-abort")
		tester.AddRouteTo("nginx-kind", "v1")
		tester.ApplyRoute()

		// validate
		tester.ValidateAccessible("origin", "nginx-kind/ip")
		// validate fault abort
		tester.ValidateInaccessible(555, "origin", "nginx-kind/ip", "-H", "User: fault-abort")
	})

	utils.Case("should be able to inject fault: delay", func() {
		tester := NewVirtualServiceTester(f, []string{"v1"})

		tester.Create()

		// create fault routes
		tester.AddRouteToWithDelayFault("nginx-kind", "v1", "User", "fault-delay", 10)
		tester.AddRouteTo("nginx-kind", "v1")
		tester.ApplyRoute()

		// validate duration < 10s
		tester.ValidateTimeout(0, time.Second*5, func() {
			tester.ValidateAccessible("origin", "nginx-kind/ip")
		})

		// validate duration > 10s
		tester.ValidateTimeout(time.Second*10, time.Second*14, func() {
			tester.ValidateAccessible("origin", "nginx-kind/ip", "-H", "User: fault-delay")
		})
	})
})
