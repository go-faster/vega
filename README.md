# vega [![Go Reference](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/vega#section-documentation) [![codecov](https://img.shields.io/codecov/c/github/go-faster/vega?label=cover)](https://codecov.io/gh/go-faster/vega) [![experimental](https://img.shields.io/badge/-experimental-blueviolet)](https://go-faster.org/docs/projects/status#experimental)

Work in progress.

Research, development and best practices incubator for:
- Platform engineering
- Application development
- Observability
- Monitoring
- Configuration management
- Documentation
- Integration and performance testing
- Integrations
  - Tetragon
  - Cilium
  - Hubble

## Running

Requirements:
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [helm](https://helm.sh/docs/intro/install/)
- [cilium cli](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/#install-the-cilium-cli)
- docker
- go 1.23

```bash
helm repo add cilium https://helm.cilium.io
helm repo add clickhouse-operator https://docs.altinity.com/clickhouse-operator/
helm repo add faster https://go-faster.github.io/charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add vm https://victoriametrics.github.io/helm-charts/

helm repo update
```

```bash
go run ./cmd/vega-install
export KUBECONFIG=_out/kubeconfig.yml
kubectl get pods -n vega
```
