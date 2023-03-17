# HPCR Controller

The [hpcr-controller](https://github.com/ibm-hyper-protect/hpcr-controller) implements a custom k8s resource that starts ah HPCR VSI based on a custom resource definition. 

## Limitations

- only support the Default k8s namespace for the moment
- poor error handling in case the VSI startup fails (e.g. because of a wrong encryption key, fixed for onprem)
- all disks for the onprem case are created on the same storage pool

## Features OnPrem

The controller defines a [custom resource definition](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to deploy a [HPCR VSI](https://cloud.ibm.com/docs/vpc?topic=vpc-about-se) to an LPAR. The controller does the following:

- make the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime) available on the LPAR
- manage the artifacts require to start the VSI (cloud init disk, boot disk, external disk, logging)
- start the VSI on the LPAR
- monitor the status of the VSI on the LPAR
- destroy the VSI on the LPAR and clean up the associated resources

### Disks

The operator will create a number of disks on the host. All disks are created on the same [storage pool](https://libvirt.org/storage.html) and the [storage pool](https://libvirt.org/storage.html) must exist on the LPAR.

#### BootDisk

The boot disk contains the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime). This image will be identical across multiple instances, so it can be reused. However each start of a VSI will modify the image (because it creates a new LUKS layer).

The operator will therefore first ensure that the correct HPCR base image is uploaded, this can be a time consuming process. It then creates a copy of that image for each VSI, since the copy is created on the host itself, this is a very fast operation. The VSI will then run on top of the copied images, therefore keeping the base image untouched.

#### CIData Disk (Contract)

The CIData disk is an ISO disk containing the [contract](https://cloud.ibm.com/docs/vpc?topic=vpc-about-contract_se), i.e. the start parameters of the VSI. This is a small piece of data of `O(kB)`. It will be created and uploaded for each new VSI.

#### Logging Disk

The operator configures the VSI to log the console output to a file and it reserves storage space for that file in form of a logging volume. The log file will be used to track the startup progress (and potential errors) of the VSI.

### Configuration

The operator uses the following pieces of configuration.

#### SSH Config

The SSH configuration consists of the hostname of the libvirt host and the required SSH parameters (key, known hosts, etc) to connect to that host via SSH.

**Important:** make sure to give the config map sensible labels so it can be selected by the custom resource defining the VSI. In the following example we use the label `app:onpremtest` for selection.

The same config map can be used for multiple VSIs running on the same LPAR.

Representation as a config map:

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
  PORT: 22
  KNOWN_HOSTS: |
    ...
    ...
  USER: root
```

You may create such a config map based on your local [ssh config](https://www.ssh.com/academy/ssh/config) via the tooling CLI:

```bash
go run tooling/cli.go ssh-config --config onpremz15 --name onprem-sshconfig-configmap --label app:onpremtest --label version:0.0.1 | kubectl apply -f -
```

#### VSI Config

The VSI configuration describes the properties of the VSI. Since the VSI runs on a remote LPAR, this configuration needs to reference a config map with the related information. This reference is done via a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/).

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

- `contract`: the [contract document](https://cloud.ibm.com/docs/vpc?topic=vpc-about-contract_se) (a string). Note that this operator does **not** deal with encrypting the contract. You might want to use [documentation](https://cloud.ibm.com/docs/vpc?topic=vpc-about-contract_se) or [tooling](https://github.com/ibm-hyper-protect/linuxone-vsi-automation-samples/tree/master/terraform-hpvs/create-contract) to do so.
- `imageURL`: an HTTP(s) URL serving the [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime). The URL should have a filename part and that filename will be used as an identifier of the HPCR image on the LPAR. See a discussion below how to provision such a URL
- `storagePool`: during the deployment of the VSI the controller manages serveral volumes on the LPAR. This settings identifies the name of the storage pool on that LPAR that host these volumes. The storage pool has to exist and it has to be large enough to hold the volumes.
- `targetSelector`: a [label selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the config map that holds the SSH configuration

#### IBM Hyper Protect Container Runtime Image

The [IBM Hyper Protect Container Runtime image](https://cloud.ibm.com/docs/vpc?topic=vpc-vsabout-images#hyper-protect-runtime) must be accessible to the controller in form of a `qcow2` file and the controller relies on an HTTP(s) accessible location for that file.

The mechanism to provision the image is out of the scope of this controller design, there exist many ways to do this:

- this could be an external server, outside of the k8s cluster but accessible to the operator
- it could be a pod running in the cluster
- it could be a full fledged file server that not only allows to download the image but that would also allow to upload new images

One very simple way to provision the image is by compiling a docker image that packages the image and deploy that as a pod in the cluster, i.e. like so:

- The docker file packages the qcow2 image (and for completeness the encryption key) and exposes it via HTTP.

  ```Dockerfile
  FROM busybox

  COPY src/base-image.qcow2 /var/lib/hpcr/hpcr.qcow2
  COPY src/base-image.crt /var/lib/hpcr/hpcr.crt

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

- for such a configuration the URL to the image would be (assuming deployment into the `default` namespace)

  ```bash
  http://hpcr-qcow2-image.default:8080/hpcr.qcow2
  ```

### Debugging

After deploying a custom resource of type `HyperProtectContainerRuntimeOnPrem` the controller will try to create the described VSI instance and will monitor it. The status of this process is captured in the `status` field of the resource. This field looks like so:

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


## Features VPC

### ID

Each custom resource definition will get a UUID assigned by k8s. The controller uses this UUID to construct the name of the HPCR VSI, i.e. the name of the VSI is not user-friendly.

### Lifecycle

The custom controller is configured to re-validate the state of the VSI every 60s. If the VSI is not in running state (e.g. because it has been deleted manually on VPC) it will be re-created. 

### Contract

The HPCR contract is passed as a pre-encrypted string to the custom resource definition. In the future we might consider to support on-the-fly encryption of the contract. Note that it's important to use the correct encryption key.

```yaml
---
apiVersion: hpse.ibm.com/v1
kind: HyperProtectContainerRuntime
metadata:
  name: carsten-metadata
spec:
  contract: |
    env: hyper-protect-basic.noOa+GF8ILM1MXUx1964WlWudwjsSLilR5QWIeW7BK0djZLD/Zfip8MqDU3hbv1IHrbulwBc4w8XC55r3ySEbiSmQqEAXG9HpdMaKu0v8nbRmxSHX6DnfanH1Wt5wwtNcgRPnDtt/L0gfxTxKXDlDLYJDplK142ehuwSWnG/rTFNyTrbIHpGFMhLEp5lrZKt2lk9tIyWkjoWdDzHMrIvw2z0skk7MVJfuJsa/WzcZNTz82ukdoYXIQxmfgY9gWNdqXAsII5yDLM0zOjxXqzXjPCh0+l2z9pUyVT68ZHsczZaxmZrY6pIF2KpKwuo0rRj4fU4Di72oIF64FYFmF2YpJ6wpyx/RfppgzmybajnrPHD89OaG/UUf84sYczQMdquOprLS7CsnMcmfv3GEahOSSq8i6GGWTDMKbFWsvE73EcecvKqdq2P0C3b6il+zfXWc1leHeXuXncKp21OjqmkCXr5f3M/H5k1w7QPPKx4JdxmXV9tPKb5B9ZHStUcAJiRapmuOIp9TaAIVDZ86CYibnB0ox1j7dyWR2hUxWaAvz4KBNul3q2wyruafn5Vm96QtluWSbBKggUrkSz9n224yUi/DuKCeLbWpErMj3lCut16ql8KbCZcSvG6IzUMBimp91Y8XELqDsIDOe3iITaGBBLtHR7PXgsgaGmCe0GV/vY=.U2FsdGVkX1/TGaz32eixpQIlbx3Npwn2Ir33H5VC97YQ3FT7FMZyTe4FrzUfInLEPx64nwBNBmAN1MG+sQ7dE0RZ0339B00cQCis37yPL2RieZV6oHUUEPr4aP82d+0MyE9yV0FiutODJC7NAnCJLgwh0kYBeee3yqOgf/VnZt5joFY7fmf+/pQyBAB4LHvrIfIdjFr32q9v9E7dfxzTDrtNSJmveiet6Kxm6+wL9Y0sXM1BVZZ/jrCsPEQDDNBMXD7Oh3k6od+3z/usZmYDe6I3YFuhN4E3NsHkcPe3mlfpie7AbkykyW5s+qUn8+Jp55H5VNzrJZeALE/I6b3JxPXCTT/xJDoD1WKRle2li7lSwYzvj4/lHA3pvU0k1Gw7ptklfKRtvGLZW08+Ot7LSRPARAC4VgQqZY2SZ3lQbEddXA4FmaP04bYerA88M8PQzbYzIfbxRKCDLlzDp2hyQne3GsYxOkHG/X+XjjDaVF2Xn4lG9d5bo1c919+7HpnnDIr307x7Eo1kp06iO3WX+/vk/5JIgvqrjDdHsgAFpjW2T3cbvbI63/h5FbNChb2Hxa5FUVsZmFbl3c1rqjBdkAz9Yna42Le/HKj8KuHMTUxalhRXdu5w+VSmvR80gj9VW6/sI2qCPKAwhklSbJXjx7sIluY5j70FBxA6UigXPk5ZMIN+YxvGwpJ61VeCH+uhqGVGTJeetZa6j9yh5MYA4ccrKn3ZcrCxgeQxkRnQ+zas6Hac0t0vBj4sbjI85ASxgx4lBr+iMbLwzYO1RWLKKN5QSZvyw2JIH5UjV7Z5fnzUqUPG8djyITRZeBxo/2J5h4yo4eh1lcEYiD2Ny20piBYMaT0mi2CmXklFiecpVn9BVo/k5WgPLNNIjU7RhHl3xcbZ/XTO+WLYmC9RN49aU5o4Sn4HHC13Yn8M0klg4JsZ9/lB4Uz1X+k/mPHtNvBJQn6gyMF7wfL7P20vC/+fSdusEnaNEkWOsP8WioN2BT+cMiT1KC7mnsiO9c3YHQvgvgZUuFpRmJ9Uu4oyU7pJsoTpKdAeNAxhWz/Jjf6N1q96hr94Oz/87yXOrdk1y7GhJH53Mbu7PWjpWK7tj2UA2BHjiqjNzkOsXtKsnIjRxQoT92WxEkULAcWQi5srFciUwYouK3rABC8gCsqwHNmdDjLc7iM8k5bdQrI6Gj67YWDuKG3zkf2zkv+4971EnTO4IDcu7pGveFX7o9vy/m2pizsAxQ+623gYOXGzng89ZPgBkxyYqq6O2clK9rHpiUMC70+lrKz1pgCbpiLQUx3acjaKL3sMrYRILH5X4kis+Vg=
    workload: hyper-protect-basic.dSt1QcaVHJGcptkuR2nFM9t11UU8zVsaPyxsxzAIo8NHFmKShEwr98pvP5jdy3jfLn2fDzI6ILJgz+QlxTz3iIvnaBUe/ohwTSJGMHKBQrWl5ktlvXaiV9ol9XTAAKcYsWTdiSXqK9JRwS91icyGe2MYHJnV+RYwMz7B4RE0Rsz891Vaw241Z/cqYCDkPI0Lb0l6FG1rNbae9E5D/wfSJmO1OKWimk+YdkLwJj4+RwqTdvZtIUEArV6DJMLT7N0W+ExFo4hC274Z/Br2vGu7FvQXLJotqqV7Q+eXtWWapwJsQ6Fw5belR2+uRCid/4MdBCT0M6q7ojZX5pgmTnbZYdqApbtFi0ticQfdV4OUVLWQBI2ja+vkiLRJFJImeM8EMvFzLbzWYnFWxK9oh4RUbkQypehvSEKbm8c9At7UnO5aDr2m88ZZYdQZXzygZSQc90gQP332u8PRkHGXIOz9MWjBOvuO1kVWIxikG9J70ulodkiMU3WRQciLYiQWigJJiHBdcRmMx67+xLmOVldpE1f3s6C/50dwVcVC5nucrSa4pK6LgIrdGRnpmIKyl3J5NhEnzlKztlkKTC+RDaqm6Vwr1+CCOqqBrlYI03PGoU+JB/MKQgGAP26r4lVrvReXOQRmvKqODc6NOVozsttKmU5l/lZQUo1M6AIzcXL3/Og=.U2FsdGVkX1/HLzVYcvJPIg0vnzHhsMpeA4yGrzdnwwUFM7rDNwIpeMBz//wHappogx89uj9Ggtrn8PXSHM/eU5PwuFoVOyY7sglhAOCQk7aVdsqwvPFpQI6iFp0rWgyNteXiXBi+GnxWv9TZZueNCOMS+4vcSQMJLKdBasrJUp+Lh07O8/utouS74d1rLtFRY6CBZ9c9lyBJplSsuz+C33+yXsbynHbO6adMtDry9IQeoqrL3eNemobU+AISGunOY7GVM/IJ/MK4ssq6DrmtJ2+zCdPb4u16YzvhzuM6PVzgqmDMxorvuoVs8Irm4/lG7EK39dNwriwDvBhkpuFdWi0tESBKFD/Hg42TTCLSZVDAH2LoBUXCkHvfqMmVmRNXCdU+OmRHqcuPjxfjpAOSmCw/XS/D8YiCtLxMNQLDEZFsbaPUbIABiJHWYyqdvSZUHRiQYc9Regjm+pUmiYbc7OoA0n0G8yIyUEOidvhNILo14G195loVy7Hldtp7hg2FywXXaBMtrz4sLjy95Xo24yQ1q8fIC+pbTErKpaNX/5MSN/LTPGHw3JE/IzUbHhihHjY8V9q77xyPauyakRZX4pCo7kWdqFVAsAFwZVUTT7WESnFUG9GpCoRO8rshtFqMj5Wz9ZyI/yuhtRtFBcuxPggKU+m2K4jLP5KI/Q4pbb6YN7lBGg7Ls4fBJWjer1DtF3EhGAe5yaZedQrKdfQdl3wdeZ6exjS4VMQkW9DJqDNq+V7Mksv3U3znTNfkYfGsgIvhyWliCHPgeUXSBQO3Y9uvQ40m7r9NkZnzTGSlMrDANuIpHEuC+2/4L9wegwO6QnIciLzGzYovbVnmYX4UGJ+zAf1buag/uvFIPQEXV1bcMIJDyf8EXD3hPrSQCgstBO4LqEUbvWE8UAK1FLD3eszLMAmVF4INes67kACDSSVu9twWe7FuJ/CzTyWkn4eAkVVb1w6xlWxBJ4IwvsJlp4AMDaGcODIH2NYCPKAd/bxiPqBofYYdg+VgCA0pKXuT1Pftiz/sqy07ugVeg6x+s1GSnb4aY3MoKbHd9U/1ryH2sBm0bdnfPK6H+VLXG34M/O2NQyrhosyBPqmKwRL5jlIAv5CpfGSlaWinXNjsi+Y1WIDj4uq6ZXeqSInAysBRmedaHT8csSlupC0zh22kgKbjIBdmAGS84GEFvKeith5Sf4vvWCXU1apybUf5HelIKKz68UDSfVNbO3mskiNFl7f7VwpJc49bkeDZkLEdCuqh5LJk1PFB7pPghzGtJGwbHdBnx8FJTE7plUMMgxuflHd4hyFy7Qtz9bFbTyeryByNIh4vYqoMOzz9a9n152W64O91KpDV6HyF5Lwm+DP3MF6lsT5UrLEicq45J1OTpodAMCxMcKXMGbK6rbMD/K4OHHSSB/QxW4BYDz/Yw0tSwrfLTsCFErspIbVjv+OGL2aIGtW5caVBPq3uRPHfr6Lror7OKF6vnYJeVBtBowI38waZrkPk/GP8/AvI49q5CaGfEBr2BfVY9YLHL6oVa6HE5Lq+kt6o/MvqzqO5bSs8t/J19gF74nJGbdCrq/LvpcwZoej49oae8eeOR/sFV9D0XTe4BUxnJ/myaLhmgMDXirrwCrTOkheHShcX1OduLItpn+vBh5Vn0eA+cabuxEOhUD3sIu0s2tra4ATf64wehqSGor9KVaksW9SPPpn0ml+jE8bFxCDS/YcTq6Jr4YTXsYMErknoax+GOPuHfvz8dddH4QDfE91BsMQDZJ4uPxOKZoFDRdgpI0cs8AORSOZb
    envWorkloadSignature: isCnt7375Vc/4EMw4HKjCfTLbwGT8VWvBcJ2QhNJIzepd3kSJDJm+TeO9aKfau0qfRkm5EVSAq1YaQw6jnevmbL2YJQnAaxIszPwbQfvrhSIQvywIBLZH5KerwSP7Ii+TU2VbH7bKyDTJv5o7QfeKlcxIyOEslcc4MVdyUd8bmh0ORqobE3YBjTDBmn7iWDG4Im3zxLrkNqMvMapPaYIBVdJVpJ0GTSbiFPO982n4NdYmp9jA+vKooj0uBwhfgv6t12DJzYResKu6wWuok3DfJP/5oaIK6wibuwwZ0IG4LPFNBeW3a/IKrVrSomKg0OOa2gAY/ol31GmIYzXVIniRmnMjerkeCAMWcEMVpIPTUxat/H8Ru/U/egIeXiDC5joJr64nLFXWcUHof3m39jBEWUTh7Muye5HP74+BmUNBDOXBpri14K2nGBON4gefxG7V1xqoNVD7y2h/lXg1cy0CDhjjBrgfS5PtnAORr4mYJumkfFr1l2c8EQr7rzTaTcKHvczz326HavffJty67/zLFDKinIgSQFV1KR65flPNBNg6jJhDRDktiMFsYpbgJu5Mwy7cz8HAvkGKoEA9v5wLaz30LyYaAqRrIFlc+eE829dyRTuEafRNVxLnaetme0EnI3zHq9MVPVoPokKHQwk0HV11u3sbrIR3pyvw/BLnSY=
  selector:
    matchLabels:
      hpcr: test
```

### Image Identifier

The **name** of the base image for the HPCR VSI can be provided via the `TARGET_IMAGE_NAME` field in one of the config maps. If this is not set, the controller tries to locate the latest stock image.

### Profile Identifier

The **name** of the profile can be provided via the `TARGET_PROFILE` field in one of the config maps. If this is not set, the controller defaults to `bz2e-2x8`.

### VPC and Subnet

The HPCR VSI will be attached to one subnet. Since a subnet is always attached to a VPC and a particular zone, the controller only requires to specify the ID of the subnet, the VPC and the zone will be derived from the subnet.

Use the `TARGET_SUBNET_ID` field in one of the config maps to specify the subnet ID.

## Usage

### Configuration

ConfigMap `vpc-env-configmap` to define the VPC endpoints. Can be omitted for a production environment.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-env-configmap
  labels:
    hpcr: test
data:
  IBMCLOUD_ACCOUNT_MANAGEMENT_API_ENDPOINT: https://accountmanagement.stage1.ng.bluemix.net
  IBMCLOUD_CERTIFICATE_MANAGER_API_ENDPOINT: https://us-south.certificate-manager.test.cloud.ibm.com
  IBMCLOUD_MCCP_API_ENDPOINT: https://mccp.us-south.cf.test.cloud.ibm.com
  IBMCLOUD_NAMESPACE_API_ENDPOINT: https://us-south.functions.cloud.ibm.com/api/v1
  IBMCLOUD_CS_API_ENDPOINT: https://containers.test.cloud.ibm.com/global
  IBMCLOUD_CIS_API_ENDPOINT: https://api.cis.test.cloud.ibm.com
  IBMCLOUD_DL_API_ENDPOINT: https://directlink.test.cloud.ibm.com/v1
  IBMCLOUD_GT_API_ENDPOINT: https://tags.global-search-tagging.test.cloud.ibm.com
  IBMCLOUD_HPCS_API_ENDPOINT: https://zcryptobroker.stage1.mybluemix.net/crypto_v2/
  IBMCLOUD_IAM_API_ENDPOINT: https://iam.test.cloud.ibm.com
  IBMCLOUD_IAMPAP_API_ENDPOINT: https://iam.test.cloud.ibm.com
  IBMCLOUD_ICD_API_ENDPOINT: https://api.us-south.databases.test.cloud.ibm.com
  IBMCLOUD_KP_API_ENDPOINT: https://qa.us-south.kms.test.cloud.ibm.com
  IBMCLOUD_PRIVATE_DNS_API_ENDPOINT: https://api.dns-svcs.cloud.ibm.com/v1
  IBMCLOUD_RESOURCE_MANAGEMENT_API_ENDPOINT: https://resource-controller.test.cloud.ibm.com
  IBMCLOUD_RESOURCE_CONTROLLER_API_ENDPOINT: https://resource-controller.test.cloud.ibm.com
  IBMCLOUD_RESOURCE_CATALOG_API_ENDPOINT: https://globalcatalog.test.cloud.ibm.com
  IBMCLOUD_SCHEMATICS_API_ENDPOINT: https://schematics.test.cloud.ibm.com
  IBMCLOUD_TG_API_ENDPOINT: https://transit.test.cloud.ibm.com/v1
  IBMCLOUD_UAA_ENDPOINT: https://login.stage1.ng.bluemix.net/UAALoginServerWAR
  IBMCLOUD_USER_MANAGEMENT_ENDPOINT: https://user-management.test.cloud.ibm.com
  IBMCLOUD_IS_API_ENDPOINT: https://us-south-stage01.iaasdev.cloud.ibm.com
  IBMCLOUD_IS_NG_API_ENDPOINT: https://us-south-stage01.iaasdev.cloud.ibm.com/v1
  IBMCLOUD_COS_ENDPOINT: https://s3.us-west.cloud-object-storage.test.appdomain.cloud
  IBMCLOUD_COS_CONFIG_ENDPOINT: https://config.cloud-object-storage.test.cloud.ibm.com/v1
```

ConfigMap `vpc-deployment-configmap` to define the VPC to deploy into:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-deployment-configmap
  labels:
    hpcr: test
data:
  IBMCLOUD_REGION: us-south
  IBMCLOUD_ZONE: us-south-2
  TARGET_IMAGE_NAME: hpse-pipeline-dev-gen2-enclaved
  TARGET_SUBNET_ID: 0726-b3c4aa3a-928a-4c8f-97b7-02ad4723c4e4
  TARGET_CONTRACT_PUB_KEY_FILENAME: |
    -----BEGIN PUBLIC KEY-----
    MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAoCnfoQTXR+OJxEL1tYZs
    +nm/y6f2Pr0Asgb4/8kmLgRchnbCUxsQQNhhgvk8iJZYKu+6CS0dKYId0X1Twm4W
    H5NfOor4UYXHmTYHvqDmvCNKWByZk2xBAWEgPm76YtlQcsTJ01S0lNBVfIqs5gWN
    o4Upv70uSPORyfINjZdMQ6a6mfI5Ittvbmx9c+VNKAXop3vVfUOlY1gFtKw9Kn/v
    uKJZ/JJXzpLx72Gq1B5k1brfCINbhXDNB9KsU/zkry1Gk1sGwLTY0xb/BzYIyis3
    +cPki+AyDmDOeBuVxYXC3j/ndWvlYiAIRVxn0zoJZxcsG9KqOoRaRRcNDYWEaNN9
    mi7mOeBczkAveSr9Jxtun6tZ4PRK/eD1HFBAcu7PtK39OcLdsayrD8Cn+tDdIFqj
    +lgHq4Z/rj11lRb9uk2aor0LnbbUhCeYQibrGN7hBz7wXm04MIpkUC1mNDhg2IuY
    uaBImbT8vt8uqsvLeLWcQg87B+gMMcOiyRu9aKFuAXYV3xsu5OckvAL+S7x43Bis
    TE2GSELABIxSgns5KniGu+V1EIN1AUJZMdECfgECuKmqCHvivy6IPa8I+y4QnXrj
    SA5Ecni8I/tCAvmMCHAzFysbjLvPoAUSYiOlW969Kc5iJuBvq40WOR8xba3NwJji
    CnzbhHxtOVAtLl5g92nLhv0CAwEAAQ==
    -----END PUBLIC KEY-----
  TARGET_PROFILE: bz2e-2x8  
  TARGET_VPC_NAME: carsten-test-3
```

ConfigMap `vpc-apikey-configmap` to define the IC API key:

```yaml
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-apikey-configmap
  labels:
    hpcr: test
data:
  IBMCLOUD_API_KEY: xxx
```

## OnPrem

### Make the HPCR image available in the k8s cluster

The HPCR image (qcow2) needs to be accessible in an HTTP addressable location from the operator. This can be achieved in many ways (including file servers that allow to upload the file) but a simple way is this:

- download the HPCR image, e.g. to `hpcr.qcow2`
- create a docker image like so

  ```Dockerfile
  FROM busybox

  COPY hpcr.qcow2 /var/lib/hpcr/hpcr.qcow2

  EXPOSE 80
  WORKDIR /var/lib/hpcr/

  ENTRYPOINT [ "httpd", "-f"]
  ```

- build the image

  ```bash
  docker build . -t hpcr-qcow2
  ```

Then deploy a pod with that image. 

For testing use

```bash
docker run --rm -it -p <PORT>:80 docker-eu-public.artifactory.swg-devops.com/sys-zaas-team-hpse-dev-docker-local/zaas/hpse-docker-22-04-dev-vm-x84_64
```

### Prerequisite

- Install [Metacontroller](https://metacontroller.github.io/metacontroller/guide/install.html):
  
  ```bash
  kubectl apply -k https://github.com/metacontroller/metacontroller/manifests/production
  ```

- Setup [Pull Secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials) when using private registries for the [hpcr-controller](https://github.com/ibm-hyper-protect/hpcr-controller) image. 
  
  The name of the [Pull Secret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials) needs to match the value of the `imagePullSecrets` field in the [webhook.yaml](k8s/webhook.yaml) file. The value of the secrets needs to allow for pull access to the image specified in the `image` field of the [webhook.yaml](k8s/webhook.yaml) file. e.g.
  ```bash
  kubectl create secret docker-registry artifactory \
  --docker-server=docker-eu-public.artifactory.swg-devops.com \
  --docker-username=<USERNAME> \
  --docker-password=<REGISTRY_APIKEY> \
  --docker-email=<USER_EMAIL>
  ```

### Deploy Resources

```bash
kubectl apply -f k8s
``` 

### Show Logs

```bash
kubectl logs -l app=hpcr-controller
``` 
 