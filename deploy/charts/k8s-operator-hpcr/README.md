# Hyper Protect Virtual Servers Kubernetes Operator

## Installing the chart

To add the operator's helm chart to your local helm repository list as `k8s-hpcr-operator`.
  ``` bash
  helm repo add k8s-hpcr-operator https://charts.k8s-hpcr-operator.io
  ```
Install the chart with the release name `k8s-hpcr-operator`:
  ``` bash
  helm install k8s-hpcr-operator k8s-hpcr-operator/k8s-hpcr-operator
  ```

## 4. Uninstalling the Chart
To uninstall `k8s-operator-hpcr` deployment:
  ```bash
  helm uninstall k8s-operator-hpcr
  ```
The command removes all the Kubernetes components associated with the chart and deletes the helm release.