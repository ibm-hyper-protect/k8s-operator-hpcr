# Deployment files

The static deployment manifests would be generated from the helm chart and bundled as part of a release on github.

## Generate my own Manifest files
You can generate your own static deployment manifests on your local workstation, using helm and make.
```bash
make manifests
``` 

### 2. To install on your cluster 
```bash
kubectl apply -k https://github.com/ibm-hyper-protect/k8s-operator-hpcr/manifests
``` 