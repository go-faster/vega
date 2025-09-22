#!/bin/bash

set -e

echo "Generate docker registry secret for Harbor"
kubectl create secret docker-registry harbor-registry \
  --docker-server=localhost:30808 \
  --docker-username=admin \
  --docker-password=Harbor12345 \
  --dry-run=client -o yaml > _hack/k8s/harbor-registry-secret.yml
