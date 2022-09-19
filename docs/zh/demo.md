# Amesh Demo

## 镜像编译

获取已打包好的镜像：

```bash
docker pull api7/amesh-iptables:v0.0.2
docker pull api7/amesh-apisix:v0.0.2
```

如需自行编译，可参考下文。

### 编译 Amesh 相关镜像

在项目根目录执行：

```bash
make build-amesh-iptables-image
make build-amesh-so-image
```

这两行命令会打包 init container 所使用的 `amesh-iptables:v0.0.2` 镜像，以及后续 APISIX 将用到的动态链接库 `libxds.so`。

构建 Amesh 专用的基础 APISIX 镜像，后续将会使用到：

```bash
make build-apisix-image
```

### 编译 APISIX 镜像

获取最新 APISIX 代码，以得到最新的 xds 支持：

```
git clone https://github.com/apache/apisix.git
```

从容器中获取 `libxds.so`：

```
cd apisix
docker run -it --rm --entrypoint sh -v $(pwd):/pwd amesh-so:dev -c "cp libxds.so /pwd"
```

修改 `conf/config.yaml`，使用下面的配置，其余配置可以自行更改：

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

构建最终使用的 Sidecar 镜像：

```bash
make build-amesh-sidecar-image
```

这将构建出 amesh-sidecar:dev 镜像，作为最终使用的 sidecar。

## 运行 Demo

### 安装 Istio

范例使用 1.13.1 版本的 Istio。

```bash
git clone git@github.com:istio/istio.git 
git checkout 1.13.1
```

使用 Amesh 的配置替换 Istio 的默认配置：

```bash
cd istio/manifests/charts
cp /path/to/amesh/manifest/injection-template.yaml ./istio-control/istio-discovery/files
```

将镜像 `amesh-iptables:dev` 与 `apisix:custom` 推到镜像源上，假设该镜像源在 `YOUR_REGISTRY` 环境变量中，例如：

```bash
export YOUR_REGISTRY="10.0.0.20:5000"
```

安装 Istio：

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
  --set global.proxy.image=amesh-sidecar \
  --set global.proxy.tag=dev \
  --set global.imagePullPolicy=IfNotPresent \
  --set global.hub="docker.io/istio" \
  --set global.tag="$ISTIO_RELEASE" \
  ./istio-control/istio-discovery
```

将 `default` namespace 标记为需要注入：

```bash
kubectl label ns default istio-injection=enabled
```

### 安装负载

安装 Istio 自带的负载程序：

```bash
kubectl apply -f ../../samples/bookinfo/platform/kube/bookinfo.yaml
```

验证运行无误：

```bash
kubectl get pods
```

输出应该类似：

```
NAME                              READY   STATUS    RESTARTS   AGE
details-v1-79f774bdb9-pprhg       2/2     Running   0          39m
productpage-v1-6b746f74dc-zvjnn   2/2     Running   0          39m
ratings-v1-b6994bb9-hx2zn         2/2     Running   0          39m
reviews-v1-545db77b95-xjgfl       2/2     Running   0          39m
reviews-v2-7bf8c9648f-98cp8       2/2     Running   0          39m
reviews-v3-84779c7bbc-hklzg       2/2     Running   0          39m
```

### 测试

运行测试负载：

```bash
kubectl run consumer --image curlimages/curl --image-pull-policy IfNotPresent --command sleep 1d
```

测试是否能够正常访问：

```bash
kubectl exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" |  grep -o "<title>.*</title>"
```

输出应该是：

```
<title>Simple Bookstore App</title>
```
