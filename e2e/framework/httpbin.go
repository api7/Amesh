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

package framework

import (
	"fmt"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework/utils"
)

const (
	_httpbinManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app: httpbin
spec:
  replicas: {{ .HttpBinReplicas }}
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      containers:
      - name: httpbin
        image: {{ .LocalRegistry }}/kennethreitz/httpbin
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          protocol: TCP
          name: http
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  selector:
    app: httpbin
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`
)

func (f *Framework) newHttpBin() {
	artifact, err := utils.RenderManifest(_httpbinManifest, f.args)
	assert.Nil(ginkgo.GinkgoT(), err, "render httpbin template")

	log.Infof("creating httpbin")
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	assert.Nil(ginkgo.GinkgoT(), err, "apply httpbin")
	f.httpbinReady = false
}

func (f *Framework) waitForHttpbinReady() {
	if f.httpbinReady {
		return
	}

	log.Infof("wait for httpbin ready")
	defer utils.LogTimeTrack(time.Now(), "httpbin ready (%v)")
	assert.Nil(ginkgo.GinkgoT(), f.WaitForDeploymentPodsReady("httpbin"), "wait for httpbin ready")
	f.httpbinReady = true
}

// GetHttpBinServiceFQDN returns the FQDN description for HttpBin service.
func (f *Framework) GetHttpBinServiceFQDN() string {
	f.waitForHttpbinReady()

	return fmt.Sprintf("httpbin.%s.svc.cluster.local", f.namespace)
}
