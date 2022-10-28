package tester

import (
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/fatih/color"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework"
	"github.com/api7/amesh/e2e/framework/utils"
	"github.com/api7/amesh/pkg/apisix"
)

type BaseTester struct {
	f      *framework.Framework
	logger *log.Logger

	HttpbinInside  bool
	HttpbinPodName string

	NginxInside         bool
	NginxDeploymentName string
	NginxReplica        int

	CurlPodName string
}

func NewBaseTester(f *framework.Framework) *BaseTester {
	logger, err := log.NewLogger(
		log.WithLogLevel("info"),
		log.WithSkipFrames(3),
	)
	utils.AssertNil(err, "create logger")
	return &BaseTester{
		f:      f,
		logger: logger,
	}
}

func (t *BaseTester) Create(httpbinInside, nginxInside bool) {
	f := t.f

	t.HttpbinPodName = "httpbin"
	if httpbinInside {
		f.CreateHttpbinInMesh(t.HttpbinPodName)
	} else {
		f.CreateHttpbinOutsideMesh(t.HttpbinPodName)
	}
	t.HttpbinInside = httpbinInside

	svc := f.GetHttpBinServiceFQDN(t.HttpbinPodName)
	if nginxInside {
		t.NginxDeploymentName = f.CreateNginxInMeshTo(svc, false)
	} else {
		t.NginxDeploymentName = f.CreateNginxOutsideMeshTo(svc, false)
	}
	t.NginxInside = nginxInside
	t.NginxReplica = 1

	t.CurlPodName = f.CreateCurl()

	utils.ParallelRunAndWait(func() {
		f.WaitForNginxReady(t.NginxDeploymentName)
	}, func() {
		f.WaitForHttpbinReady(t.HttpbinPodName)
	}, func() {
		f.WaitForCurlReady(t.CurlPodName)
	})
	time.Sleep(time.Second * 5)
}

func (t BaseTester) DeleteAllNginxPods() {
	log.Infof(color.BlueString("Delete " + t.NginxDeploymentName + " pods"))
	utils.AssertNil(t.f.DeletePodByLabel(t.f.AppNamespace(), "app="+t.NginxDeploymentName), "delete nginx pods")
	t.f.WaitForNginxReady(t.NginxDeploymentName)
	time.Sleep(time.Second * 5)
}

func (t BaseTester) DeletePartialNginxPods() {
	log.Infof(color.BlueString("Delete " + t.NginxDeploymentName + " pods"))

	podNames, err := t.f.GetDeploymentPodNames(t.f.AppNamespace(), t.NginxDeploymentName)
	utils.AssertNil(err, "get nginx pods")
	assert.Equal(ginkgo.GinkgoT(), true, len(podNames) >= 1, "nginx pods count")

	utils.AssertNil(t.f.DeletePod(t.f.AppNamespace(), podNames[0]), "delete single nginx pod")
	t.f.WaitForNginxReady(t.NginxDeploymentName)
	time.Sleep(time.Second * 5)
}

func (t BaseTester) MakeNginxInMesh() {
	log.Infof(color.BlueString("Make " + t.NginxDeploymentName + " in mesh"))
	t.f.MakeNginxInsideMesh(t.NginxDeploymentName, true)
	t.f.WaitForNginxReady(t.NginxDeploymentName)

	t.NginxInside = true

	time.Sleep(time.Second * 5)
}

func (t BaseTester) MakeNginxOutsideMesh() {
	log.Infof(color.BlueString("Make " + t.NginxDeploymentName + " outside mesh"))
	t.f.MakeNginxOutsideMesh(t.NginxDeploymentName, true)
	t.f.WaitForNginxReady(t.NginxDeploymentName)

	t.NginxInside = false

	time.Sleep(time.Second * 5)
}

func (t *BaseTester) ValidateProxiedAndAccessible() {
	output := t.f.CurlInPod(t.CurlPodName, t.NginxDeploymentName+"/ip")
	assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
	assert.Contains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
	assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
}

func (t *BaseTester) ValidateNotProxiedAndAccessible() {
	output := t.f.CurlInPod(t.CurlPodName, t.NginxDeploymentName+"/ip")
	assert.Contains(ginkgo.GinkgoT(), output, "200 OK", "make sure it works properly")
	assert.NotContains(ginkgo.GinkgoT(), output, "Via: APISIX", "make sure it works properly")
	assert.Contains(ginkgo.GinkgoT(), output, "origin", "make sure it works properly")
}

func (t *BaseTester) ValidateNginxUpstreamNodesCount(count int) []*apisix.Node {
	upstreams := t.f.GetSidecarUpstreams(t.CurlPodName)
	var nginxUpstream *apisix.Upstream
	for _, upstream := range upstreams {
		if strings.Contains(upstream.Name, t.NginxDeploymentName) {
			nginxUpstream = upstream
		}
	}

	assert.NotNil(ginkgo.GinkgoT(), nginxUpstream)
	assert.Equal(ginkgo.GinkgoT(), count, len(nginxUpstream.Nodes), "nginx upstream nodes count")
	return nginxUpstream.Nodes
}
