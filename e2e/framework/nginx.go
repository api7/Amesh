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
	"strings"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/api7/amesh/e2e/framework/utils"
)

const (
	nginxTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  labels:
    appKind: nginx
    app: {{ .Name }}
    version: {{ .Version }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      appKind: nginx
      app: {{ .Name }}
      version: {{ .Version }}
  template:
    metadata:
      name: {{ .Name }}
      labels:
        appKind: nginx
        app: {{ .Name }}
        version: {{ .Version }}
      annotations:
        sidecar.istio.io/inject: "{{ .InMesh }}"
    spec:
      volumes:
      - name: conf
        configMap:
          name: {{ .ConfigMapName }}
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
    protocol: TCP
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .ConfigMapName }}
data:
  proxy.conf: |
    server {
        listen 80;
        server_name test.com;
        location /status {
            return 200 '{{ .Name }} to {{ .ProxyService }}';
            default_type text/plain;
        }
        location / {
            proxy_pass http://{{ .ProxyService }};
            proxy_set_header Connection "";
            proxy_set_header Host {{ .ProxyService }};
            proxy_pass_request_headers on;
            proxy_http_version 1.1;
        }
    }
`

	nginxService = `
apiVersion: v1
kind: Service
metadata:
  name: nginx-kind
spec:
  selector:
    appKind: nginx
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`
)

func (f *Framework) CreateNginxService() {
	utils.AssertNil(f.ApplyResourceFromString(nginxService), "create nginx service")
}

type NginxArgs struct {
	*ManifestArgs
	Name          string
	ConfigMapName string
	InMesh        bool
	Version       string
	Replicas      int
	ProxyService  string
}

func (f *Framework) getNginxArgs(name string) *NginxArgs {
	if args, ok := f.appArgs[name]; ok {
		if nginxArgs, ok := args.(*NginxArgs); ok {
			return nginxArgs
		} else {
			ginkgo.Fail(fmt.Sprintf("failed to convert config to nginx args: %s", name), 1)
		}
	} else {
		ginkgo.Fail(fmt.Sprintf("failed to get nginx args: %s", name), 1)
	}

	return nil
}

func (f *Framework) CreateNginxOutsideMeshTo(svc string, waitReady bool, nameOpt ...string) string {
	log.Infof("Create NGINX outside Mesh to " + svc)
	return f.createNginxTo(f.args.LocalRegistry, svc, false, waitReady, nameOpt...)
}

func (f *Framework) CreateNginxInMeshTo(svc string, waitReady bool, nameOpt ...string) string {
	log.Infof("Create NGINX in Mesh to " + svc)
	return f.createNginxTo(f.args.LocalRegistry, svc, true, waitReady, nameOpt...)
}

func (f *Framework) CreateUnavailableNginxOutsideMeshTo(svc string, waitReady bool, nameOpt ...string) string {
	log.Infof("Create unavailable NGINX outside Mesh to " + svc)
	return f.createNginxTo("unavailable-registry", svc, false, waitReady, nameOpt...)
}

func (f *Framework) CreateUnavailableNginxInMeshTo(svc string, waitReady bool, nameOpt ...string) string {
	log.Infof("Create unavailable NGINX in Mesh to " + svc)
	return f.createNginxTo("unavailable-registry", svc, true, waitReady, nameOpt...)
}

func (f *Framework) MakeNginxInsideMesh(name string, waitReady bool) {
	args := f.getNginxArgs(name)
	args.InMesh = true
	f.appArgs[name] = args

	f.applyNginx(args, waitReady)
}

func (f *Framework) MakeNginxOutsideMesh(name string, waitReady bool) {
	args := f.getNginxArgs(name)
	args.InMesh = false
	f.appArgs[name] = args

	f.applyNginx(args, waitReady)
}

func (f *Framework) MakeNginxUnavailable(name string) {
	args := f.getNginxArgs(name)
	newArgs := &NginxArgs{
		ManifestArgs: &ManifestArgs{
			LocalRegistry: "unknown-registry",
		},
		Name:          args.Name,
		ConfigMapName: args.ConfigMapName,
		InMesh:        args.InMesh,
		Version:       args.Version,
		Replicas:      args.Replicas,
		ProxyService:  args.ProxyService,
	}

	f.applyNginx(newArgs, false)
}

func (f *Framework) MakeNginxAvailable(name string, waitReady bool) {
	args := f.getNginxArgs(name)
	args.LocalRegistry = f.args.LocalRegistry
	f.applyNginx(args, waitReady)
}

func (f *Framework) ScaleNginx(name string, replicas int, waitReady bool) {
	args := f.getNginxArgs(name)
	args.Replicas = replicas
	f.appArgs[name] = args

	f.applyNginx(args, waitReady)
}

func (f *Framework) createNginxTo(registry, svc string, inMesh bool, waitReady bool, nameOpt ...string) string {
	randomName := fmt.Sprintf("ngx-%d", time.Now().Nanosecond())
	if len(nameOpt) > 0 {
		randomName = nameOpt[0]
	}

	version := "v1"
	if strings.HasSuffix(randomName, "-v2") {
		version = "v2"
	} else if strings.HasSuffix(randomName, "-v3") {
		version = "v3"
	}
	args := &NginxArgs{
		ManifestArgs: &ManifestArgs{
			LocalRegistry: registry,
		},
		Name:          randomName,
		ConfigMapName: randomName,
		InMesh:        inMesh,
		Version:       version,
		Replicas:      1,
		ProxyService:  svc,
	}
	f.appArgs[randomName] = args
	f.applyNginx(args, waitReady)

	return randomName
}

// applyNginx doesn't record the appArgs
func (f *Framework) applyNginx(args *NginxArgs, waitReady bool) {
	artifact, err := utils.RenderManifest(nginxTemplate, args)
	utils.AssertNil(err, "render nginx template")
	err = f.ApplyResourceFromString(artifact)
	if err != nil {
		log.Errorf("failed to apply nginx pod: %s", err.Error())
	}
	utils.AssertNil(err, "apply nginx")

	if waitReady {
		f.WaitForNginxReady(args.Name)
	}

	return
}

func (f *Framework) WaitForNginxReady(name string) {
	log.Infof("wait for nginx %v ready", name)
	defer utils.LogTimeTrack(time.Now(), name+" ready (%v)")
	utils.AssertNil(f.WaitForDeploymentPodsReady(name), "wait for nginx ready")
}

// NewHTTPClientToNginx creates a http client which sends requests to
// nginx.
func (f *Framework) NewHTTPClientToNginx(name string) (*httpexpect.Expect, *k8s.Tunnel) {
	endpoint, tunnel := f.buildTunnelToNginx(name)
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
	}), tunnel
}

func (f *Framework) buildTunnelToNginx(name string) (string, *k8s.Tunnel) {
	localPort := rand.IntnRange(12000, 12100)
	tunnel := k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, name, localPort, 80)
	err := tunnel.ForwardPortE(ginkgo.GinkgoT())
	utils.AssertNil(err, "port-forward nginx tunnel to "+name)

	f.tunnels = append(f.tunnels, tunnel)

	log.Infof("Create local tunnel to %v at %v", name, tunnel.Endpoint())
	return tunnel.Endpoint(), tunnel
}
