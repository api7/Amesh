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
	"encoding/json"
	"github.com/api7/amesh/pkg/apisix"
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"

	"github.com/api7/amesh/e2e/framework/utils"
	"github.com/api7/amesh/pkg/amesh/provisioner"
)

const (
	curlPod = `
apiVersion: v1
kind: Pod
metadata:
  name: {{ .Name }}
  labels:
    app: {{ .Name }}
  annotations:
    sidecar.istio.io/inject: "{{ .InMesh }}"
spec:
  containers:
    - name: curl
      image: {{ .LocalRegistry }}/curlimages/curl
      imagePullPolicy: IfNotPresent
      command: [ "sleep", "1d" ]
`
)

type curlRenderArgs struct {
	*ManifestArgs
	Name   string
	InMesh bool
}

func (f *Framework) createCurl(name string, inMesh bool) string {
	if inMesh {
		log.Infof("Create in Mesh Curl %s", name)
	} else {
		log.Infof("Create outside Mesh Curl %s", name)
	}

	artifact, err := utils.RenderManifest(curlPod, &curlRenderArgs{
		ManifestArgs: f.args,
		Name:         name,
		InMesh:       inMesh,
	})
	utils.AssertNil(err, "render curl template for %s", name)
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	if err != nil {
		log.Errorf("failed to apply curl pod %s: %s", name, err.Error())
	}
	utils.AssertNil(err, "apply curl pod %s", name)

	return name
}

func (f *Framework) CreateCurl(nameOpt ...string) string {
	name := "consumer"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}
	return f.createCurl(name, true)
}

func (f *Framework) CreateCurlOutsideMesh(nameOpt ...string) string {
	name := "consumer"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}
	return f.createCurl(name, false)
}

func (f *Framework) WaitForCurlReady(nameOpt ...string) {
	name := "consumer"
	if len(nameOpt) > 0 {
		name = nameOpt[0]
	}
	log.Infof("wait for curl ready")
	defer utils.LogTimeTrack(time.Now(), "curl ready (%v)")

	utils.AssertNil(f.WaitForPodsReady(name), "wait for curl pod %s ready", name)
}

func (f *Framework) CurlInPod(name string, args ...string) string {
	log.SkipFramesOnce(1)
	log.Infof("Executing: curl -s -i " + strings.Join(args, " "))

	cmd := []string{"exec", name, "-c", "istio-proxy", "--", "curl", "-s", "-i"}
	cmd = append(cmd, args...)
	output, err := k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), f.kubectlOpts, cmd...)

	if err != nil {
		log.Errorf("curl failed: %s", err.Error())
	}
	utils.AssertNil(err, "failed to curl "+args[0])

	return output
}

func (f *Framework) queryStatusServer(podName, api string) string {
	cmd := []string{"exec", podName, "-c", "istio-proxy", "--", "curl", "-s", "localhost:9999/" + api}
	output, err := k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), f.kubectlOpts, cmd...)

	log.SkipFramesOnce(1)
	log.Infof("Executing: kubectl " + strings.Join(cmd, " "))
	if err != nil {
		log.Errorf("get sidecar %s status failed: %s", podName, err.Error())
	}
	utils.AssertNil(err, "failed to get sidecar %s status", podName)

	return output
}

func (f *Framework) GetSidecarStatus(podName string) *provisioner.XdsProvisionerStatus {
	output := f.queryStatusServer(podName, "status")

	var status provisioner.XdsProvisionerStatus
	err := json.Unmarshal([]byte(output), &status)
	utils.AssertNil(err, "failed to unmarshal sidecar %s status", podName)

	return &status
}

func (f *Framework) GetSidecarRoutes(podName string) map[string]*apisix.Route {
	output := f.queryStatusServer(podName, "routes")

	var routes map[string]*apisix.Route
	err := json.Unmarshal([]byte(output), &routes)
	utils.AssertNil(err, "failed to unmarshal sidecar %s routes", podName)

	return routes
}

func (f *Framework) GetSidecarUpstreams(podName string) map[string]*apisix.Upstream {
	output := f.queryStatusServer(podName, "upstreams")

	var upstreams map[string]*apisix.Upstream
	err := json.Unmarshal([]byte(output), &upstreams)
	utils.AssertNil(err, "failed to unmarshal sidecar %s upstreams", podName)

	return upstreams
}
