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

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

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
	namespace string
}

type Options struct {
	KubeConfig string

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
	assert.Nil(ginkgo.GinkgoT(), f.cp.Deploy(), "deploy istio")
	assert.Nil(ginkgo.GinkgoT(), f.cp.InjectNamespace(f.namespace), "inject namespace")

	f.newHttpBin()
}

func (f *Framework) beforeEach() {
	err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace)
	assert.Nil(ginkgo.GinkgoT(), err, "create namespace "+f.namespace)
	err = k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace())
	assert.Nil(ginkgo.GinkgoT(), err, "create namespace "+f.cpNamespace())
	f.deploy()
}

func (f *Framework) afterEach() {
	err := k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace)
	assert.Nil(ginkgo.GinkgoT(), err, "delete namespace "+f.namespace)

	// Should delete the control plane components explicitly since there are some cluster scoped
	// resources, which will be intact if we just only delete the cp namespace.
	err = f.cp.Uninstall()
	assert.Nil(ginkgo.GinkgoT(), err, "uninstall istio")

	err = k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace())
	assert.Nil(ginkgo.GinkgoT(), err, "delete namespace "+f.cpNamespace())

	for _, tunnel := range f.tunnels {
		tunnel.Close()
	}
}
