@echo off

go run tooling/cli.go onprem --compose samples/busybox --name busybox --label app:hpcr --target config:onpremsample --image-url http://hpcr-qcow2-image.default:8080/hpcr.qcow2 --storage-pool images --cert build/mavenResolver/hpcr.crt | kubectl apply -f -
