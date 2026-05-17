#! /bin/bash

export KUBECONFIG=tmp/kubeconfig.yaml
kubectl delete gateway --all --now
kubectl delete httproute --all --now
kubectl delete gatewayclass --all --now
kubectl delete svc http-backend