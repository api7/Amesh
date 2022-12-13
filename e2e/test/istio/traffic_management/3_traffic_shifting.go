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
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
	f := framework.NewDefaultFramework()
	utils.Case("should be able to split traffic", func() {
		tester := NewVirtualServiceTester(f, []string{"v1", "v2"})

		tester.Create()

		// apply routes
		tester.AddWeightedRoutes("nginx-kind", []string{"v1", "v2"})
		tester.ApplyRoute()

		// validate access count
		v1Counter := 0
		v2Counter := 0
		doRequest := func() {
			output := tester.DoAccess(200, nil, nil, "nginx-kind/status")
			if strings.Contains(output, "ngx-v1") {
				v1Counter++
			} else if strings.Contains(output, "ngx-v2") {
				v2Counter++
			}
		}
		doRequest()
		doRequest()
		doRequest()
		doRequest()

		assert.NotEqual(ginkgo.GinkgoT(), 0, v1Counter, "check v1 accessed")
		assert.NotEqual(ginkgo.GinkgoT(), 0, v2Counter, "check v2 accessed")
		assert.Equal(ginkgo.GinkgoT(), 4, v1Counter+v2Counter, "check total accessed")
	})
})
