# Using k8s-operator-hpcr for IBM Cloud

Now that you have installed the Hyper Protect Virtual Servers Kubernetes Operator, create Kubernetes artifacts that will be used to control your IBM Cloud account. 

## 1. Creating a Kubernetes ConfigMap for your IBM Cloud API key

Ensure you have an `IC_API_KEY` environment variable set up with your IBM Cloud API key
- this will likelly require a paying account
- you can create an API account by visiting the [IBM Cloud API keys page](https://cloud.ibm.com/iam/apikeys). Ensure you have selected the account you want to use before creating the key as the key will be associtated to the account you have selected at the time of creation.
- If you have downloaded your `apikey.json` file from the IBM Cloud UI you may use this command:
  `export IC_API_KEY=$(cat ~/apikey.json | jq -r .apikey)`

```bash
kubectl create configmap vpc-apikey-configmap --from-literal=IBMCLOUD_API_KEY=${IC_API_KEY} && kubectl label configmap vpc-apikey-configmap "hpcr=test"
```

## 2. Creating a Kubernetes ConfigMap for your IBM Cloud VPC Configuration

Create a ConfigMap `vpc-deployment-configmap` to define the VPC to utilize.  The VSI will be attached to one subnet. Since a subnet is always attached to a VPC and a particular zone, the controller only requires to specify the ID of the subnet, the VPC and the zone will be derived from the subnet.

To gt a list of all Subnet IDs for your current region, you can use the command:

```bash
ibmcloud is subnets
```

Set the `TARGET_SUBNET_ID` field to specify the subnet ID.  

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vpc-deployment-configmap
  labels:
    hpcr: test
data:
  IBMCLOUD_REGION: us-east
  TARGET_SUBNET_ID: "xxx"
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
```

| ConfigMap Setting | Description |
|-------------------|-------------|
| IBMCLOUD_REGION   | The region where your VPC is located. |
| TARGET_SUBNET_ID  | The subnet where you want to deploy the HPCR VSI. | 
| TARGET_IMAGE_NAME | The **name** of the base image for the HPCR VSI. If this is not set, the controller will use the latest HPCR stock image. |
| TARGET_PROFILE    | The **name** of the compute hardware profile to use to create the HPCR VSI. If this is not set, the controller defaults to `bz2e-2x8`. |
| TARGET_CONTRACT_PUB_KEY_FILENAME | tbd |
| IBMCLOUD_ZONE  (**Depreciated**)   | The zone within the VPC where you want to deploy the HPCR VSI. The controller only requires to specify the ID of the subnet, the VPC and the zone will be derived from the subnet.  |
| TARGET_VPC_NAME  (**Depreciated**)   | The VPC where you want to deploy the HPCR VSI. The controller only requires to specify the ID of the subnet, the VPC and the zone will be derived from the subnet.  |


## 3. Deploying a Hyper Protect Container Runtime VSI

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

Use `kubectl apply` to create your `HyperProtectContainerRuntime` and watch it create on IBM Cloud!

### Footnotes

1. Each custom resource definition will get a UUID assigned by k8s. The controller uses this UUID to construct the name of the HPCR VSI, i.e. the name of the VSI is not user-friendly.
2. The custom controller is configured to re-validate the state of the VSI every 60s. If the VSI is not in running state (e.g. because it has been deleted manually on VPC) it will be re-created. 
