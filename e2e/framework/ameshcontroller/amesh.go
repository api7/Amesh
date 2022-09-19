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

package ameshcontroller

import (
	"bytes"
	"os/exec"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_helm = "helm"
)

// AmeshController represents the amesh controller in e2e test cases.
type AmeshController interface {
	// Type returns the amesh controller type.
	Type() string
	// Namespace fetches the deployed namespace of amesh controller components.
	Namespace() string
	// Deploy deploys the amesh controller.
	Deploy() error
	// Uninstall uninstalls the amesh controller.
	Uninstall() error
	// Addr returns the address to communicate with the amesh controller for fetching
	// configuration changes.
	Addr() string
}

type amesh struct {
	install         *exec.Cmd
	uninstall       *exec.Cmd
	installStderr   *bytes.Buffer
	uninstallStderr *bytes.Buffer

	logger *log.Logger

	options   *AmeshOptions
	clusterIP string
}

// AmeshOptions contains options to customize Amesh controller.
type AmeshOptions struct {
	// KubeConfig is the kube config file path.
	KubeConfig string
	// AmeshImage is the image of amesh-controller
	AmeshImage string
	// Namespace is the target namespace to install amesh-controller.
	Namespace string
	// ChartsPath is a directory that contains charts for amesh-controller.
	ChartsPath string

	KubectlOpts *k8s.KubectlOptions
}

// NewAmeshController creates an amesh controller.
func NewAmeshController(opts *AmeshOptions) AmeshController {
	kc := opts.KubeConfig
	image := opts.AmeshImage

	install := exec.Command(_helm, "install", "amesh-controller", "--namespace", opts.Namespace, "--kubeconfig", kc,
		"--set", "controller.image="+image,
		opts.ChartsPath,
	)
	uninstall := exec.Command(_helm, "uninstall", "amesh-controller", "--namespace", opts.Namespace, "--kubeconfig", kc)

	discoveryStderr := bytes.NewBuffer(nil)
	cleanupDiscoveryStderr := bytes.NewBuffer(nil)

	install.Stderr = discoveryStderr
	uninstall.Stderr = cleanupDiscoveryStderr

	logger, err := log.NewLogger(
		log.WithContext("amesh"),
		log.WithLogLevel("error"),
	)
	if err != nil {
		assert.Nil(ginkgo.GinkgoT(), err)

		return nil
	}

	return &amesh{
		logger:          logger,
		install:         install,
		uninstall:       uninstall,
		options:         opts,
		installStderr:   discoveryStderr,
		uninstallStderr: cleanupDiscoveryStderr,
	}
}

func (cp *amesh) Namespace() string {
	return cp.options.Namespace
}

func (cp *amesh) Type() string {
	return "amesh"
}

func (cp *amesh) Addr() string {
	return "grpc://" + cp.clusterIP + ":15810"
}

func (cp *amesh) Deploy() error {
	err := cp.install.Run()
	if err != nil {
		log.Errorw("failed to run amesh-controller install command",
			zap.String("command", cp.install.String()),
			zap.String("stderr", cp.installStderr.String()),
		)
		return err
	}

	ctlOpts := &k8s.KubectlOptions{
		ConfigPath: cp.options.KubeConfig,
		Namespace:  cp.options.Namespace,
	}

	var (
		svc *corev1.Service
	)

	condFunc := func() (bool, error) {
		svc, err = k8s.GetServiceE(ginkgo.GinkgoT(), ctlOpts, "amesh-controller")
		if err != nil {
			return false, err
		}
		return k8s.IsServiceAvailable(svc), nil
	}

	if err := wait.PollImmediate(3*time.Second, 15*time.Second, condFunc); err != nil {
		return err
	}

	cp.clusterIP = svc.Spec.ClusterIP
	return nil
}

func (cp *amesh) Uninstall() error {
	err := cp.uninstall.Run()
	if err != nil {
		log.Errorw("failed to uninstall amesh-controller",
			zap.Error(err),
			zap.String("stderr", cp.uninstallStderr.String()),
		)
		return err
	}
	return nil
}
