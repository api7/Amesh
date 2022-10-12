# Amesh Plugin Config

This documentation assumes that all required images are prepared as [Demo](./demo.md) described.

## Install Istio

To install Istio, run:

```bash
export YOUR_REGISTRY="10.0.0.20:5000"
export ISTIO_RELEASE=1.13.1
helm install istio-base --namespace istio-system ./base
helm install istio-discovery \
--namespace istio-system \
--set pilot.image=istio/pilot:$ISTIO_RELEASE \
  --set global.proxy.privileged=true \
  --set global.proxy_init.hub="$YOUR_REGISTRY" \
  --set global.proxy_init.image=amesh-iptables \
  --set global.proxy_init.tag=dev \
  --set global.proxy.hub="$YOUR_REGISTRY" \
  --set global.proxy.image=amesh-sidecar \
  --set global.proxy.tag=dev \
  --set global.imagePullPolicy=IfNotPresent \
  --set global.hub="docker.io/istio" \
  --set global.tag="$ISTIO_RELEASE" \
  ./istio-control/istio-discovery
```

Add label for `test` namespace:

```bash
kubectl create ns test
kubectl label ns test istio-injection=enabled
```

## Prepare Amesh Controller Image

Run the following command in the project root directory:

```bash
export REGISTRY=$YOUR_REGISTRY
make build-amesh-controller-image
```

This will build image `amesh-controller:latest`, push it into the registry if needed.

## Install Amesh Controller

Run the following commands to deploy Amesh Controller and CRDs:

```
cd controller
helm install amesh-controller -n istio-system ./charts/amesh-controller
kubectl apply -k ./config/crd/
```

## Check Installation

Run the following command to display the control plane status:

```
kubectl -n istio-system get pods
```

The output should like:

```
NAME                                READY   STATUS    RESTARTS   AGE
amesh-controller-76b6d579f5-vnlt7   1/1     Running   0          62s
istiod-7869755557-vtx9r             1/1     Running   0          7m25s
```

## Install Test Workload

```bash
# Run in the Istio directory
kubectl -n test apply -f samples/bookinfo/platform/kube/bookinfo.yaml
kubectl -n test run consumer --image curlimages/curl --image-pull-policy IfNotPresent --command sleep 1d
```

Run the following command to test if the workload is accessible,

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" | grep -o "<title>.*</title>"
```

The output should like:

```
<title>Simple Bookstore App</title>
```

## Add Plugin Config

```bash
# Run in the Amesh directory
kubectl -n test apply -f controller/samples/*
```

This will add the following yaml config to your K8s cluster, 

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: AmeshPluginConfig
metadata:
  name: ameshpluginconfig-sample
spec:
  plugins:
    - name: response-rewrite
      type: ""
      config: '{"body": "BODY_REWRITE", "headers": {"X-Header":"Rewrite"}}'
```

Check if the Amesh Controller has synced the config,

```bash
kubectl -n test get ampc ameshpluginconfig-sample -o jsonpath-as-json='{.status.conditions[0]}'
```

The output should like:

```json
[
    {
        "message": "Sync Success",
        "reason": "Reconciled",
        "observedGeneration": 1,
        "status": "True",
        "type": "Sync"
    }
]
```

This indicates that the Amesh Controller has successfully synchronized the plugin configurations from the CRDs.

## Test Plugin Config Functionality

Try to access the workload again:

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage"
```

The output should contain the following content:

```
Via: APISIX
Server: APISIX/2.13.0
X-Header: Rewrite

BODY_REWRITE
```

## Update Plugin Config

Edit and apply the sample AmeshPluginConfig as following, this removes the `body` rule,

```yaml
apiVersion: apisix.apache.org/v1alpha1
kind: AmeshPluginConfig
metadata:
  name: ameshpluginconfig-sample
spec:
  plugins:
  - config: '{"headers": {"X-Header":"Rewrite"}}'
    name: response-rewrite
    type: ""
```

Check the Amesh Controller status again,

```bash
kubectl -n test get ampc ameshpluginconfig-sample -o jsonpath-as-json='{.status.conditions[0]}'
```

The output `observedGeneration` field should be greater than the previous one by 1,

```js
[
    {
        "message": "Sync Success",
        "reason": "Reconciled",
        "observedGeneration": 2, // Here
        "status": "True",
        "type": "Sync"
    }
]
```

Access the workload again:

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" | grep -o "<title>.*</title>"
```

The output should like:

```
<title>Simple Bookstore App</title>
```

This indicates that the `body` rule has been removed as we want.
Now we should check if the header rule is still working:

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" | grep "X-Header"
```

The output should like:

```
X-Header: Rewrite
```
