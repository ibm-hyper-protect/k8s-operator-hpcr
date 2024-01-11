# Hyper Protect Virtual Servers Kubernetes Operator

[![Controller Package](https://github.com/ibm-hyper-protect/k8s-operator-hpcr/actions/workflows/release-package.yml/badge.svg)](https://github.com/ibm-hyper-protect/k8s-operator-hpcr/actions/workflows/release-package.yml)

By using the [k8s-operator-hpcr](https://github.com/ibm-hyper-protect/k8s-operator-hpcr) you can implement a solution that manages Hyper Protect Virtual Servers based on a custom resource definition from within your Kubernetes cluster. By using Hyper Protect Virtual Servers, you can run industry-standard, Open Container Initiative (OCI) images within a secure enclave which provides a highly isolated, highly trusted environment for your workloads.

Hyper Protect Virtual Servers are provided by [IBM Cloud Hyper Protect Virtual Servers for VPC (HPVS for VPC)](https://cloud.ibm.com/docs/vpc?topic=vpc-about-se#about-hyper-protect-virtual-servers-for-vpc) in IBM Cloud data centers, or by [IBM Hyper Protect Virtual Servers v2](https://www.ibm.com/products/hyper-protect-virtual-servers) on a IBM Z(R) or LinuxONE system in your own hybrid multicloud environments.

To get started, see [how to setup the controller in your cluster](#installing-the-controller).  Once installed:

- To use the operator to manage IBM Cloud Hyper Protect Virtual Servers for VPC, see [Using-OnCloud.md](Using-OnCloud.md).
- To use the operator to manage IBM Hyper Protect Virtual Servers, see [Using-OnPrem.md](Using-OnPrem.md).

## Limitations

- only support the Default k8s namespace for the moment
- poor error handling in case the VSI startup fails (e.g. because of a wrong encryption key, fixed for onprem)
- all disks for the onprem case are created on the same storage pool
- IBM Hyper Protect Virtual Servers v1 and IBM Cloud® Hyper Protect Virtual Servers v1 are not supported.

## Installing the Controller

You need a Kubernetes cluster with Internet connectivity.

### 1. Install [Metacontroller](https://metacontroller.github.io/metacontroller/guide/install.html):
  
  ```bash
  kubectl apply -k https://github.com/metacontroller/metacontroller/manifests/production
  ```

### 2. Install the Hyper Protect Virtual Servers Kubernetes Operator

```bash
kubectl apply -k https://github.com/ibm-hyper-protect/k8s-operator-hpcr/manifests
``` 

### 3. Verify your installation by checking for the existence of the custom resources

```bash
kubectl get crds

NAME                                         CREATED AT
compositecontrollers.metacontroller.k8s.io   2023-03-15T21:32:11Z
controllerrevisions.metacontroller.k8s.io    2023-03-15T21:32:11Z
decoratorcontrollers.metacontroller.k8s.io   2023-03-15T21:32:11Z
onprem-hpcrs.hpse.ibm.com                    2023-03-17T12:44:30Z
vpc-hpcrs.hpse.ibm.com                       2023-03-17T12:44:30Z
```

```bash
kubectl get compositecontrollers

NAME                       AGE
k8s-operator-hpcr-onprem   5m37s
k8s-operator-hpcr-vpc      5m37s
```

```bash
kubectl get deployments

NAME                READY   UP-TO-DATE   AVAILABLE   AGE
k8s-operator-hpcr   1/1     1            1           6m35s
```

### Show Logs

```bash
kubectl logs -l app=k8s-operator-hpcr
```
### NOTE :

You should own the security related responsibilities of the Virtual Servers following security best practices that help in maintaining a more secure environment. If your environment is IBM Hyper Protect Virtual Servers then, please follow https://www.ibm.com/docs/en/hpvs/2.1.x?topic=servers-additional-security-responsibilities-hyper-protect-virtual for the additional security responsibilities.

