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

package tester

import (
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
	"github.com/api7/amesh/pkg/apisix"
)

type BaseTester struct {
	f      *framework.Framework
	logger *log.Logger

	HttpbinInside  bool
	HttpbinPodName string

	NginxInside         bool
	NginxDeploymentName string
	NginxReplica        int

	CurlPodName string
}

func NewBaseTester(f *framework.Framework) *BaseTester {
	logger, err := log.NewLogger(
		log.WithLogLevel("info"),
		log.WithSkipFrames(3),
	)
	utils.AssertNil(err, "create logger")
	return &BaseTester{
		f:      f,
		logger: logger,
	}
}

func (t *BaseTester) Create(httpbinInside, nginxInside bool, unavailable ...bool) {
	f := t.f

	t.HttpbinPodName = "httpbin"
	if httpbinInside {
		f.CreateHttpbinInMesh(t.HttpbinPodName)
	} else {
		f.CreateHttpbinOutsideMesh(t.HttpbinPodName)
	}
	t.HttpbinInside = httpbinInside

	svc := f.GetHttpBinServiceFQDN(t.HttpbinPodName)

	available := true
	if len(unavailable) >= 1 {
		available = !unavailable[0]
	}
	if nginxInside {
		if available {
			t.logger.Infof("Creating inside nginx")
			t.NginxDeploymentName = f.CreateNginxInMeshTo(svc, false)
		} else {
			t.logger.Infof("Creating unavailable inside nginx")
			t.NginxDeploymentName = f.CreateUnavailableNginxInMeshTo(svc, false)
		}
	} else {
		if available {
			t.NginxDeploymentName = f.CreateNginxOutsideMeshTo(svc, false)
		} else {
			t.NginxDeploymentName = f.CreateUnavailableNginxOutsideMeshTo(svc, false)
		}
	}
	t.NginxInside = nginxInside
	t.NginxReplica = 1

	t.CurlPodName = f.CreateCurl()

	utils.ParallelRunAndWait(func() {
		if available {
			f.WaitForNginxReady(t.NginxDeploymentName)
		}
	}, func() {
		f.WaitForHttpbinReady(t.HttpbinPodName)
	}, func() {
		f.WaitForCurlReady(t.CurlPodName)
	})
	time.Sleep(time.Second * 5)
}

// ========================
// == Action functions ==
// ========================

func (t *BaseTester) DeleteAllNginxPods() {
	log.Infof(color.BlueString("Delete " + t.NginxDeploymentName + " pods"))
	utils.AssertNil(t.f.DeletePodByLabel(t.f.AppNamespace(), "app="+t.NginxDeploymentName), "delete nginx pods")
	t.f.WaitForNginxReady(t.NginxDeploymentName)
	time.Sleep(time.Second * 5)
}

func (t *BaseTester) DeletePartialNginxPods() {
	log.Infof(color.BlueString("Delete partial " + t.NginxDeploymentName + " pods"))

	podNames, err := t.f.GetDeploymentPodNames(t.f.AppNamespace(), t.NginxDeploymentName)
	utils.AssertNil(err, "get nginx pods")
	assert.Equal(ginkgo.GinkgoT(), true, len(podNames) >= 1, "nginx pods count")

	utils.AssertNil(t.f.DeletePod(t.f.AppNamespace(), podNames[0]), "delete single nginx pod")
	t.f.WaitForNginxReady(t.NginxDeploymentName)
	time.Sleep(time.Second * 5)
}

func (t *BaseTester) MakeNginxInMesh() {
	log.Infof(color.BlueString("Make " + t.NginxDeploymentName + " in mesh"))
	t.f.MakeNginxInsideMesh(t.NginxDeploymentName, true)
	t.f.WaitForNginxReady(t.NginxDeploymentName)

	t.NginxInside = true

	time.Sleep(time.Second * 5)
}

func (t *BaseTester) MakeNginxOutsideMesh() {
	log.Infof(color.BlueString("Make " + t.NginxDeploymentName + " outside mesh"))
	t.f.MakeNginxOutsideMesh(t.NginxDeploymentName, true)
	t.f.WaitForNginxReady(t.NginxDeploymentName)

	t.NginxInside = false

	time.Sleep(time.Second * 5)
}

func (t *BaseTester) MakeNginxUnavailable() {
	log.Infof(color.BlueString("Make " + t.NginxDeploymentName + " unavailable"))
	t.f.MakeNginxUnavailable(t.NginxDeploymentName)

	time.Sleep(time.Second * 5)
}

func (t *BaseTester) MakeNginxAvailable() {
	log.Infof(color.BlueString("Make " + t.NginxDeploymentName + " available"))
	t.f.MakeNginxAvailable(t.NginxDeploymentName, true)

	time.Sleep(time.Second * 5)
}

func (t *BaseTester) ScaleNginx(replica int) {
	log.Infof(color.BlueString("Scale nginx "+t.NginxDeploymentName+" to %v", replica))
	t.f.ScaleNginx(t.NginxDeploymentName, replica, true)
	time.Sleep(time.Second * 5)
}

// ========================
// == Validate functions ==
// ========================

func (t *BaseTester) ValidateProxiedAndAccessible() {
	time.Sleep(time.Second * 3)

	output := t.f.CurlInPod(t.CurlPodName, t.NginxDeploymentName+"/ip")
	assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
	assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
	assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
}

func (t *BaseTester) ValidateNotAccessible() {
	output := t.f.CurlInPod(t.CurlPodName, t.NginxDeploymentName+"/ip")
	assert.NotContains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
}

func (t *BaseTester) ValidateNotProxiedAndAccessible() {
	output := t.f.CurlInPod(t.CurlPodName, t.NginxDeploymentName+"/ip")
	assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
	assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
	assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
}

func (t *BaseTester) ValidateNginxUpstreamNodesCount(count int) []*apisix.Node {
	upstreams := t.f.GetSidecarUpstreams(t.CurlPodName)
	var nginxUpstream *apisix.Upstream
	for _, upstream := range upstreams {
		if strings.Contains(upstream.Name, t.NginxDeploymentName) {
			nginxUpstream = upstream
		}
	}

	assert.NotNil(ginkgo.GinkgoT(), nginxUpstream)
	assert.Equal(ginkgo.GinkgoT(), count, len(nginxUpstream.Nodes), "nginx upstream nodes count")
	return nginxUpstream.Nodes
}
