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

package base

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
	apisixutils "github.com/api7/amesh/pkg/apisix/utils"
)

var _ = ginkgo.Describe("[deployment lifecycle]", func() {
	f := framework.NewDefaultFramework()

	utils.Case("inside mesh curl should be able to access inside mesh after pod deletion", func() {
		// Inside Curl -> (Inside NGINX -> Outside HTTPBIN)
		// Delete Inside NGINX and make it restart

		t := NewBaseTester(f)

		t.Create(false, true)
		t.ValidateProxiedAndAccessible()

		// Delete all pods
		t.DeleteAllNginxPods()
		t.ValidateProxiedAndAccessible()

		// Scale to 2
		t.ScaleNginx(2)
		beforeDeleteNodes := t.ValidateNginxUpstreamNodesCount(2)
		t.ValidateProxiedAndAccessible()

		// Delete a pod. We have 2 replicas now,
		// so we can't verify the function by access the upstream,
		// we need to verify the nodes are changed to ensure the update are handled.
		t.DeletePartialNginxPods()
		afterDeleteNodes := t.ValidateNginxUpstreamNodesCount(2)

		assert.Equal(ginkgo.GinkgoT(), false, apisixutils.IsSameNodes(beforeDeleteNodes, afterDeleteNodes), "upstream nodes changed")
	})

	utils.Case("inside mesh curl should be able to access outside->inside mesh", func() {
		// Inside Curl -> (Outside NGINX -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX -> Outside HTTPBIN)

		t := NewBaseTester(f)

		t.Create(false, false)
		t.ValidateNotProxiedAndAccessible()

		// Make the ngx inside mesh
		t.MakeNginxInMesh()
		t.ValidateProxiedAndAccessible()
	})

	utils.Case("inside mesh curl should be able to access inside->outside mesh", func() {
		// Inside Curl -> (Inside NGINX -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Outside NGINX -> Outside HTTPBIN)

		t := NewBaseTester(f)

		t.Create(false, true)
		t.ValidateProxiedAndAccessible()

		// Make the ngx inside mesh
		t.MakeNginxOutsideMesh()
		t.ValidateNotProxiedAndAccessible()
	})

	utils.Case("should be able to handle replica changes", func() {
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Replica: 2) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)

		t := NewBaseTester(f)

		t.Create(false, true)
		t.ValidateProxiedAndAccessible()

		// Verify upstream.nodes length
		t.ValidateNginxUpstreamNodesCount(1)
		t.ValidateProxiedAndAccessible()

		// Scale to 2 and verify upstream.nodes length
		t.ScaleNginx(2)
		t.ValidateNginxUpstreamNodesCount(2)
		t.ValidateProxiedAndAccessible()

		// Scale to 1 and verify upstream.nodes length
		t.ScaleNginx(1)
		t.ValidateNginxUpstreamNodesCount(1)
		t.ValidateProxiedAndAccessible()
	})

	utils.Case("should be able to handle deployment become unavailable and recovered", func() {
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Unavailable) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)

		t := NewBaseTester(f)

		t.Create(false, true)
		t.ValidateProxiedAndAccessible()
		beforeNodes := t.ValidateNginxUpstreamNodesCount(1)

		// make the deployment unavailable
		t.MakeNginxUnavailable()
		unavailableNodes := t.ValidateNginxUpstreamNodesCount(1)
		// k8s won't delete old pod since the new pod unavailable, so the nodes is the same
		assert.Equal(ginkgo.GinkgoT(), true, apisixutils.IsSameNodes(beforeNodes, unavailableNodes), "upstream nodes should unchanged")
		// still accessible because the old pod alive
		t.ValidateProxiedAndAccessible()

		// make the deployment available
		t.MakeNginxAvailable()
		afterNodes := t.ValidateNginxUpstreamNodesCount(1)
		// the deployment recovered but the config is the same, so the old pod kept
		assert.Equal(ginkgo.GinkgoT(), true, apisixutils.IsSameNodes(beforeNodes, afterNodes), "upstream nodes changed")
		t.ValidateProxiedAndAccessible()
	})

	utils.Case("should be able to handle deployment started as unavailable and recovered", func() {
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Unavailable) -> Outside HTTPBIN)
		// ->
		// Inside Curl -> (Inside NGINX (Replica: 1) -> Outside HTTPBIN)

		t := NewBaseTester(f)

		t.Create(false, true, true)
		beforeNodes := t.ValidateNginxUpstreamNodesCount(0)
		t.ValidateNotAccessible()

		// make the deployment available
		t.MakeNginxAvailable()
		afterNodes := t.ValidateNginxUpstreamNodesCount(1)
		assert.Equal(ginkgo.GinkgoT(), false, apisixutils.IsSameNodes(beforeNodes, afterNodes), "upstream nodes changed")
		t.ValidateProxiedAndAccessible()
	})
})
