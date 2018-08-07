#!/bin/bash

export GOOS=linux; go build .
cp crd-kube-provenance ./artifacts/simple-image/kube-crdprovenance-apiserver
docker build -t kube-crdprovenance-apiserver:latest ./artifacts/simple-image


