package framework

import (
	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
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

func (f *Framework) CreateCurl() {
	log.Infof("Create in Mesh Curl")

	artifact, err := RenderManifest(curlPod, f.args)
	assert.Nil(ginkgo.GinkgoT(), err, "render curl template")
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	assert.Nil(ginkgo.GinkgoT(), err, "apply curl pod")

	assert.Nil(ginkgo.GinkgoT(), f.waitUntilAllCurlPodsReady("consumer"), "wait for curl ready")

	return
}

func (f *Framework) Curl(args ...string) string {
	log.Infof("Executing: curl -s -i " + strings.Join(args, " "))

	cmd := []string{"exec", "consumer", "-c", "istio-proxy", "--", "curl", "-s", "-i"}
	cmd = append(cmd, args...)
	output, err := k8s.RunKubectlAndGetOutputE(ginkgo.GinkgoT(), f.kubectlOpts, cmd...)

	if err != nil {
		log.Errorw(err.Error())
	}
	assert.Nil(ginkgo.GinkgoT(), err, "failed to curl "+args[0])

	return output
}

func (f *Framework) waitUntilAllCurlPodsReady(name string) error {
	opts := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			log.Debugf("no consumer pods created")
			return false, nil
		}
		for _, pod := range items {
			found := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type != corev1.PodReady {
					continue
				}
				found = true
				if cond.Status != corev1.ConditionTrue {
					return false, nil
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}
