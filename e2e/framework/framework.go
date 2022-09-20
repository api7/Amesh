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
	"os"
	"path/filepath"
	"strings"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework/ameshcontroller"
	"github.com/api7/amesh/e2e/framework/controlplane"
)

func init() {
	gomega.RegisterFailHandler(ginkgo.Fail)
}

type ManifestArgs struct {
	// Public arguments to render manifests.
	LocalRegistry string

	HttpBinReplicas int
}

type Framework struct {
	opts *Options
	args *ManifestArgs

	t           testing.TestingT
	kubectlOpts *k8s.KubectlOptions
	tunnels     []*k8s.Tunnel

	cp        controlplane.ControlPlane
	amesh     ameshcontroller.AmeshController
	namespace string
}

type Options struct {
	KubeConfig string

	AmeshControllerImage   string
	ControlPlaneImage      string
	SidecarInitImage       string
	SidecarImage           string
	ControlPlaneChartsPath []string
}

func NewDefaultFramework() *Framework {
	opts := &Options{}
	return NewFramework(opts)
}

func NewFramework(opts *Options) *Framework {
	e2eHome := os.Getenv("AMESH_E2E_HOME")

	if opts.KubeConfig == "" {
		opts.KubeConfig = GetKubeConfig()
	}
	if opts.ControlPlaneImage == "" {
		opts.ControlPlaneImage = "istio/pilot:1.13.1"
	}
	if opts.SidecarInitImage == "" {
		opts.SidecarInitImage = "amesh-iptables:dev"
	}
	if opts.SidecarImage == "" {
		opts.SidecarImage = "amesh-sidecar:dev"
	}
	if len(opts.ControlPlaneChartsPath) == 0 {
		opts.ControlPlaneChartsPath = []string{
			filepath.Join(e2eHome, "charts/base"),
			filepath.Join(e2eHome, "charts/istio-discovery"),
		}
	}
	if opts.AmeshControllerImage == "" {
		opts.AmeshControllerImage = "amesh-controller:latest"
	}

	args := &ManifestArgs{
		LocalRegistry:   os.Getenv("REGISTRY"),
		HttpBinReplicas: 1,
	}
	if args.LocalRegistry == "" {
		args.LocalRegistry = "localhost:5000"
	}

	f := &Framework{
		opts: opts,
		args: args,

		t:         ginkgo.GinkgoT(),
		namespace: randomNamespace(),
	}
	f.kubectlOpts = &k8s.KubectlOptions{
		ConfigPath: opts.KubeConfig,
		Namespace:  f.namespace,
	}

	istioOpts := &controlplane.IstioOptions{
		KubeConfig:       f.opts.KubeConfig,
		Namespace:        f.cpNamespace(),
		KubectlOpts:      f.kubectlOpts,
		IstioImage:       f.args.LocalRegistry + "/" + f.opts.ControlPlaneImage,
		SidecarInitImage: f.args.LocalRegistry + "/" + f.opts.SidecarInitImage,
		SidecarImage:     f.args.LocalRegistry + "/" + f.opts.SidecarImage,
		ChartsPath:       f.opts.ControlPlaneChartsPath,
	}
	f.cp = controlplane.NewIstioControlPlane(istioOpts)

	f.amesh = ameshcontroller.NewAmeshController(&ameshcontroller.AmeshOptions{
		KubeConfig:  f.opts.KubeConfig,
		Namespace:   f.cpNamespace(),
		KubectlOpts: f.kubectlOpts,
		AmeshImage:  f.args.LocalRegistry + "/" + opts.AmeshControllerImage,
		ChartsPath:  filepath.Join(e2eHome, "../controller/charts/amesh-controller"),
	})

	ginkgo.BeforeEach(f.beforeEach)
	ginkgo.AfterEach(f.afterEach)

	return f
}

func (f *Framework) cpNamespace() string {
	//return f.namespace + "-cp"

	// TODO: FIXME currently our dynamic lib use hard-coded istio-system
	return "istio-system"
}

func (f *Framework) deploy() {
	log.Infof("installing istio")
	assert.Nil(ginkgo.GinkgoT(), f.cp.Deploy(), "deploy istio")
	assert.Nil(ginkgo.GinkgoT(), f.cp.InjectNamespace(f.namespace), "inject namespace")

	log.Infof("installing amesh-controller")
	assert.Nil(ginkgo.GinkgoT(), f.amesh.Deploy(), "deploy amesh-controller")

	f.newHttpBin()
}

func (f *Framework) beforeEach() {
	f.WaitForNamespaceDeletion(f.namespace)
	f.WaitForNamespaceDeletion(f.cpNamespace())

	err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace)
	assert.Nil(ginkgo.GinkgoT(), err, "create namespace "+f.namespace)
	err = k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace())
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Nil(ginkgo.GinkgoT(), err, "create namespace "+f.cpNamespace())
	}
	f.deploy()
}

func (f *Framework) afterEach() {
	log.Infof("cleanup...")

	err := k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace)
	assert.Nil(ginkgo.GinkgoT(), err, "delete namespace "+f.namespace)

	// Should delete the control plane components explicitly since there are some cluster scoped
	// resources, which will be intact if we just only delete the cp namespace.

	assert.Nil(ginkgo.GinkgoT(), f.cp.Uninstall(), "uninstall istio")
	assert.Nil(ginkgo.GinkgoT(), f.amesh.Uninstall(), "uninstall amesh-controller")
	assert.Nil(ginkgo.GinkgoT(), k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace()), "delete namespace "+f.cpNamespace())

	for _, tunnel := range f.tunnels {
		tunnel.Close()
	}
	f.tunnels = nil
}
