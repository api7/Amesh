package controlplane

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	_helm = "helm"
)

type istio struct {
	base                   *exec.Cmd
	discovery              *exec.Cmd
	cleanupBase            *exec.Cmd
	cleanupDiscovery       *exec.Cmd
	baseStderr             *bytes.Buffer
	cleanupBaseStderr      *bytes.Buffer
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
	// ChartsPath is a directory that contains charts for Istio.
	// The first element should be the chart for istio-base and
	// the second is the istio-discovery.
	ChartsPath []string

	KubectlOpts *k8s.KubectlOptions
}

// NewIstioControlPlane creates an istio control plane.
func NewIstioControlPlane(opts *IstioOptions) ControlPlane {
	kc := opts.KubeConfig
	image := opts.IstioImage

	base := exec.Command(_helm,
		"install", "istio-base", "--namespace", opts.Namespace, "--kubeconfig", kc,
		"--set", "global.istioNamespace="+opts.Namespace,

		opts.ChartsPath[0])
	deleteBase := exec.Command(_helm, "uninstall", "istio-base", "--namespace", opts.Namespace, "--kubeconfig", kc)
	discovery := exec.Command(_helm, "install", "istio-discovery", "--namespace", opts.Namespace, "--kubeconfig", kc,
		"--set", "pilot.image="+image,
		"--set", "global.imagePullPolicy=IfNotPresent",
		"--set", "global.proxy.privileged=true",
		"--set", "global.proxy_init.image="+opts.SidecarInitImage,
		"--set", "global.proxy.image="+opts.SidecarImage,
		"--set", "global.istioNamespace="+opts.Namespace,
		"--set", "global.defaultResources.requests.cpu=1000m",
		opts.ChartsPath[1],
	)
	deleteDiscovery := exec.Command(_helm, "uninstall", "istio-discovery", "--namespace", opts.Namespace, "--kubeconfig", kc)

	baseStderr := bytes.NewBuffer(nil)
	cleanupBaseStderr := bytes.NewBuffer(nil)
	discoveryStderr := bytes.NewBuffer(nil)
	cleanupDiscoveryStderr := bytes.NewBuffer(nil)

	base.Stderr = baseStderr
	deleteBase.Stderr = cleanupBaseStderr
	discovery.Stderr = discoveryStderr
	deleteDiscovery.Stderr = cleanupDiscoveryStderr

	logger, err := log.NewLogger(
		log.WithContext("istio"),
		log.WithLogLevel("error"),
	)
	if err != nil {
		assert.Nil(ginkgo.GinkgoT(), err)

		return nil
	}

	return &istio{
		logger:                 logger,
		base:                   base,
		discovery:              discovery,
		cleanupBase:            deleteBase,
		cleanupDiscovery:       deleteDiscovery,
		options:                opts,
		baseStderr:             baseStderr,
		cleanupBaseStderr:      cleanupBaseStderr,
		discoveryStderr:        discoveryStderr,
		cleanupDiscoveryStderr: cleanupDiscoveryStderr,
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

func (cp *istio) Deploy() error {
	err := cp.base.Run()
	if err != nil {
		log.Errorw("failed to run istio-base install command",
			zap.String("command", cp.base.String()),
			zap.Error(err),
			zap.String("stderr", cp.baseStderr.String()),
		)
		return err
	}
	err = cp.discovery.Run()
	if err != nil {
		log.Errorw("failed to run istio-discovery install command",
			zap.String("command", cp.discovery.String()),
			zap.String("stderr", cp.discoveryStderr.String()),
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
		svc, err = k8s.GetServiceE(ginkgo.GinkgoT(), ctlOpts, "istiod")
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

func (cp *istio) Uninstall() error {
	err := cp.cleanupDiscovery.Run()
	if err != nil {
		log.Errorw("failed to uninstall istio-discovery",
			zap.Error(err),
			zap.String("stderr", cp.cleanupDiscoveryStderr.String()),
		)
		return err
	}
	err = cp.cleanupBase.Run()
	if err != nil {
		log.Errorw("failed to uninstall istio-base",
			zap.Error(err),
			zap.String("stderr", cp.cleanupBaseStderr.String()),
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
