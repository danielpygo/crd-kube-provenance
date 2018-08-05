#!/bin/bash

#kubectl delete ns wardle
#kubectl delete -f artifacts/example/auth-delegator.yaml -n kube-system
#kubectl delete -f artifacts/example/auth-reader.yaml -n kube-system
#kubectl delete -f artifacts/example/apiservice.yaml

kubectl.sh create -f artifacts/example/ns.yaml
#kubectl.sh create configmap -n provenance kind-compositions-config-map --from-file=kind_compositions.yaml


kubectl.sh create -f artifacts/example/sa.yaml -n provenance
kubectl.sh create -f artifacts/example/auth-delegator.yaml -n kube-system
kubectl.sh create -f artifacts/example/auth-reader.yaml -n kube-system
kubectl.sh create -f artifacts/example/grant-cluster-admin.yaml
kubectl.sh create -f artifacts/example/rc.yaml -n provenance
kubectl.sh create -f artifacts/example/service.yaml -n provenance
kubectl.sh create -f artifacts/example/apiservice.yaml
