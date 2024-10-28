# cluster-api-provider-yandex
Infrastructure Cluster API provider for YandexCloud

## Description
Provide an opportunity to deploy kubernetes clusters based on YandexCloud Compute via Cluster API 

## Compatibility with Cluster API
This provider's versions are compatible with the following versions of Cluster API:
|             |Cluster API v1beta1 (v1.x)|
|:------------|:------------------------:|
|CAPY v1alpha1|✓|

## Getting Started

### Prerequisites
#### Requirements

- go version v1.21.0+
- docker version 17.03+
- kubectl version v1.11.3+
- clusterctl version v1.5.0+
- Access to a Kubernetes v1.11.3+ cluster.

#### Service Account
To create and manage clusters, this infrastructure provider uses a service account to authenticate with YC’s APIs.

Follow [these instructions](https://yandex.cloud/en/docs/iam/operations/sa/create) to create a new service account. Then [add](https://yandex.cloud/en/docs/iam/operations/sa/assign-role-for-sa#binding-role-resource) the following roles to the service account:
- [compute.editor](https://yandex.cloud/en/docs/compute/security/#compute-editor) 
- [alb.editor](https://yandex.cloud/en/docs/iam/roles-reference#alb-editor)

Next, [generate a JSON key](https://yandex.cloud/en/docs/iam/operations/authorized-key/create) and store it in a safe place.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
export IMG=<some-registry>/cluster-api-provider-yandex:tag
make docker-build docker-push
```

**Install the CRDs into the cluster:**

```sh
make install
```

**Create secret with YC key**
```
kubectl create secret generic yc-sa-key --from-file=</path/to/serviceaccount-key.json> -n capy-system
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy
```

**Generate workload cluster manifests**
```sh
export YANDEX_CONTROL_PLANE_MACHINE_IMAGE_ID=<compute-image-id>
export YANDEX_FOLDER_ID=<folder-id>
export YANDEX_SUBNET_ID=<subnet-id>
export YANDEX_NETWORK_ID=<network-id>
clusterctl generate cluster <some-cluster-name> --from templates/cluster-template.yaml > /tmp/capy-cluster.yaml
```

**Deploy generated cluster**

```sh
kubectl apply -k /tmp/capy-cluster.yaml
```

**Install CCM**
See https://github.com/deckhouse/yandex-cloud-controller-manager

**Install CNI**
See https://github.com/cilium/cilium

### To Uninstall
**Delete cluster CRs:**

```sh
kubectl delete -k /tmp/capy-cluster.yaml
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
