# Cluster API infrastructure provider for Yandex Cloud

## Summary

[Kubernetes Cluster API](https://cluster-api.sigs.k8s.io/introduction) is opensource cluster lifecycle management tool for platform operators.

This repo is implementation of Cluster API [infrastructure provider](https://cluster-api.sigs.k8s.io/reference/providers#infrastructure) on top of [Public Yandex Cloud API](https://cloud.yandex.ru/ru/docs/overview/api).

## Features
* Declarative kubernetes cluster on Yandex Cloud
* Automatical kube api LB target group reconciliation


## How to use
### Common Prerequisites
* Install clusterctl
* Install kubectl
* Install helm
* Install kind
* Install yc

### Service Account
To create and manage clusters, this infrastructure provider uses a service account to authenticate with YC’s APIs.

Follow [these instructions](https://yandex.cloud/en/docs/iam/operations/sa/create) to create a new service account. Then [add](https://yandex.cloud/en/docs/iam/operations/sa/assign-role-for-sa#binding-role-resource) the following roles to the service account:
- [compute.editor](https://yandex.cloud/en/docs/compute/security/#compute-editor) 
- [alb.editor](https://yandex.cloud/en/docs/iam/roles-reference#alb-editor)

### Start kind or choose management cluster
```
kind create cluster
```
or
```
export KUBECONFIG=<actual-management-cluster-kubeconfig>
```

### Initialize management cluster
```
clusterctl init
```

### Build docker image
```
export IMG=<image>
make docker-build
```

### Push or load docker image
```
make docker-push
```
or if using kind:
```
kind load docker-image ${IMG}
```

### Deploy CAPY
```
make deploy
```

### Create secret with YC key
```
yc iam key create --service-account-id=<your-service-account> --description=<description> -o /tmp/key
kubectl create secret generic yc-sa-key --from-file=/tmp/key -n capy-system
```

### Create Application Load Balancer
Use Yandex Cloud UI https://console.yandex.cloud

### Download OS image to your Yandex Cloud folder
Note that image must meet the kubeadm host requirements, more information on https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/
```
yc compute image create --name <some-image-name> --source-uri <object-storage-uri>
```

### Generate workload cluster manifests
Note that you need to set YANDEX_SUBNET_ID from zone equal to YANDEX_ZONE_ID.
```
export YANDEX_CONTROL_PLANE_ENDPOINT_HOST=<app-loadbalancer-listener-ip>
export YANDEX_CONTROL_PLANE_MACHINE_IMAGE_ID=<os-image-id>
export YANDEX_CONTROL_PLANE_TARGET_GROUP_ID=<app-loadbalancer-target-group-id>
export YANDEX_FOLDER_ID=<some-yandex-cloud-folder-id>
export YANDEX_SUBNET_ID=<some-yandex-cloud-subnet-id>
export YANDEX_ZONE_ID=<some-yandex-cloud-subnet-id>
clusterctl generate cluster <some-cluster-name> --from templates/cluster-template.yaml > workload-cluster.yaml
```

### Apply the workload cluster
```
kubectl apply -f workload-cluster.yaml
```

### Install Cloud Controller Manager
For example, Deckhouse https://github.com/deckhouse/yandex-cloud-controller-manager

### Install CNI
For example, Cilium https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/

## Clean Up

### Delete workload cluster
```
kubectl delete cluster <cluster-name>
```

### Delete kind cluster (if created)
```
kind delete cluster
```

### Delete Load Balancer
```
yc alb load-balancer delete <load-balancer-name>
yc alb backend-group delete <backend-group-id>
yc alb target-group delete <target-group-id>
```

### Delete image
```
yc compute image delete <image-name>
```

### Internal navigation
* [Contributors Guide](CONTRIBUTING.md)
* [Style Guide](CONTRIBUTING.md#style-guide)