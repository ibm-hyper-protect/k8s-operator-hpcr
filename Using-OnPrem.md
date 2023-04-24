
# Using k8s-operator-hpcr to manage IBM Hyper Protect Virtual Servers on a LPAR

Now that you have installed the Hyper Protect Virtual Servers Kubernetes Operator, create Kubernetes artifacts that will be used to create [IBM Hyper Protect Virtual Servers V2](https://www.ibm.com/docs/en/hpvs/2.1.x) through the libvirt API.  The operator defines a [custom resource definition](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to deploy a [Hyper Protect Virtual Server](https://www.ibm.com/docs/en/hpvs/2.1.x?topic=servers-setting-up-configuring-hyper-protect-virtual) to an LPAR. The controller does the following:

- make the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime) available on the LPAR
- manage the artifacts required to start the VSI (cloud init disk, boot disk, external disk, logging)
- start the VSI on the LPAR through the [libvirt virtualization API](https://www.libvirt.org/manpages/virsh.html)
- monitor the status of the VSI on the LPAR
- destroy the VSI on the LPAR and clean up the associated resources

## 1. Creating a Kubernetes ConfigMap for your SSH Configuration

The SSH configuration consists of the hostname of the libvirt host and the required SSH parameters (key, known hosts, etc) to connect to that host via SSH.

**Important:** make sure to give the config map sensible labels so it can be referenced by the custom resource defining the VSI. In the following example we use the label `app:onpremtest` for selection.

The same config map can be used for multiple VSIs running on the same LPAR.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-onprem-configmap
  labels:
    app: onpremtest
    version: 20.0.1
data:
  HOSTNAME: example.lpar.com
  KEY: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  PORT: "22"
  KNOWN_HOSTS: |
    ...
    ...
  USER: root
```

You may create such a config map based on your local [ssh config](https://www.ssh.com/academy/ssh/config) via the tooling CLI:

```bash
go run tooling/cli.go ssh-config --config onpremz15 --name onprem-sshconfig-configmap --label app:onpremtest --label version:0.0.1 | kubectl apply -f -
```

## 2. Make the HPCR image available in the k8s cluster

The [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime) must be accessible to the controller in form of a `qcow2` file and the controller relies on an HTTP(s) accessible location for that file.

 
The mechanism to provision the image is out of the scope of this controller design, there exist many ways to do this:

- this could be an external server, outside of the k8s cluster but accessible to the operator
- it could be a pod running in the cluster
- it could be a full fledged file server that not only allows to download the image but that would also allow to upload new images

One very simple way to provision the image is by compiling a docker image that packages the image, push to a known container registry and deploy that as a pod or other workload resource in the cluster, e.g.

- [Download and uncompress the HPCR image](https://www.ibm.com/docs/en/hpvs/2.1.x?topic=servers-downloading-hyper-protect-container-runtime-image)

- The docker file packages the qcow2 image (and for completeness the encryption key) and exposes it via HTTP.

  ```Dockerfile
  FROM busybox

  COPY ibm-hyper-protect-container-runtime-23.1.0.qcow2 /var/lib/hpcr/hpcr.qcow2
  COPY ibm-hyper-protect-container-runtime-23.1.0-encrypt.crt /var/lib/hpcr/hpcr.crt

  EXPOSE 80
  WORKDIR /var/lib/hpcr/

  ENTRYPOINT [ "httpd", "-fvv"]
  ```

- The container based on this file can be deployed as a pod in the cluster:

  ```yaml
  ---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: hpcr-qcow2-image
    labels:
      hpcr: qcow2
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: hpcr-qcow2-image
    template:
      metadata:
        labels:
          app: hpcr-qcow2-image
      spec:
        containers:
        - name: hpcr-qcow2-image
          image: <IMAGE REFERENCE COMES HERE>
          ports:
            - containerPort: 80
          resources:
            limits:
              memory: 512Mi
              cpu: "1"
            requests:
              memory: 256Mi
              cpu: "0.2"
  ---
  apiVersion: v1
  kind: Service
  metadata:
    name: hpcr-qcow2-image
  spec:
    selector:
      app: hpcr-qcow2-image
    ports:
    - port: 8080
      targetPort: 80
  ```

  **Note:** in case the image resides in a private container registry, make sure to add [imagePullSecrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/). If you prefer to use a local image please consult the documentation of your k8s cluster to see how to import an image to the cluster.

- for such a configuration the imageURL to be used would be (assuming deployment into the `default` namespace)

  ```bash
  http://hpcr-qcow2-image.default:8080/hpcr.qcow2
  ```

## 3. Deploying a Hyper Protect Container Runtime KVM guest

The `HyperProtectContainerRuntimeOnPrem` custom resource describes the properties of the HPCR KVM guest. Since the guest runs on a remote LPAR, this configuration needs to reference the config map with the related SSH login information. This reference is done via a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/).

### a. Deploying a VSI without a Data Disk

The following example shows how to deploy a VSI that does not need persistent storage.

Example:

```yaml
apiVersion: hpse.ibm.com/v1
kind: HyperProtectContainerRuntimeOnPrem
metadata:
  name: onpremsample
spec:
  contract: ...
  imageURL: ...
  storagePool: ...
  targetSelector: 
    matchLabels:
      ...
```

Where the fields carry the following semantic:

- `contract`: the [contract document](https://www.ibm.com/docs/en/hpvs/2.1.x?topic=servers-about-contract) (a string). Note that this operator does **not** deal with encrypting the contract. You might want to use [tooling](https://github.com/ibm-hyper-protect/linuxone-vsi-automation-samples/tree/master/terraform-hpvs/create-contract) to do so.
- `imageURL`: an HTTP(s) URL serving the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime). The URL should be resolvable from the Kubernetes cluster, have a filename part, and that filename will be used as an identifier of the HPCR image on the LPAR. 
- `storagePool`: during the deployment of the VSI the controller manages several volumes on the LPAR. This setting identifies the name of the storage pool on that LPAR that hosts these volumes. The storage pool has to exist and it has to be large enough to hold the volumes.
- `targetSelector`: a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the config map that holds the SSH configuration

### b. Deploying a VSI with a Data Disk

The following example shows how to deploy a VSI that does need persistent storage.

1. Define the data disk. Note that the disk is labeled as `app:hpcr`

    ```yaml
    ---
    kind: HyperProtectContainerRuntimeOnPremDataDisk
    apiVersion: hpse.ibm.com/v1
    metadata:
      name: sampledisk
      labels:
        app: hpcr
    spec:
      size: 107374182400
      storagePool: images
      targetSelector:
        matchLabels:
          config: onpremsample
    ```

2. Define the VSI and reference the data disk:

    ```yaml
    apiVersion: hpse.ibm.com/v1
    kind: HyperProtectContainerRuntimeOnPrem
    metadata:
      name: onpremsample
    spec:
      contract: ...
      imageURL: ...
      storagePool: ...
      targetSelector: 
        matchLabels:
          ...
      diskSelector: 
        matchLabels:
          app: hpcr
    ```

    Where the fields carry the following semantic:

    - `contract`: the [contract document](https://www.ibm.com/docs/en/hpvs/2.1.x?topic=servers-about-contract) (a string). Note that this operator does **not** deal with encrypting the contract. You might want to use [tooling](https://github.com/ibm-hyper-protect/linuxone-vsi-automation-samples/tree/master/terraform-hpvs/create-contract) to do so.
    - `imageURL`: an HTTP(s) URL serving the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime). The URL should be resolvable from the Kubernetes cluster, have a filename part, and that filename will be used as an identifier of the HPCR image on the LPAR. 
    - `storagePool`: during the deployment of the VSI the controller manages several volumes on the LPAR. This setting identifies the name of the storage pool on that LPAR that hosts these volumes. The storage pool has to exist and it has to be large enough to hold the volumes.
    - `targetSelector`: a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the config map that holds the SSH configuration
    - `diskSelector`: a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the data disk descriptor or a data disk reference descriptor (or a mix)

    **Note:** In this setup we attach the data disk to the VSI but in order to be useable to an OCI container it has to be [mounted via the contract](https://cloud.ibm.com/docs/vpc?topic=vpc-about-contract_se#hpcr_contract_volumes).

### c. Deploying a VSI with a Data Disk Reference

The following example shows how to deploy a VSI that does need persistent storage but the persistent storage has alread been pre-created and is not managed by the operator.

1. Define the data disk reference. Note that the disk reference is labeled as `app:hpcr`

    ```yaml
    ---
    kind: HyperProtectContainerRuntimeOnPremDataDiskRef
    apiVersion: hpse.ibm.com/v1
    metadata:
      name: samplediskref
      labels:
        app: hpcr
    spec:
      volumeName: existingName
      storagePool: images
      targetSelector:
        matchLabels:
          config: onpremsample
    ```

2. Define the VSI and reference the data disk:

    ```yaml
    apiVersion: hpse.ibm.com/v1
    kind: HyperProtectContainerRuntimeOnPrem
    metadata:
      name: onpremsample
    spec:
      contract: ...
      imageURL: ...
      storagePool: ...
      targetSelector: 
        matchLabels:
          ...
      diskSelector: 
        matchLabels:
          app: hpcr
    ```

    Please refer to [Deploying a VSI with a Data Disk](#b-deploying-a-vsi-with-a-data-disk) for a description of these parameters.


### d. Deploying a VSI with a Network Reference

The following example shows how to deploy a VSI that attaches to a predefined network.

1. Define the network reference. Note that the network is labeled as `app:hpcr`

    ```yaml
    ---
    kind: HyperProtectContainerRuntimeOnPremNetworkRef
    apiVersion: hpse.ibm.com/v1
    metadata:
      name: samplenetworkref
      labels:
        app: hpcr
    spec:
      networkName: default
      targetSelector:
        matchLabels:
          config: onpremsample
    ```

2. Define the VSI and reference the network:

    ```yaml
    apiVersion: hpse.ibm.com/v1
    kind: HyperProtectContainerRuntimeOnPrem
    metadata:
      name: onpremsample
    spec:
      contract: ...
      imageURL: ...
      storagePool: ...
      targetSelector: 
        matchLabels:
          ...
      networkSelector: 
        matchLabels:
          app: hpcr
    ```

    Please refer to [Deploying a VSI with a Data Disk](#b-deploying-a-vsi-with-a-data-disk) for a description of these parameters.

    - `networkSelector`: a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the network or network reference

## Footnotes

### Disks

The operator will create a number of disks on the host. All disks are created on the same [storage pool](https://libvirt.org/storage.html) and the [storage pool](https://libvirt.org/storage.html) must exist on the LPAR.

#### BootDisk

The boot disk contains the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime). This image will be identical across multiple instances, so it can be reused. However each start of a VSI will modify the image (because it creates a new LUKS layer).

The operator will therefore first ensure that the correct HPCR base image is uploaded, this can be a time consuming process. It then creates a copy of that image for each VSI, since the copy is created on the host itself, this is a very fast operation. The VSI will then run on top of the copied images, therefore keeping the base image untouched.

#### CIData Disk (Contract)

The CIData disk is an ISO disk containing the [contract](https://cloud.ibm.com/docs/vpc?topic=vpc-about-contract_se), i.e. the start parameters of the VSI. This is a small piece of data of `O(kB)`. It will be created and uploaded for each new VSI.

#### Logging Disk

The operator configures the VSI to log the console output to a file and it reserves storage space for that file in form of a logging volume. The log file will be used to track the startup progress (and potential errors) of the VSI.

#### DataDisk

Data disks represent persistent volumes. They are created via the custom resource `HyperProtectContainerRuntimeOnPremDataDisk` and linked to the VSI via labels.

The following example defines a data disk:

```yaml
---
kind: HyperProtectContainerRuntimeOnPremDataDisk
apiVersion: hpse.ibm.com/v1
metadata:
  name: sampledisk
  labels:
    app: hpcr
spec:
  size: 107374182400
  storagePool: images
  targetSelector:
    matchLabels:
      app: onpremtest
```

Notice how the selector `app: onpremtest` selects the SSH configuration.

The data disk may be stored on a different storage pool than the boot disk of the VSI.

## Debugging

### OnPrem VSIs

After deploying a custom resource of type `HyperProtectContainerRuntimeOnPrem` the controller will try to create the described VSI instance and will synchronise it state. The state of this process is captured in the `status` field of the `HyperProtectContainerRuntimeOnPrem` resource as shown:

```yaml
status:
  description: |
    LOADPARM=[        ]
    Using virtio-blk.
    Using SCSI scheme.
    ..........................................................................................................................
    # HPL11 build:23.3.16 enabler:23.3.0
    # Fri Mar 17 10:18:45 UTC 2023
    # create new root partition...
    # encrypt root partition...
    # create root filesystem...
    # write OS to root disk...
    # decrypt user-data...
    2 token decrypted, 0 encrypted token ignored
    # run attestation...
    # set hostname...
    # finish root disk setup...
    # Fri Mar 17 10:19:13 UTC 2023
    # HPL11 build:23.3.16 enabler:23.3.0
    # HPL11099I: bootloader end
    hpcr-dnslookup[860]: HPL14000I: Network connectivity check completed successfully.
    hpcr-logging[1123]: Configuring logging ...
    hpcr-logging[1124]: Version [1.1.93]
    hpcr-logging[1124]: Configuring logging, input [/var/hyperprotect/user-data.decrypted] ...
    hpcr-logging[1124]: Sending logging probe to [https://logs.eu-gb.logging.cloud.ibm.com/logs/ingest/?hostname=6d997109-6b44-40eb-8d88-8bf7fc90bfb5&now=1679048355] ...
    hpcr-logging[1124]: HPL01010I: Logging has been setup successfully.
    hpcr-logging[1123]: Logging has been configured
    hpcr-catch-success[1421]: VSI has started successfully.
    hpcr-catch-success[1421]: HPL10001I: Services succeeded -> systemd triggered hpl-catch-success service
  status: 1
```

With the following semantics:

- `status`: a status flag
- `description`: for a running VSI this carries the console log. For an errored instance it carries the error information

### Network References

After deploying a custom resource of type `HyperProtectContainerRuntimeOnPremNetworkRef` the controller will try to locate the referenced network and will synchronise it state. The state of this process is captured in the `status` field of the `HyperProtectContainerRuntimeOnPremNetworkRef` resource as shown:

```yaml
status:
  description: |2-
      <network>
          <name>default</name>
          <uuid>97d16d9e-da57-492c-82a0-0388561bf065</uuid>
          <forward mode="nat">
              <nat>
                  <port start="1024" end="65535"></port>
              </nat>
          </forward>
          <bridge name="virbr0" stp="on" delay="0"></bridge>
          <mac address="52:54:00:2b:4b:c4"></mac>
          <ip address="192.168.122.1" netmask="255.255.255.0">
              <dhcp>
                  <range start="192.168.122.2" end="192.168.122.254"></range>
              </dhcp>
          </ip>
      </network>
  metadata:
    Name: default
    networkXML: |2-
        <network>
            <name>default</name>
            <uuid>97d16d9e-da57-492c-82a0-0388561bf065</uuid>
            <forward mode="nat">
                <nat>
                    <port start="1024" end="65535"></port>
                </nat>
            </forward>
            <bridge name="virbr0" stp="on" delay="0"></bridge>
            <mac address="52:54:00:2b:4b:c4"></mac>
            <ip address="192.168.122.1" netmask="255.255.255.0">
                <dhcp>
                    <range start="192.168.122.2" end="192.168.122.254"></range>
                </dhcp>
            </ip>
        </network>
  observedGeneration: 1
  status: 1
```

With the following semantics:

- `status`: a status flag
- `description`: some textual description of the status of the network or the error message
- `networkXML`: an XML description of the network
