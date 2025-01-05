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
- docker
- go 1.23

```bash
go run ./cmd/vega-install
export KUBECONFIG=_out/kubeconfig.yml
kubectl get pods -n vega
```
