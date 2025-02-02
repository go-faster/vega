package vega

//go:generate go run -mod=mod github.com/ogen-go/ogen/cmd/ogen --target internal/oas --package oas --clean _oas/openapi.yaml
//go:generate go run ./cmd/generate-dashboards/ -o  _hack/k8s-create/grafana-dashboards.yml
