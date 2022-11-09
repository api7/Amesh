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

package utils

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func RenderManifest(manifest string, data any) (string, error) {
	temp, err := template.New("manifest").Parse(manifest)
	if err != nil {
		return "", err
	}

	artifact := new(strings.Builder)
	if err := temp.Execute(artifact, data); err != nil {
		return "", err
	}
	return artifact.String(), nil
}

// TODO: automatically recover assertion failure
func RetryUntilTimeout(condFunc func() (bool, error)) error {
	return wait.PollImmediate(time.Second*5, time.Second*30, condFunc)
}

func WaitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: time.Second,
		Factor:   1,
		Steps:    60,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}

func WaitForServiceReady(kubectlOpts *k8s.KubectlOptions, ns, name string) (string, error) {
	var (
		svc *corev1.Service
		err error
	)

	condFunc := func() (bool, error) {
		svc, err = k8s.GetServiceE(ginkgo.GinkgoT(), &k8s.KubectlOptions{
			ContextName:   kubectlOpts.ContextName,
			ConfigPath:    kubectlOpts.ConfigPath,
			Namespace:     ns,
			Env:           kubectlOpts.Env,
			InClusterAuth: kubectlOpts.InClusterAuth,
		}, name)
		if err != nil {
			return false, err
		}
		return k8s.IsServiceAvailable(svc), nil
	}

	if err := wait.PollImmediate(1*time.Second, 15*time.Second, condFunc); err != nil {
		return "", err
	}

	return svc.Spec.ClusterIP, nil
}

const (
	failureToleration = 10
)

func WaitForDeploymentPodsReady(kubectlOpts *k8s.KubectlOptions, namespace, name string) error {
	opts := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}

	deploymentFailures := 0
	podFailures := 0
	condFunc := func() (bool, error) {
		if (deploymentFailures + podFailures) >= 2*failureToleration {
			log.Warnf("waiting %s pods... (%v times)", name, deploymentFailures+podFailures)
		} else {
			log.Debugf("waiting %s pods...", name)
		}

		allItems, err := k8s.ListPodsE(ginkgo.GinkgoT(), &k8s.KubectlOptions{
			ContextName:   kubectlOpts.ContextName,
			ConfigPath:    kubectlOpts.ConfigPath,
			Namespace:     namespace,
			Env:           kubectlOpts.Env,
			InClusterAuth: kubectlOpts.InClusterAuth,
		}, opts)
		if err != nil {
			return false, err
		}

		var items []corev1.Pod
		for _, item := range allItems {
			if item.DeletionTimestamp == nil {
				items = append(items, item)
			}
		}

		if len(items) == 0 {
			if deploymentFailures >= failureToleration {
				log.Warnf("no %s pods created (%v times)", name, deploymentFailures)
			} else {
				log.Debugf("no %s pods created", name)
			}
			deploymentFailures++
			clientset, err := k8s.GetKubernetesClientFromOptionsE(ginkgo.GinkgoT(), kubectlOpts)
			if err != nil {
				return false, err
			}

			deployments, err := clientset.AppsV1().Deployments(kubectlOpts.Namespace).List(context.Background(), opts)
			if err != nil {
				return false, err
			}
			if len(deployments.Items) == 0 {
				log.Debugf("no %s deployment created", name)
				return false, nil
			}
			for _, deployment := range deployments.Items {
				for _, cond := range deployment.Status.Conditions {
					if deploymentFailures >= failureToleration {
						log.Warnf("%v: %v", deployment.Name, cond.Message)
					} else {
						log.Debugf("%v: %v", deployment.Name, cond.Message)
					}
				}
			}
			return false, nil
		}
		defer func() { podFailures++ }()
		for _, pod := range items {
			found := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type != corev1.PodReady {
					if podFailures >= failureToleration {
						log.Warnf("pod %s type %s status %s: %s", pod.Name, cond.Type, cond.Status, cond.Message)
					} else {
						log.Debugf("pod %s cond %s", pod.Name, cond.Type)
					}
					continue
				}
				found = true
				if cond.Status != corev1.ConditionTrue {
					if podFailures >= failureToleration {
						log.Warnf("pod %s type %s status %s: %s", pod.Name, cond.Type, cond.Status, cond.Message)
					} else {
						log.Debugf("pod %s status %s", pod.Name, cond.Status)
					}
					return false, nil
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	}
	return WaitExponentialBackoff(condFunc)
}

// GetKubeConfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeConfig() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeConfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeConfig); err != nil && !os.IsNotExist(err) {
			kubeConfig = ""
		}
	}
	return kubeConfig
}

func RandomNamespace() string {
	return fmt.Sprintf("amesh-e2e-%d", time.Now().Nanosecond())
}

func TimeTrack(started time.Time, handler func(duration time.Duration)) {
	elapsed := time.Since(started)
	handler(elapsed)
}

func LogTimeElapsed(format string) func(duration time.Duration) {
	return func(duration time.Duration) {
		log.SkipFramesOnce(1).Infof(color.CyanString(format, duration))
	}
}

func LogTimeTrack(started time.Time, format string) {
	log.SkipFramesOnce(1)
	TimeTrack(started, LogTimeElapsed(format))
}

func LogTimeTrackSkipFrames(started time.Time, format string, frames int) {
	log.SkipFramesOnce(1 + frames)
	TimeTrack(started, LogTimeElapsed(format))
}

func caseWrapper(name string, f func()) func() {
	return func() {
		log.SkipFramesOnce(99) // Skip more frames to ignore file name
		log.Infof(color.GreenString(fmt.Sprintf("=== CASE: %s ===", name)))
		started := time.Now()
		defer func() {
			log.SkipFramesOnce(99)
			log.Infof(color.GreenString(fmt.Sprintf("=== CaseEnd: %v ===", time.Since(started))))
		}()

		f()
	}
}

func Case(name string, f func()) {
	ginkgo.It(name, caseWrapper(name, f), ginkgo.Offset(1))
}

func FCase(name string, f func()) {
	ginkgo.FIt(name, caseWrapper(name, f), ginkgo.Offset(1))
}

func SCase(name string, f func()) {
	ginkgo.It(name, caseWrapper(name, func() {
		ginkgo.Skip("Temporary Skip")
		f()
	}), ginkgo.Offset(1))
}

func IgnorePanic(f func()) {
	defer ginkgo.GinkgoRecover()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		//err, ok := r.(error)
		//if ok {
		//	log.Errorf("ERROR: %s", err.Error())
		//}
	}()
	f()
}

func AssertNil(err error, msg ...interface{}) {
	if err != nil {
		errMsg := err.Error()
		if len(msg) > 0 {
			errMsg = fmt.Sprintf(msg[0].(string), msg[1:]...) + " failed: " + errMsg
		}
		log.SkipFramesOnce(1)
		log.Errorf(errMsg)
		//time.Sleep(time.Hour)
	}
	assert.Nil(ginkgo.GinkgoT(), err, msg...)
}

func PtrOf[T any](v T) *T {
	return &v
}
