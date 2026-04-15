#! /bin/bash

export KUBECONFIG=tmp/kubeconfig.yaml
kubectl delete gateway --all
kubectl delete httproute --all
kubectl delete gatewayclass --all