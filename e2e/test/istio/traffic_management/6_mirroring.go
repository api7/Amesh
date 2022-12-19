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

//var _ = ginkgo.Describe("[istio functions] Route Configuration:", func() {
//	f := framework.NewDefaultFramework()
//	utils.Case("should be able to config timeout", func() {
//		v1tester := NewVirtualServiceTester(f, []string{"v1"})
//		v2tester := NewVirtualServiceTester(f, []string{"v2"})
//
//		v1tester.Create()
//		v2tester.Create()
//
//		// apply routes
//		v2tester.AddRouteTo("httpbin-v2", "v2")
//		v2tester.ApplyRoute()
//
//		v1tester.AddRouteToWithMirror("httpbin-v1", "v1", "httpbin-v2", "v2")
//		v1tester.ApplyRoute()
//
//		time.Sleep(time.Hour * 4)
//		// validate access with timeout
//	})
//})
