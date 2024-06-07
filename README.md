# Cluster API infrastructure provider for Yandex Cloud

## Summary

[Kubernetes Cluster API](https://cluster-api.sigs.k8s.io/introduction) is opensource cluster lifecycle management tool for platform operators.

This repo is implementation of Cluster API [infrastructure provider](https://cluster-api.sigs.k8s.io/reference/providers#infrastructure) on top of [Public Yandex Cloud API](https://cloud.yandex.ru/ru/docs/overview/api).

## Features
* Declarative kubernetes cluster on Yandex Cloud
* Support both LB types: ALB and NLB
* Automatical kube api LB target group reconciliation


## How to use
### Common Prerequisites
* Install clusterctl
* Install kubectl
* Install helm
* Install kind
* Install yc

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

### Push docker image
```
make docker-push
```

### Deploy CAPY
```
make deploy
```

### Create Load Balancer
```
yc alb backend-group create <some-backend-group-name>
yc alb backend-group create <some-target-group-name>
yc alb backend-group add-stream-backend testbg --name  --port 8443 --target-group-id <target-group-id> --stream-healthcheck 
yc alb backend-group add-stream-backend --backend-group-name <backend-group-name>  --name <some-backend-name> --port 8443 --target-group-id <target-group-id> --stream-healthcheck port=8443,timeout=1s,interval=1s
yc alb load-balancer create <some-lb-name> --network-id <suitable-network-id> --location subnet-id=<suitable-subnet-id>,zone=<suitable-zone>
yc alb load-balancer add-stream-listener <lb-name> --listener-name <some-listener-name> --internal-ipv4-endpoint port=8443,subnet-id=<suitable-subnet-id> --backend-group-id <backend-group-id>
```
or use Yandex Cloud UI https://console.yandex.cloud/

### Download OS image to your Yandex Cloud folder
```
yc compute image create --name <some-image-name> --source-uri <object-storage-uri>
```

### Generate workload cluster manifests
```
export YANDEX_CONTROL_PLANE_ENDPOINT_HOST=<app-loadbalancer-listener-ip>
export YANDEX_CONTROL_PLANE_MACHINE_IMAGE_ID=<os-image-id>
export YANDEX_CONTROL_PLANE_TARGET_GROUP_ID=<app-loadbalancer-target-group-id>
export YANDEX_FOLDER_ID=<some-yandex-cloud-folder-id>
export YANDEX_SUBNET_ID=<some-yandex-cloud-subnet-id>
clusterctl generate cluster <some-cluster-name> --from template/cluster-template.yaml > workload-cluster.yaml
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
```

### Delete image
```
yc compute image delete <image-name>
```

### Internal navigation
* [Contributors Guide](CONTRIBUTING.md)
* [Style Guide](CONTRIBUTING.md#style-guide)