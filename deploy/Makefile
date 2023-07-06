.PHONY: manifests
manifests: 
	helm install k8s-operator-hpcr-deployment ../charts/k8s-operator-hpcr -f helm-values.yaml  --create-namespace --dry-run
