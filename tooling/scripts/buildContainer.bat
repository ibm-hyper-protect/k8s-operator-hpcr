@echo off

setlocal

set CGO_ENABLED=0 
set GOOS=linux
set GOARCH=amd64
set OUT=tooling\scripts\build\%GOOS%\%GOARCH%

for /f %%i in ('busybox date -u +%%s') do set UNIX_TIME=%%i

mkdir %OUT% 2> NUL

copy /Y Dockerfile %OUT%\Dockerfile

go build -v -ldflags "-X github.com/ibm-hyper-protect/k8s-operator-hpcr/cli.version=0.0.0 -X github.com/ibm-hyper-protect/k8s-operator-hpcr/cli.compiled=%UNIX_TIME%" -o %OUT%\k8s-operator-hpcr
docker build %OUT% -t ghcr.io/ibm-hyper-protect/k8s-operator-hpcr:latest

