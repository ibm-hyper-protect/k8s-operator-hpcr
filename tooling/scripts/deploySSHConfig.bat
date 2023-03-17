@echo off

rem go run tooling/cli.go ssh-config --config onpremz15 --name onprem-sshconfig-configmap --label config:onpremsample --label version:0.0.1 | kubectl apply -f -
go run tooling/cli.go ssh-config --config onpremz15 --name onprem-sshconfig-configmap --label config:onpremsample --label version:0.0.1