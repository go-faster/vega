#!/bin/bash

set -e -o pipefail

kind create cluster --name vega --config _hack/vega.kind.yml

cilium install --version 1.16.5
cilium status --wait
