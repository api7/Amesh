#!/usr/bin/bash

kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io istiod-default-validator
kubectl delete mutatingwebhookconfigurations $(kubectl get mutatingwebhookconfigurations| grep amesh-e2e | awk "{print \$1}")
kubectl delete clusterrole $(kubectl get clusterrole| grep amesh-e2e | awk "{print \$1}")
kubectl delete clusterrolebinding $(kubectl get clusterrolebinding | grep amesh-e2e | awk "{print \$1}")
