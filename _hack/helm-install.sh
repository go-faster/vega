#!/bin/bash

set -e

helm repo add cilium https://helm.cilium.io
helm repo add clickhouse-operator https://docs.altinity.com/clickhouse-operator/
helm repo add faster https://go-faster.github.io/charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add vm https://victoriametrics.github.io/helm-charts/
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm repo add minio-operator https://operator.min.io
helm repo add aqua https://aquasecurity.github.io/helm-charts/
helm repo add harbor https://helm.goharbor.io
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts

helm repo update
