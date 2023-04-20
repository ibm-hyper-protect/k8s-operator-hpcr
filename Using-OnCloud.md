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

## 1. Creating a Kubernetes Secret for your IBM Cloud API key

Create a [Secret](https://kubernetes.io/docs/concepts/configuration/secret/) to store the IBM Cloud API key.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vpc-apikey
  labels:
    app: my-sample
stringData:
  IBMCLOUD_API_KEY: xxx
```

If you have downloaded your `apikey.json` file from the IBM Cloud UI and have the `jq` program installed you may use these commands:

```bash
kubectl create secret generic vpc-apikey --from-literal=IBMCLOUD_API_KEY=$(cat ~/apikey.json | jq -r .apikey)
kubectl label secret vpc-apikey "app=my-sample"
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
  TARGET_SUBNET_ID: "xxx"
```

| ConfigMap Setting | Description |
|-------------------|-------------|
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
4. IBM Cloud® Hyper Protect Virtual Servers v1 are not supported.

## Debugging

After deploying a custom resource of type `HyperProtectContainerRuntimeVPC` the controller will try to create the described VSI instance and will synchronise it state. The state of this process is captured in the `status` field of the `HyperProtectContainerRuntimeVPC` resource as shown:

```yaml
status:
  description: k8s-operator-hpcr-baf43d67-2f16-43b3-b896-290bd32c12fa
  metadata:
    instance: '{"availability_policy":{"host_failure":"restart"},"bandwidth":4000,"boot_volume_attachment":{"device":{"id":"02u7-b7c701b0-4060-43f0-ab09-5595791a8d5e-vdbhf"},"href":"https://br-sao.iaas.cloud.ibm.com/v1/instances/02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706/volume_attachments/02u7-b7c701b0-4060-43f0-ab09-5595791a8d5e","id":"02u7-b7c701b0-4060-43f0-ab09-5595791a8d5e","name":"omnivore-frosting-unbolted-molecule","volume":{"crn":"crn:v1:bluemix:public:is:br-sao-2:a/b3fabd5a6aaf4af09142ad425ffeaee8::volume:r042-69ee8418-4e37-4c1f-8c26-32b84644f72f","href":"https://br-sao.iaas.cloud.ibm.com/v1/volumes/r042-69ee8418-4e37-4c1f-8c26-32b84644f72f","id":"r042-69ee8418-4e37-4c1f-8c26-32b84644f72f","name":"ambitious-capital-luckless-pacific"}},"created_at":"2023-03-28T13:36:28.000Z","crn":"crn:v1:bluemix:public:is:br-sao-2:a/b3fabd5a6aaf4af09142ad425ffeaee8::instance:02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706","disks":[],"href":"https://br-sao.iaas.cloud.ibm.com/v1/instances/02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706","id":"02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706","image":{"crn":"crn:v1:bluemix:public:is:br-sao:a/811f8abfbd32425597dc7ba40da98fa6::image:r042-9d1e6bf1-6161-4392-a9b2-6ab97c71e367","href":"https://br-sao.iaas.cloud.ibm.com/v1/images/r042-9d1e6bf1-6161-4392-a9b2-6ab97c71e367","id":"r042-9d1e6bf1-6161-4392-a9b2-6ab97c71e367","name":"ibm-hyper-protect-container-runtime-1-0-s390x-9"},"lifecycle_reasons":[],"lifecycle_state":"stable","memory":8,"metadata_service":{"enabled":false,"protocol":"http","response_hop_limit":1},"name":"k8s-operator-hpcr-baf43d67-2f16-43b3-b896-290bd32c12fa","network_interfaces":[{"href":"https://br-sao.iaas.cloud.ibm.com/v1/instances/02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706/network_interfaces/02u7-a41c39de-0945-4ae9-8c80-769132ea50e9","id":"02u7-a41c39de-0945-4ae9-8c80-769132ea50e9","name":"cone-swore-trickle-proponent","primary_ip":{"address":"10.250.64.10","href":"https://br-sao.iaas.cloud.ibm.com/v1/subnets/02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd/reserved_ips/02u7-3af34e10-289f-4a0a-b1ba-4a5cf55621ce","id":"02u7-3af34e10-289f-4a0a-b1ba-4a5cf55621ce","name":"neon-hatbox-atom-creation","resource_type":"subnet_reserved_ip"},"resource_type":"network_interface","subnet":{"crn":"crn:v1:bluemix:public:is:br-sao-2:a/b3fabd5a6aaf4af09142ad425ffeaee8::subnet:02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd","href":"https://br-sao.iaas.cloud.ibm.com/v1/subnets/02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd","id":"02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd","name":"r3df970ce505b6f9f229a18cc9e57d511edce85455034e7d0c687d263f6b136","resource_type":"subnet"}}],"primary_network_interface":{"href":"https://br-sao.iaas.cloud.ibm.com/v1/instances/02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706/network_interfaces/02u7-a41c39de-0945-4ae9-8c80-769132ea50e9","id":"02u7-a41c39de-0945-4ae9-8c80-769132ea50e9","name":"cone-swore-trickle-proponent","primary_ip":{"address":"10.250.64.10","href":"https://br-sao.iaas.cloud.ibm.com/v1/subnets/02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd/reserved_ips/02u7-3af34e10-289f-4a0a-b1ba-4a5cf55621ce","id":"02u7-3af34e10-289f-4a0a-b1ba-4a5cf55621ce","name":"neon-hatbox-atom-creation","resource_type":"subnet_reserved_ip"},"resource_type":"network_interface","subnet":{"crn":"crn:v1:bluemix:public:is:br-sao-2:a/b3fabd5a6aaf4af09142ad425ffeaee8::subnet:02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd","href":"https://br-sao.iaas.cloud.ibm.com/v1/subnets/02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd","id":"02u7-41252784-f50b-4d82-bd50-eca9a02bb6fd","name":"r3df970ce505b6f9f229a18cc9e57d511edce85455034e7d0c687d263f6b136","resource_type":"subnet"}},"profile":{"href":"https://br-sao.iaas.cloud.ibm.com/v1/instance/profiles/bz2e-2x8","name":"bz2e-2x8"},"resource_group":{"href":"https://resource-controller.cloud.ibm.com/v2/resource_groups/8bf261ed77b447e3a5d8c7f5dfbc8428","id":"8bf261ed77b447e3a5d8c7f5dfbc8428","name":"hosting-tribe-se"},"resource_type":"instance","startable":true,"status":"running","status_reasons":[],"total_network_bandwidth":3000,"total_volume_bandwidth":1000,"vcpu":{"architecture":"s390x","count":2},"volume_attachments":[{"device":{"id":"02u7-b7c701b0-4060-43f0-ab09-5595791a8d5e-vdbhf"},"href":"https://br-sao.iaas.cloud.ibm.com/v1/instances/02u7_17e574b4-a5b6-45d4-9c1a-d0db40bfc706/volume_attachments/02u7-b7c701b0-4060-43f0-ab09-5595791a8d5e","id":"02u7-b7c701b0-4060-43f0-ab09-5595791a8d5e","name":"omnivore-frosting-unbolted-molecule","volume":{"crn":"crn:v1:bluemix:public:is:br-sao-2:a/b3fabd5a6aaf4af09142ad425ffeaee8::volume:r042-69ee8418-4e37-4c1f-8c26-32b84644f72f","href":"https://br-sao.iaas.cloud.ibm.com/v1/volumes/r042-69ee8418-4e37-4c1f-8c26-32b84644f72f","id":"r042-69ee8418-4e37-4c1f-8c26-32b84644f72f","name":"ambitious-capital-luckless-pacific"}}],"vpc":{"crn":"crn:v1:bluemix:public:is:br-sao:a/b3fabd5a6aaf4af09142ad425ffeaee8::vpc:r042-9ae36eb1-d450-4af1-a846-f8c9fda2ad47","href":"https://br-sao.iaas.cloud.ibm.com/v1/vpcs/r042-9ae36eb1-d450-4af1-a846-f8c9fda2ad47","id":"r042-9ae36eb1-d450-4af1-a846-f8c9fda2ad47","name":"hpcr-tests","resource_type":"vpc"},"zone":{"href":"https://br-sao.iaas.cloud.ibm.com/v1/regions/br-sao/zones/br-sao-2","name":"br-sao-2"}}'
  observedGeneration: 1
  status: 1
```

The `instance` field contains the JSON serialization the VPC VSI representation.