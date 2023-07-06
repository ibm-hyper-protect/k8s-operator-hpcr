# Installing the Controller

You need a Kubernetes cluster with Internet connectivity.

## 1. Install [Metacontroller](https://metacontroller.github.io/metacontroller/guide/install.html):
  
  ```bash
  kubectl apply -k https://github.com/metacontroller/metacontroller/manifests/production
  ```

## 2. Install the Hyper Protect Virtual Servers Kubernetes Operator
The operator is installed via its helm chart.
Add the operator's helm chart to your local helm repository list as `k8s-hpcr-operator`.
  ``` bash
  helm repo add k8s-hpcr-operator https://github.io/ibm-hyper-protect/k8s-operator-hpcr/deploy/charts/k8s-hpcr-operator
  ```
Install the chart with the release name `k8s-hpcr-operator`:
  ``` bash
  helm install k8s-hpcr-operator k8s-hpcr-operator/k8s-hpcr-operator
  ```

### Generate my own Manifest files
The static deployment manifests would be generated from the helm chart and bundled as part of a release on github.
You can also generate your own static deployment manifests on your local workstation, using helm and make. 
The default deployment values can be overwrtitten by customizing the `helm-values.yaml` file.
  ```bash
  make manifests
  ``` 

## 3. Verify your installation by checking for the existence of the custom resources

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
kubectl get deployments -n k8s-hpcr-operator

NAME                READY   UP-TO-DATE   AVAILABLE   AGE
k8s-operator-hpcr   1/1     1            1           6m35s
```

## Show Logs

  ```bash
  kubectl logs -l app=k8s-operator-hpcr -n k8s-hpcr-operator
  ```

## 4. Uninstalling the Chart
To uninstall the k8s-operator-hpcr deployment via helm:
  ```bash
  helm uninstall k8s-operator-hpcr
  ```
The command removes all the Kubernetes components associated with the chart and deletes the helm release.