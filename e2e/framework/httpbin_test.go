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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/amesh/e2e/framework/utils"
)

func TestName(t *testing.T) {

	artifact, _ := utils.RenderManifest(_httpbinManifest, &httpbinRenderArgs{
		ManifestArgs: &ManifestArgs{
			LocalRegistry: "10.0.0.2:5000",
		},
		HttpBinReplicas: 1,
		Name:            "httpbin",
		InMesh:          true,
	})

	assert.Equal(t, `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  labels:
    app: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
  template:
    metadata:
      labels:
        app: httpbin
      annotations:
        sidecar.istio.io/inject: "true"
    spec:
      containers:
      - name: httpbin
        image: 10.0.0.2:5000/kennethreitz/httpbin
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          protocol: TCP
          name: http
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  selector:
    app: httpbin
  ports:
  - name: http
    targetPort: 80
    port: 80
    protocol: TCP
`, artifact)
}
