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
		log.SkipFramesOnce(1)
		errorMsg := fmt.Sprintf("ERROR: %v", err.Error())
		if len(msg) > 0 {
			errorMsg += ", " + fmt.Sprintf(msg[0].(string), msg[1:]...)
		}
		log.Errorf(errorMsg)
	}
	assert.Nil(ginkgo.GinkgoT(), err, msg...)
}
