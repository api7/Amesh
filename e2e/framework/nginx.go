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

	"github.com/api7/amesh/e2e/framework/utils"
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

func (f *Framework) CreateNginxOutsideMeshTo(svc string, waitReady bool) string {
	log.Infof("Create NGINX outside Mesh to " + svc)
	return f.createNginxTo(svc, false, waitReady)
}

func (f *Framework) CreateNginxInMeshTo(svc string, waitReady bool) string {
	log.Infof("Create NGINX in Mesh to " + svc)
	return f.createNginxTo(svc, true, waitReady)
}

func (f *Framework) createNginxTo(svc string, inMesh bool, waitReady bool) string {

	conf := fmt.Sprintf(nginxConfTemplate, svc, svc)

	randomName := fmt.Sprintf("ngx-%d", time.Now().Nanosecond())

	utils.AssertNil(f.CreateConfigMap(randomName, "proxy.conf", conf), "create config map "+randomName)

	args := &renderArgs{
		ManifestArgs: f.args,
		Name:         randomName,
		ConfigMap:    randomName,
		InMesh:       inMesh,
	}
	artifact, err := utils.RenderManifest(nginxTemplate, args)
	utils.AssertNil(err, "render nginx template")
	err = k8s.KubectlApplyFromStringE(ginkgo.GinkgoT(), f.kubectlOpts, artifact)
	if err != nil {
		log.Errorf("failed to apply nginx pod: %s", err.Error())
	}
	utils.AssertNil(err, "apply nginx")

	if waitReady {
		f.WaitForNginxReady(randomName)
	}

	return randomName
}

func (f *Framework) WaitForNginxReady(name string) {
	log.Infof("wait for nginx ready")
	defer utils.LogTimeTrack(time.Now(), "nginx ready (%v)")
	utils.AssertNil(f.WaitForPodsReady(name), "wait for nginx ready")
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
	utils.AssertNil(err, "port-forward nginx tunnel")

	f.tunnels = append(f.tunnels, tunnel)

	return tunnel.Endpoint()
}
