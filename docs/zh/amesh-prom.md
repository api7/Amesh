# Amesh 结合 Prometheus 的使用


## Amesh 的指标暴露

Istio 可以通过 `prometheus.io` 注解来控制 sidecar 指标是否被 Prometheus 获取。

该选项默开启，Amesh 也将使用该方案。

Sidecar 会将指标以纯文本的形式暴露在 `/stats/prometheus:15020`，通过该接口被 Prometheus 收集。

它同时也会采集来自 APISIX 9091 端口的 APISIX 指标。支持 gzip 和纯文本格式。

## 安装 Prometheus 与配置

在示例中，我们将直接使用 Istio 提供的快速示例配置来安装、运行 Prometheus。

```bash
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.16/samples/addons/prometheus.yaml
```

以默认配置安装完成后，访问 Prometheus 的 `9090` 端口即可访问 Prometheus UI 界面。

此外，可以为该配置文件添加以下 job 来抓取、监测各 sidecar 的状态。

```yaml
- job_name: 'envoy-stats'
  metrics_path: /stats/prometheus
  kubernetes_sd_configs:
  - role: pod

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_container_port_name]
    action: keep
    regex: 'prom'
```

该 job 匹配了 Amesh 所注入的 sidecar 配置中名为 `prom` 的端口（该端口即为 `15020`）。

这样，就能够在 Prometheus UI 看到各 sidecar 的状态了。

## 验证 Amesh 的指标

运行示例程序，访问 `/stats/prometheus:15020` 即可查看指标数据。

```bash
kubectl exec -it consumer -- curl http://127.0.0.1:15020/stats/prometheus
```

注意，部分指标，例如延迟数据等，在没有实际数据的情况下是不会显示的。需要至少一次请求，这些指标才会出现。

随后，便可以从 Prometheus UI 查询指标了。
