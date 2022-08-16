package framework

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestName(t *testing.T) {

	artifact, _ := RenderManifest(_httpbinManifest, &ManifestArgs{
		LocalRegistry:   "10.0.0.2:5000",
		HttpBinReplicas: 1,
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
