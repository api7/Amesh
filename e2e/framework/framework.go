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
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/api7/amesh/controller/apis/client/clientset/versioned"
	"github.com/api7/amesh/e2e/framework/ameshcontroller"
	"github.com/api7/amesh/e2e/framework/controlplane"
	"github.com/api7/amesh/e2e/framework/utils"
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

	AmeshClient clientset.Interface

	e2eHome string

	t           testing.TestingT
	kubectlOpts *k8s.KubectlOptions
	tunnels     []*k8s.Tunnel

	cp        controlplane.ControlPlane
	amesh     ameshcontroller.AmeshController
	namespace string

	httpbinReady bool
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
		opts.KubeConfig = utils.GetKubeConfig()
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", opts.KubeConfig)
	utils.AssertNil(err, "build kubeconfig")
	ameshClient, err := clientset.NewForConfig(cfg)
	utils.AssertNil(err, "build Amesh client")

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
		AmeshClient: ameshClient,
		opts:        opts,
		args:        args,

		e2eHome: e2eHome,

		t:         ginkgo.GinkgoT(),
		namespace: utils.RandomNamespace(),
	}
	f.kubectlOpts = &k8s.KubectlOptions{
		ConfigPath: opts.KubeConfig,
		Namespace:  f.namespace,
	}

	ginkgo.BeforeEach(f.beforeEach)
	ginkgo.AfterEach(f.afterEach)

	return f
}

func (f *Framework) cpNamespace() string {
	return f.namespace + "-cp"
}

func (f *Framework) initFramework() {
	f.namespace = utils.RandomNamespace()
	f.kubectlOpts = &k8s.KubectlOptions{
		ConfigPath: f.kubectlOpts.ConfigPath,
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
		AmeshImage:  f.args.LocalRegistry + "/" + f.opts.AmeshControllerImage,
		ChartsPath:  filepath.Join(f.e2eHome, "../controller/charts/amesh-controller"),
	})
}

func (f *Framework) deploy() {
	log.Infof(color.CyanString("=== Installing ==="))
	defer utils.LogTimeTrack(time.Now(), "=== Installation Available (%v) ===")

	e := utils.NewParallelExecutor("")
	e.Add(func() {
		log.Infof("installing istio")
		defer utils.LogTimeTrack(time.Now(), "istio installed (%v)")
		utils.AssertNil(f.cp.Deploy(), "deploy istio")
	}, func() {
		log.Infof("wait for istio ready")
		defer utils.LogTimeTrack(time.Now(), "istio ready (%v)")
		utils.AssertNil(f.cp.WaitForReady(), "wait istio")
	}, func() {
		f.newHttpBin()
		f.waitForHttpbinReady()
	})
	e.Add(func() {
		log.Infof("installing amesh-controller")
		defer utils.LogTimeTrack(time.Now(), "amesh-controller installed (%v)")
		utils.AssertNil(f.amesh.Deploy(), "deploy amesh-controller")
	}, func() {
		log.Infof("wait for amesh-controller ready")
		defer utils.LogTimeTrack(time.Now(), "amesh-controller ready (%v)")
		utils.AssertNil(f.amesh.WaitForReady(), "wait amesh-controller")
	})
	e.Wait()
}

func (f *Framework) beforeEach() {
	log.Infof(color.CyanString("=== Environment Initializing ==="))
	defer utils.LogTimeTrack(time.Now(), "=== Environment Initialized (%v) ===")

	f.initFramework()

	e := utils.NewParallelExecutor("")
	e.Add(func() {
		log.Infof("creating namespace " + f.namespace)
		defer log.Infof("created namespace " + f.namespace)
		f.WaitForNamespaceDeletion(f.namespace)

		err := k8s.CreateNamespaceWithMetadataE(ginkgo.GinkgoT(), f.kubectlOpts, metav1.ObjectMeta{
			Name: f.namespace,
			Labels: map[string]string{
				"istio-injection": "enabled",
			},
		})
		utils.AssertNil(err, "create namespace "+f.namespace)

		//utils.AssertNil(f.cp.InjectNamespace(f.namespace), "inject namespace")
	})
	e.Add(func() {
		log.Infof("creating namespace " + f.cpNamespace())
		defer log.Infof("created namespace " + f.cpNamespace())
		f.WaitForNamespaceDeletion(f.cpNamespace())

		err := k8s.CreateNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace())
		utils.AssertNil(err, "create namespace "+f.cpNamespace())
	})
	e.Wait()

	f.deploy()
}

func (f *Framework) afterEach() {
	log.Infof(color.CyanString("=== Environment Cleaning ==="))
	//started := time.Now()
	// TODO: this sometimes appears after the [SLOW TEST] mark, don't know why
	defer utils.LogTimeTrack(time.Now(), "=== Environment Cleaned (%v) ===")

	defer func() {
		log.Infof("delete namespace " + f.cpNamespace())
		utils.AssertNil(k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.cpNamespace()), "delete namespace "+f.cpNamespace())

		//utils.LogTimeTrack(started, "=== Environment Cleaned (%v) ===")
	}()
	// Should delete the control plane components explicitly since there are some cluster scoped
	// resources, which will be intact if we just only delete the cp namespace.

	e := utils.NewParallelExecutor("")
	e.Add(func() {
		log.Infof("delete namespace " + f.namespace)
		utils.AssertNil(k8s.DeleteNamespaceE(ginkgo.GinkgoT(), f.kubectlOpts, f.namespace), "delete namespace "+f.namespace)
	})
	e.Add(func() {
		utils.IgnorePanic(func() {
			log.Infof("delete istio")
			// FIXME: make sure we handle cluster-range resource correctly since users may interrupt when deleting
			utils.AssertNil(f.cp.Uninstall(), "uninstall istio")
		})
	})
	e.Add(func() {
		utils.IgnorePanic(func() {
			log.Infof("delete amesh-controller")
			utils.AssertNil(f.amesh.Uninstall(), "uninstall amesh-controller")
		})
	})
	e.Add(func() {
		for _, tunnel := range f.tunnels {
			tunnel.Close()
		}
		f.tunnels = nil
	})

	e.Wait()
}
