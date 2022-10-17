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

package tester

import (
	"github.com/api7/gopkg/pkg/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
)

type ReconnectTester struct {
	f      *framework.Framework
	logger *log.Logger

	curlPod string
}

func NewReconnectTester(f *framework.Framework) *ReconnectTester {
	logger, err := log.NewLogger(
		log.WithLogLevel("info"),
		log.WithSkipFrames(3),
	)
	utils.AssertNil(err, "create logger")
	return &ReconnectTester{
		f: f,

		logger: logger,
	}
}

func (t *ReconnectTester) Create() {
	t.curlPod = t.f.CreateCurl()
	t.f.WaitForCurlReady()
}

func (t *ReconnectTester) ValidateStatusAmeshIsOK() {
	condFunc := func() (bool, error) {
		return t.f.GetSidecarStatus(t.curlPod).AmeshConnected, nil
	}
	_ = utils.WaitExponentialBackoff(condFunc) // ignore timeout error

	assert.Equal(ginkgo.GinkgoT(), true, t.f.GetSidecarStatus(t.curlPod).AmeshConnected, "amesh should connected")
	assert.Equal(ginkgo.GinkgoT(), true, t.f.GetSidecarStatus(t.curlPod).AmeshProvisionerReady, "amesh provisioner should ready")
}

func (t *ReconnectTester) DeleteAmeshController() {
	t.logger.Infof("delete amesh-controller pods...")
	utils.AssertNil(t.f.DeletePodByLabel(t.f.ControlPlaneNamespace(), "app=amesh-controller"), "delete amesh controller pods")

	utils.AssertNil(t.f.WaitForDeploymentPodsReady("amesh-controller", t.f.ControlPlaneNamespace()), "wait for amesh-controller pods")
}
