# Amesh Demo

## Get Images

To get a pre-built image.：

```bash
docker pull api7/amesh-iptables:v0.0.2
docker pull api7/amesh-apisix:v0.0.2
```

If you want to build images yourself, please refer to the following section.

### Build Amesh Images

In the project root directory, run the following commands,

```bash
make build-amesh-iptables-image
make build-amesh-so-image
```

These two commands will package the `amesh-iptables:v0.0.2` image used by the init container, and the dynamic library `libxds.so` that will be used by APISIX.

Build the base APISIX image for Amesh, which will be used later,

```bash
make build-apisix-image
```

### Build APISIX Image for Amesh

Get the latest APISIX code to get the xds support,

```
git clone https://github.com/apache/apisix.git
```

Get `libxds.so` from the image we built above,

```
cd apisix
docker run -it --rm --entrypoint sh -v $(pwd):/pwd amesh-so:dev -c "cp libxds.so /pwd"
```

Modify ``conf/config.yaml`` to use the following configuration, the rest of the configuration can be customized according to your needs.

```yaml
apisix:
  config_center: xds
  extra_lua_cpath: "/usr/local/apisix/?.so;" 
  node_listen:
    - 19080
  enable_admin: false
nginx_config:
  http_configuration_snippet: |
    server {
          access_log on;
          listen 19081 reuseport;
          location / {
              proxy_http_version 1.1;
              proxy_set_header Connection "";
              proxy_set_header Host $http_host;
              proxy_pass http://$connection_original_dst;
              add_header Via APISIX always;
          }
    }
plugins:
  - cors
  - prometheus
  - traffic-split
  - proxy-rewrite
```

Build the Amesh APISIX sidecar image using the following command,

```bash
make build-amesh-sidecar-image
```

This will build the amesh-sidecar:dev image, which will be the image that sidecar will use.

## Run the Demo

### Install Istio

This document uses Istio v1.13.1.

```bash
git clone git@github.com:istio/istio.git 
git checkout 1.13.1
```

Replace the default injection template of Istio with the Amesh configuration.

```bash
cd istio/manifests/charts
cp /path/to/amesh/manifest/injection-template.yaml ./istio-control/istio-discovery/files
```

Push image `amesh-iptables:dev` and `apisix:custom` to your registry, assuming the registry address is in the `YOUR_REGISTRY` environment variable,

```bash
export YOUR_REGISTRY="10.0.0.20:5000"
```

To install Istio, run：

```bash
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
  --set global.proxy.image=amesh-apisix \
  --set global.proxy.tag=custom \
  --set global.imagePullPolicy=IfNotPresent \
  --set global.hub="docker.io/istio" \
  --set global.tag="$ISTIO_RELEASE" \
  ./istio-control/istio-discovery
```

Enable injection webhook on `default` namespace,

```bash
kubectl label ns default istio-injection=enabled
```

### Install the Workload

Install the bookinfo workload of Istio,

```bash
kubectl apply -f ../../samples/bookinfo/platform/kube/bookinfo.yaml
```

Verify that it runs without errors,

```bash
kubectl get pods
```

The output should be similar to the following,

```
NAME                              READY   STATUS    RESTARTS   AGE
details-v1-79f774bdb9-pprhg       2/2     Running   0          39m
productpage-v1-6b746f74dc-zvjnn   2/2     Running   0          39m
ratings-v1-b6994bb9-hx2zn         2/2     Running   0          39m
reviews-v1-545db77b95-xjgfl       2/2     Running   0          39m
reviews-v2-7bf8c9648f-98cp8       2/2     Running   0          39m
reviews-v3-84779c7bbc-hklzg       2/2     Running   0          39m
```

### Test Service Mesh Functionality

Run the test pod,

```bash
kubectl run consumer --image curlimages/curl --image-pull-policy IfNotPresent --command sleep 1d
```

Verify that the test pod can access bookinfo properly,

```bash
kubectl exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" |  grep -o "<title>.*</title>"
```

The output should be as follows,

```
<title>Simple Bookstore App</title>
```
