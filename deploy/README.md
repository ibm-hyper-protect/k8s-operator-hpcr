## Installing the Controller

You need a Kubernetes cluster with Internet connectivity.

### 1. Install [Metacontroller](https://metacontroller.github.io/metacontroller/guide/install.html):
  
  ```bash
  kubectl apply -k https://github.com/metacontroller/metacontroller/manifests/production
  ```

### 2. Install the Hyper Protect Virtual Servers Kubernetes Operator
    Generate the manifest files from the manifest diecrtory
    
    ```bash
    kubectl apply https://github.com/ibm-hyper-protect/k8s-operator-hpcr/deploy/manifests
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
