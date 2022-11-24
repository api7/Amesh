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

package controlplane

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/api7/amesh/e2e/framework/utils"
)

var (
	_helm = "helm"
)

type istio struct {
	discovery              *exec.Cmd
	cleanupDiscovery       *exec.Cmd
	baseStderr             *bytes.Buffer
	discoveryStderr        *bytes.Buffer
	cleanupDiscoveryStderr *bytes.Buffer

	logger *log.Logger

	options     *IstioOptions
	installCmds [][]string
	clusterIP   string
}

// IstioOptions contains options to customize Istio control plane.
type IstioOptions struct {
	// KubeConfig is the kube config file path.
	KubeConfig string
	// IstioImage is the image of Istiod (pilot)
	IstioImage string
	// SidecarInitImage is the sidecar init image
	SidecarInitImage string
	// SidecarImage is the sidecar image
	SidecarImage string
	// Namespace is the target namespace to install istio.
	Namespace string
	// ChartsPath is a directory that contains charts for Amesh.
	ChartsPath string

	KubectlOpts *k8s.KubectlOptions
}

// NewIstioControlPlane creates an istio control plane.
func NewIstioControlPlane(opts *IstioOptions) ControlPlane {
	logger, err := log.NewLogger(
		log.WithContext("istio"),
		log.WithLogLevel("error"),
	)
	if err != nil {
		utils.AssertNil(err)

		return nil
	}

	return &istio{
		logger:  logger,
		options: opts,
	}
}

func (cp *istio) Namespace() string {
	return cp.options.Namespace
}

func (cp *istio) Type() string {
	return "istio"
}

func (cp *istio) Addr() string {
	return "grpc://" + cp.clusterIP + ":15010"
}

func (cp *istio) initCmd() {
	opts := cp.options

	kc := opts.KubeConfig
	image := opts.IstioImage

	discovery := exec.Command(_helm, "install", "amesh", "--namespace", opts.Namespace, "--kubeconfig", kc,
		"--set", "pilot.image="+image,
		"--set", "global.imagePullPolicy=IfNotPresent",
		"--set", "global.proxy.privileged=true",
		"--set", "global.proxy_init.image="+opts.SidecarInitImage,
		"--set", "global.proxy.image="+opts.SidecarImage,
		"--set", "global.istioNamespace="+opts.Namespace,
		"--set", "global.defaultResources.requests.cpu=1000m",
		opts.ChartsPath,
	)
	deleteDiscovery := exec.Command(_helm, "uninstall", "istio-discovery", "--namespace", opts.Namespace, "--kubeconfig", kc)

	baseStderr := bytes.NewBuffer(nil)
	discoveryStderr := bytes.NewBuffer(nil)
	cleanupDiscoveryStderr := bytes.NewBuffer(nil)

	discovery.Stderr = discoveryStderr
	deleteDiscovery.Stderr = cleanupDiscoveryStderr

	cp.discovery = discovery
	cp.cleanupDiscovery = deleteDiscovery
	cp.baseStderr = baseStderr
	cp.discoveryStderr = discoveryStderr
	cp.cleanupDiscoveryStderr = cleanupDiscoveryStderr
}

func (cp *istio) Deploy() error {
	cp.initCmd()

	err := cp.discovery.Run()
	if err != nil {
		log.Errorf("failed to run istio-discovery install command")
		log.Errorf("command: %s", cp.discovery.String())
		log.Errorf("err: %s", err.Error())
		log.Errorf("stderr: %s", cp.discoveryStderr.String())
		return err
	}
	return nil
}

func (cp *istio) WaitForReady() error {
	var err error
	cp.clusterIP, err = utils.WaitForServiceReady(cp.options.KubectlOpts, cp.options.Namespace, "istiod")
	if err != nil {
		return err
	}

	err = utils.WaitForDeploymentPodsReady(cp.options.KubectlOpts, cp.options.Namespace, "istiod")
	return err
}

func (cp *istio) Uninstall() error {
	err := cp.cleanupDiscovery.Run()
	if err != nil {
		log.Errorw("failed to uninstall istio-discovery",
			zap.Error(err),
			zap.String("stderr", cp.cleanupDiscoveryStderr.String()),
		)
		return err
	}
	return nil
}

func (cp *istio) InjectNamespace(ns string) error {
	client, err := k8s.GetKubernetesClientFromOptionsE(ginkgo.GinkgoT(), cp.options.KubectlOpts)
	if err != nil {
		return err
	}
	obj, err := client.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if obj.Labels == nil {
		obj.Labels = make(map[string]string)
	}
	obj.Labels["istio-injection"] = "enabled"
	if _, err := client.CoreV1().Namespaces().Update(context.TODO(), obj, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}
