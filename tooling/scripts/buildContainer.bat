@echo off

setlocal

set CGO_ENABLED=0 
set GOOS=linux
set GOARCH=amd64

for /f %%i in ('busybox date -u +%%s') do set UNIX_TIME=%%i

mkdir tooling/scripts/build 2> NUL

go build -v -ldflags "-X github.com/ibm-hyper-protect/k8s-operator-hpcr/cli.version=0.0.0 -X github.com/ibm-hyper-protect/k8s-operator-hpcr/cli.compiled=%UNIX_TIME%" -o tooling/scripts/build/hpcr-controller
docker build tooling/scripts -t ghcr.io/ibm-hyper-protect/k8s-operator-hpcr:latest

