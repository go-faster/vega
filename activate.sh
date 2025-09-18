#!/bin/bash

# Use as "source activate.sh"

echo "Activating vega environment..."

export KUBECONFIG=_out/kubeconfig.yml
echo "KUBECONFIG set to $KUBECONFIG"

kubectl get pods

echo "Environment activated."
