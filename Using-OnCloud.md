# Using k8s-operator-hpcr for IBM Cloud

Now that you have installed the Hyper Protect Virtual Servers Kubernetes Operator, create Kubernetes artifacts that will be used to create [IBM Cloud Hyper Protect Virtual Servers for VPC](https://cloud.ibm.com/docs/vpc?topic=vpc-about-se) through the IBM Cloud API.  The operator defines a [custom resource definition](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to deploy a Virtual Server Instance within your VPC. The controller does the following:

- create a VSI on IBM Cloud through the [Virtual Private Cloud API](https://cloud.ibm.com/apidocs/vpc/latest) when the custom resource is created in the Kubernetes cluster.
- monitor the status of the VSI
- destroy the VSI when the custom resource is deleted in the Kubernetes cluster.

## Prerequisites

- A Kubernetes cluster with Internet connectivity.
- An IBM Cloud account that has been [upgraded to a paid account](https://cloud.ibm.com/docs/account?topic=account-accountfaqs#changeacct).
- An API key for your account. If you don't have an API key, see [Creating an API key](https://cloud.ibm.com/docs/account?topic=account-userapikey#create_user_key).
- You will need to have a Virtual Private Cloud with at least one subnet. If you don't have a VPC, see [Getting started with Virtual Private Cloud (VPC)](https://cloud.ibm.com/docs/vpc?topic=vpc-getting-started).
- You will need a Hyper Protect Virtual Servers contract. To create the contract by script, see this example: https://github.com/ibm-hyper-protect/linuxone-vsi-automation-samples/tree/master/terraform-hpvs/create-contract.  

## Planning

Before you begin, you need to decide upon a [label](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for the Virtual Server Instance. That's how the different Kubernetes resources discover each other. For this example we use the label `app: my-sample`

## 1. Creating a Kubernetes ConfigMap for your IBM Cloud API key

Create a ConfigMap to store the IBM Cloud API key.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-apikey
  labels:
    app: my-sample
data:
  IBMCLOUD_API_KEY: xxx
```

If you have downloaded your `apikey.json` file from the IBM Cloud UI and have the `jq` program installed you may use these commands:

```bash
kubectl create configmap vpc-apikey --from-literal=IBMCLOUD_API_KEY=$(cat ~/apikey.json | jq -r .apikey)
kubectl label configmap vpc-apikey "app=my-sample"
```

## 2. Creating a Kubernetes ConfigMap for your IBM Cloud VPC Configuration

Create a ConfigMap to define the VPC to utilize.  The VSI will be attached to one subnet. Since a subnet is always attached to a VPC and a particular zone, the controller only requires to specify the ID of the subnet, the VPC and the zone will be derived from the subnet.

To gt a list of all Subnet IDs for your current region, you can use the command:

```bash
ibmcloud is subnets
```

Set the `TARGET_SUBNET_ID` field to specify the subnet ID.  

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-config
  labels:
    app: my-sample
data:
  IBMCLOUD_REGION: us-east
  TARGET_SUBNET_ID: "xxx"
```

| ConfigMap Setting | Description |
|-------------------|-------------|
| IBMCLOUD_REGION   | The region where your VPC is located. |
| TARGET_SUBNET_ID  | The subnet where you want to deploy the HPCR VSI. |
| TARGET_IMAGE_NAME (**optional**) | The **name** of the base image for the HPCR VSI. If this is not set, the controller will use the latest HPCR stock image. |
| TARGET_PROFILE (**optional**)   | The **name** of the compute hardware profile to use to create the HPCR VSI. If this is not set, the controller defaults to `bz2e-2x8`. |

## 3. Deploying a Hyper Protect Container Runtime VSI

The HPCR contract is passed as a pre-encrypted string to the custom resource definition. Paste the contract for your workload into the YAML file.

```yaml
apiVersion: hpse.ibm.com/v1
kind: HyperProtectContainerRuntimeVPC
metadata:
  name: vpc-hpcr
spec:
  contract: |
    env: hyper-protect-basic.vNhbxHYgLU.....................cAt/3O20=
    envWorkloadSignature: aN4acEDTeC5u9.....................Q9NjXINA=
    workload: hyper-protect-basic.sKTWp.....................OvqBG9Q==
  targetSelector:
    matchLabels:
      app: my-sample
```

Use `kubectl apply` to create your `HyperProtectContainerRuntimeVPC` resource and watch it create on IBM Cloud!

### Footnotes

1. Each custom resource definition will get a UUID assigned by k8s. The controller uses this UUID to construct the name of the HPCR VSI, i.e. the name of the VSI is not user-friendly.
2. The custom controller is configured to re-validate the state of the VSI every 60s. If the VSI is not in running state (e.g. because it has been deleted manually on VPC) it will be re-created.
3. If your contract uses an OCI image from an outside registry, you may need to add a Public Gateway to your VPC subnet.
