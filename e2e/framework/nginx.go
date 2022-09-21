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
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nginxConfTemplate = `
server {
	listen 80;
	server_name test.com;
	location / {
		proxy_pass http://%s;
		proxy_set_header Host %s;
		proxy_http_version 1.1;
		proxy_set_header Connection "";
	}
}
`

	nginxTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      name: {{ .Name }}
      labels:
        app: {{ .Name }}
      annotations:
        sidecar.istio.io/inject: "{{ .InMesh }}"
    spec:
      volumes:
      - name: conf
        configMap:
          name: {{ .ConfigMap }}
      containers:
      - name: nginx
        image: {{ .LocalRegistry }}/nginx:1.19.3
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          protocol: TCP
          name: http
        volumeMounts:
        - name: conf
          mountPath: /etc/nginx/conf.d
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
spec:
  selector:
    app: {{ .Name }}
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP`
)

type renderArgs struct {
	*ManifestArgs
	Name      string
	ConfigMap string
	InMesh    bool
}

func (f *Framework) CreateNginxOutsideMeshTo(svc string) string {
	log.Infof("Create NGINX outside Mesh to " + svc)
	return f.createNginxTo(svc, false)
}

func (f *Framework) CreateNginxInMeshTo(svc string) string {
	log.Infof("Create NGINX in Mesh to " + svc)
	return f.createNginxTo(svc, true)
}

func (f *Framework) createNginxTo(svc string, inMesh bool) string {

	conf := fmt.Sprintf(nginxConfTemplate, svc, svc)

	randomName := fmt.Sprintf("ngx-%d", time.Now().Nanosecond())

	assert.Nil(ginkgo.GinkgoT(), f.CreateConfigMap(randomName, "proxy.conf", conf), "create config map "+randomName)

	args := &renderArgs{
		ManifestArgs: f.args,
		Name:         randomName,
		ConfigMap:    randomName,
		InMesh:       inMesh,
	}
	artifact, err := RenderManifest(nginxTemplate, args)
	assert.Nil(ginkgo.GinkgoT(), err, "render nginx template")
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	assert.Nil(ginkgo.GinkgoT(), err, "apply nginx")

	assert.Nil(ginkgo.GinkgoT(), f.waitUntilAllNginxPodsReady(randomName), "wait for nginx ready")

	return randomName
}

func (f *Framework) waitUntilAllNginxPodsReady(name string) error {
	opts := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}
	condFunc := func() (bool, error) {
		items, err := k8s.ListPodsE(ginkgo.GinkgoT(), f.kubectlOpts, opts)
		if err != nil {
			return false, err
		}
		if len(items) == 0 {
			log.Debugf("no nginx pods created")
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

// NewHTTPClientToNginx creates a http client which sends requests to
// nginx.
func (f *Framework) NewHTTPClientToNginx(name string) *httpexpect.Expect {
	endpoint := f.buildTunnelToNginx(name)
	u := url.URL{
		Scheme: "http",
		Host:   endpoint,
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: http.DefaultTransport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Reporter: httpexpect.NewAssertReporter(httpexpect.NewAssertReporter(ginkgo.GinkgoT())),
	})
}

func (f *Framework) buildTunnelToNginx(name string) string {
	tunnel := k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, name, 12384, 80)
	err := tunnel.ForwardPortE(ginkgo.GinkgoT())
	assert.Nil(ginkgo.GinkgoT(), err, "port-forward nginx tunnel")

	f.tunnels = append(f.tunnels, tunnel)

	return tunnel.Endpoint()
}
