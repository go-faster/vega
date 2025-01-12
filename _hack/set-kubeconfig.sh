#!/bin/bash

# Use as:
# source _hack/set-kubeconfig.sh

export KUBECONFIG=_out/kubeconfig.yml
echo "KUBECONFIG set to $KUBECONFIG"
