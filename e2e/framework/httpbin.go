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

	"github.com/api7/amesh/e2e/framework/utils"
)

const (
	_httpbinManifest = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  labels:
    app: {{ .Name }}
spec:
  replicas: {{ .HttpBinReplicas }}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
      annotations:
        sidecar.istio.io/inject: "{{ .InMesh }}"
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
  name: {{ .Name }}
spec:
  selector:
    app: {{ .Name }}
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`
)

type httpbinRenderArgs struct {
	*ManifestArgs
	HttpBinReplicas int
	Name            string
	InMesh          bool
}

func (f *Framework) newHttpBin(name string, inMesh bool) {
	artifact, err := utils.RenderManifest(_httpbinManifest, &httpbinRenderArgs{
		ManifestArgs:    f.args,
		HttpBinReplicas: 1,
		Name:            name,
		InMesh:          inMesh,
	})
	utils.AssertNil(err, "render httpbin %s template", name)

	log.Infof("creating httpbin %s", name)
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	utils.AssertNil(err, "apply httpbin %s", name)
}

func (f *Framework) CreateHttpbinInMesh(nameOpt ...string) {
	name := "httpbin"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}

	f.newHttpBin(name, true)
}

func (f *Framework) CreateHttpbinOutsideMesh(nameOpt ...string) {
	name := "httpbin"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}

	f.newHttpBin(name, false)
}

func (f *Framework) WaitForHttpbinReady(nameOpt ...string) {
	name := "httpbin"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}

	log.Infof("wait for httpbin ready")
	defer utils.LogTimeTrack(time.Now(), "httpbin ready (%v)")
	utils.AssertNil(f.WaitForDeploymentPodsReady(name), "wait for httpbin ready")
}

// GetHttpBinServiceFQDN returns the FQDN description for HttpBin service.
func (f *Framework) GetHttpBinServiceFQDN(nameOpt ...string) string {
	name := "httpbin"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}

	return fmt.Sprintf("%s.%s.svc.cluster.local", name, f.namespace)
}
