#!/usr/bin/bash

kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io istiod-default-validator
kubectl delete mutatingwebhookconfigurations $(kubectl get mutatingwebhookconfigurations| grep amesh | awk "{print \$1}")
kubectl delete clusterrole $(kubectl get clusterrole| grep amesh | awk "{print \$1}")
kubectl delete clusterrolebinding $(kubectl get clusterrolebinding | grep amesh | awk "{print \$1}")
