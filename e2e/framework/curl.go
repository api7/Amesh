package framework

import (
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework/utils"
)

const (
	curlPod = `
apiVersion: v1
kind: Pod
metadata:
  name: consumer
  labels:
    app: consumer
spec:
  containers:
    - name: consumer
      image: {{ .LocalRegistry }}/curlimages/curl
      imagePullPolicy: IfNotPresent
      command: [ "sleep", "1d" ]
`
)

func (f *Framework) CreateCurl() string {
	log.Infof("Create in Mesh Curl")

	artifact, err := utils.RenderManifest(curlPod, f.args)
	assert.Nil(ginkgo.GinkgoT(), err, "render curl template")
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	if err != nil {
		log.Errorf("failed to apply curl pod: %s", err.Error())
	}
	assert.Nil(ginkgo.GinkgoT(), err, "apply curl pod")

	return "consumer"
}

func (f *Framework) WaitForCurlReady() {
	log.Infof("wait for curl ready")
	defer utils.LogTimeTrack(time.Now(), "curl ready (%v)")

	assert.Nil(ginkgo.GinkgoT(), f.WaitForPodsReady("consumer"), "wait for curl ready")
}

func (f *Framework) Curl(name string, args ...string) string {
	log.SkipFramesOnce(1).Infof("Executing: curl -s -i " + strings.Join(args, " "))

	cmd := []string{"exec", name, "-c", "istio-proxy", "--", "curl", "-s", "-i"}
	cmd = append(cmd, args...)
	output, err := k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), f.kubectlOpts, cmd...)

	if err != nil {
		log.Errorf("curl failed: %s", err.Error())
	}
	assert.Nil(ginkgo.GinkgoT(), err, "failed to curl "+args[0])

	return output
}
