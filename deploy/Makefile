.PHONY: manifests
manifests: 
	helm template k8s-operator-hpcr-deployment charts/k8s-operator-hpcr -f helm-values.yaml  > manifests/manifests.yaml

.PHONY: kustomize-install
kustomize-install: 
	kubectl apply -k manifests
