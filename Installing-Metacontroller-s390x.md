# Installing Metacontroller on an s390x k8s cluster

[Metacontroller](https://github.com/metacontroller/metacontroller) is not available for the s390x platform out of the box. This [PR](https://github.com/metacontroller/metacontroller/pull/771) adds support but in the meantime it is possible to deploy metacontroller on s390x using the following recipe:

## Build Metacontroller

Prequisites:

- an s390x virtual server (e.g. a LinuxONE box)
- installation of [go](https://go.dev/doc/install)
- installation of [docker](https://docs.docker.com/engine/install/ubuntu/)

Steps:

- clone the metacontroller repository in the desired version e.g.

    ```bash
    git clone https://github.com/metacontroller/metacontroller -b v4.7.10
    ```

- build the metacontroller binary

    ```bash
    cd metacontroller
    CGO_ENABLED=0 GOOS=linux GOARCH=s390x /usr/local/go/bin/go build ./pkg/cmd/metacontroller
    ```

- build the docker image

    ```bash
    docker build . -t metacontrollerio/metacontroller:v4.7.10
    ```

    Make sure to tag the image using the same version identifier as used for the checkout

## Deploy the Metacontroller image into the k8s cluster (alternative A)

TBD - find out how to do this

## Deploy the Metacontroller image into a custom registry (alternative B)

- tag the image with your registry URL

    ```bash
    docker tag metacontrollerio/metacontroller:v4.7.10 <YOUR_REGISTRY>
    ```

- push the image

    ```bash
    docker image push <YOUR_REGISTRY>
    ```

- edit the file `manifests/production/metacontroller.yaml` and change the line

    ```yaml
    image: metacontrollerio/metacontroller:v4.7.10
    ```

    to 

    ```yaml
    image: <YOUR_REGISTRY>
    ```
 

## Deploy the Metacontroller descriptors

From within the directory used to build the controller above, execute the following line:

```bash
kubectl apply -k manifests/production
```
