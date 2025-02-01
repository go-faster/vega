#!/bin/bash

set -e

# Get all pods in all namespaces
pods=$(kubectl get pods --all-namespaces -o json)

# Extract and print the images
echo "$pods" | jq -r '.items[] | .spec.containers[]?.image' | grep -v vega | sort | uniq > _hack/images.txt
