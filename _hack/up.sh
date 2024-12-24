#!/bin/bash

set -e -o pipefail

kind create cluster --name vega --config _hack/vega.kind.yml

kubectl apply -f _hack/monitoring.coreos.com_servicemonitors.yaml

helm upgrade --install cilium cilium/cilium --version 1.16.5 --values _hack/cilium.yml --namespace kube-system

cilium status --wait
