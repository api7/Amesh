# Amesh Plugin Config

本文假设镜像准备已如 [Demo](./demo.md) 中所述准备完全。

## 安装 Istio

使用如下命令安装 Istio：

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

将 `test` namespace 标记为需要注入：

```bash
kubectl create ns test
kubectl label ns test istio-injection=enabled
```

## 准备 Amesh Controller 镜像

在项目根目录执行以下命令：

```bash
export REGISTRY=$YOUR_REGISTRY
make build-amesh-controller-image
```

这将构建出 amesh-controller:latest，将其推送到镜像源上（若有）。

## 部署 Amesh Controller

使用如下命令部署 Amesh Controller 与 CRD：

```
cd controller
helm install amesh-controller -n istio-system ./charts/amesh-controller
kubectl apply -k ./config/crd/
```

## 确认安装情况

执行如下命令查看控制面安装情况：

```
kubectl -n istio-system get pods
```

输出应该类似：

```
NAME                                READY   STATUS    RESTARTS   AGE
amesh-controller-76b6d579f5-vnlt7   1/1     Running   0          62s
istiod-7869755557-vtx9r             1/1     Running   0          7m25s
```

## 部署测试程序

```bash
# 在 Istio 目录下执行
kubectl -n test apply -f samples/bookinfo/platform/kube/bookinfo.yaml
kubectl -n test run consumer --image curlimages/curl --image-pull-policy IfNotPresent --command sleep 1d
```

测试是否能够正常访问：

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" | grep -o "<title>.*</title>"
```

输出应该是：

```
<title>Simple Bookstore App</title>
```

## 部署测试插件配置

```bash
# 在 Amesh 目录下执行
kubectl -n test apply -f controller/samples/*
```

该示例配置如下：

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

观察 Amesh Controller 是否同步到了该配置：

```bash
kubectl -n test get ampc ameshpluginconfig-sample -o jsonpath-as-json='{.status.conditions[0]}'
```

其输出应该类似：

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

这表明 Amesh Controller 成功同步到了该配置。

## 测试插件配置

再次尝试访问测试负载：

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage"
```

输出应该包含：

```
Via: APISIX
Server: APISIX/2.13.0
X-Header: Rewrite

BODY_REWRITE
```

## 更新插件配置

将示例 AmeshPluginConfig 修改为如下，移除 Body 配置：

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

再次观察 status：

```bash
kubectl -n test get ampc ameshpluginconfig-sample -o jsonpath-as-json='{.status.conditions[0]}'
```

其输出的 `observedGeneration` 字段应该比上一次增加了 1：

```json
[
    {
        "message": "Sync Success",
        "reason": "Reconciled",
        "observedGeneration": 2,
        "status": "True",
        "type": "Sync"
    }
]
```

再次请求测试负载：

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" | grep -o "<title>.*</title>"
```

输出应该是：

```
<title>Simple Bookstore App</title>
```

```bash
kubectl -n test exec -it -c istio-proxy consumer -- curl -i -XGET "http://productpage:9080/productpage" | grep "X-Header"
```

输出应该是：

```
X-Header: Rewrite
```
