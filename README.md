# Провайдер Yandex Cloud для Kubernetes® Cluster API

`cluster-api-provider-yandex` — провайдер для развертывания кластера Kubernetes в облачной инфраструктуре [Yandex Cloud](https://yandex.cloud) с помощью [Kubernetes Cluster API](https://cluster-api.sigs.k8s.io/).

Кластер разворачивается на базе [виртуальных машин](https://yandex.cloud/ru/docs/compute/concepts/vm) Yandex Compute Cloud и [L7-балансировщика](https://yandex.cloud/ru/docs/application-load-balancer/concepts/application-load-balancer) Yandex Application Load Balancer.

**Преимущества создания кластера с помощью провайдера Yandex Cloud**

* интеграция с API Yandex Cloud;
* декларативный подход к созданию и управлению кластером;
* кластер как [CustomResourceDefinition](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/);
* широкий спектр параметров для конфигурации вычислительных ресурсов кластера;
* [пользовательские образы](#подготовьте-образ-ос-для-узлов-кластера) операционных систем для мастера и узлов;
* пользовательский Control Plane;
* альтернатива [Terraform](https://www.terraform.io/) в CI-процессах.

## Совместимость провайдера с Cluster API

| Версия провайдера | Версия Cluster API | Совместимость |
| :---: | :---: | :---: |
| v1alpha1 | v1beta1 (v1.x) | ✓ |

Чтобы развернуть кластер Kubernetes в Yandex Cloud с помощью Cluster API:
1. [Подготовьте облако к работе](#подготовьте-облако-к-работе).
1. [Настройте окружение](#настройте-окружение).
1. [Подготовьте образ ОС для узлов кластера](#подготовьте-образ-ос-для-узлов-кластера).
1. [Получите Docker-образ с провайдером Yandex Cloud](#получите-docker-образ-с-провайдером-yandex-cloud).
1. [Установите провайдер Yandex Cloud и провайдер Kubernetes Cluster API](#установите-провайдер-yandex-cloud-и-провайдер-kubernetes-cluster-api).
1. [Сформируйте манифесты кластера](#сформируйте-манифесты-кластера).
1. [Разверните кластер](#разверните-кластер).
1. [Подключитесь к кластеру](#подключитесь-к-кластеру).
1. [Установите в созданный кластер CNI](#установите-в-созданный-кластер-cni).

Если созданные ресурсы вам больше не нужны, [удалите](#как-удалить-созданные-ресурсы) их.

## Подготовьте облако к работе

Зарегистрируйтесь в Yandex Cloud и создайте [платежный аккаунт](https://yandex.cloud/ru/docs/billing/concepts/billing-account):

1. Перейдите в [консоль управления](https://console.yandex.cloud/), затем войдите в Yandex Cloud или зарегистрируйтесь.
1. На странице **[Yandex Cloud Billing](https://billing.yandex.cloud/accounts)** убедитесь, что у вас подключен платежный аккаунт, и он находится в [статусе](https://yandex.cloud/ru/docs/billing/concepts/billing-account-statuses) `ACTIVE` или `TRIAL_ACTIVE`. Если платежного аккаунта нет, [создайте](https://yandex.cloud/ru/docs/billing/quickstart/) его и [привяжите](https://yandex.cloud/ru/docs/billing/operations/pin-cloud) к нему облако.

    Если у вас есть активный платежный аккаунт, вы можете создать или выбрать [каталог](https://yandex.cloud/ru/docs/resource-manager/concepts/resources-hierarchy#folder), в котором будет работать ваша инфраструктура, на [странице облака](https://console.yandex.cloud/cloud).

    [Подробнее об облаках и каталогах](https://yandex.cloud/ru/docs/resource-manager/concepts/resources-hierarchy)

### Платные ресурсы

В стоимость поддержки инфраструктуры кластера входят:
* плата за вычислительные ресурсы ВМ, диски и образы (см. [тарифы Yandex Compute Cloud](https://yandex.cloud/ru/docs/compute/pricing));
* плата за хранение образа в бакете и операции с данными (см. [тарифы Yandex Object Storage](https://yandex.cloud/ru/docs/storage/pricing));
* плата за использование вычислительных ресурсов L7-балансировщика (см. [тарифы Yandex Application Load Balancer](https://yandex.cloud/ru/docs/application-load-balancer/pricing));
* (опционально) плата за использование динамического внешнего IP-адреса для вспомогательной ВМ (см. [тарифы Yandex Virtual Private Cloud](https://yandex.cloud/ru/docs/vpc/pricing#prices-public-ip));
* (опционально) плата за использование управляющего кластера (см. [тарифы Yandex Managed Service for Kubernetes](https://yandex.cloud/ru/docs/managed-kubernetes/pricing)).

### Подготовьте инфраструктуру

1. Настройте [сервисный аккаунт](https://yandex.cloud/ru/docs/iam/concepts/users/service-accounts) Yandex Cloud:
    1. [Создайте](https://yandex.cloud/ru/docs/iam/operations/sa/create) сервисный аккаунт, от имени которого будут создаваться ресурсы кластеры.
    1. [Назначьте](https://yandex.cloud/ru/docs/iam/operations/sa/assign-role-for-sa) сервисному аккаунту роли [compute.editor](https://yandex.cloud/ru/docs/compute/security/#compute-editor) и [alb.editor](https://yandex.cloud/ru/docs/application-load-balancer/security/#alb-editor) на каталог.
    1. [Получите](https://yandex.cloud/ru/docs/iam/operations/authorized-key/create) авторизованный ключ для сервисного аккаунта в формате JSON.
1. Если в вашем каталоге еще нет [облачной сети](https://yandex.cloud/ru/docs/vpc/concepts/network#network) Virtual Private Cloud, [создайте](https://yandex.cloud/ru/docs/vpc/operations/network-create) ее. Также создайте [подсеть](https://yandex.cloud/ru/docs/vpc/operations/subnet-create).
1. Инфраструктуре создаваемого кластера в Virtual Private Cloud назначается [группа безопасности](https://yandex.cloud/ru/docs/vpc/concepts/security-groups) по умолчанию. [Добавьте](https://yandex.cloud/ru/docs/vpc/operations/security-group-add-rule) в эту группу следующие правила для _входящего_ трафика:

    | Протокол | Диапазон портов | Тип источника | Источник | Описание |
    | --- | --- | --- | --- | --- |
    | `TCP` | `0-65535` | `Группа безопасности` | `Balancer` | Проверки состояния L7-балансировщиком |
    | `Any` | `8443` | `CIDR` | `0.0.0.0/0` | Доступ к Kubernetes API |

1. Создаваемый кластер будет доступен в облачной сети по [внутреннему IP-адресу](https://yandex.cloud/ru/docs/vpc/concepts/address#internal-addresses). Чтобы обеспечить удаленный доступ в кластер, [создайте](https://yandex.cloud/ru/docs/compute/operations/vm-create/create-linux-vm) вспомогательную ВМ в той же сети, в которой будет развернут кластер, и с той же группой безопасности. Установите на ВМ [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl).

1. Создайте _управляющий_ [кластер](https://yandex.cloud/ru/docs/managed-kubernetes/operations/kubernetes-cluster/kubernetes-cluster-create) и [группу узлов](https://yandex.cloud/ru/docs/managed-kubernetes/operations/node-group/node-group-create) Yandex Managed Service for Kubernetes. Из этого кластера будут осуществляться развертывание нового кластера с помощью Cluster API и управление кластерной инфраструктурой.

    Также вы можете развернуть управляющий кластер локально, например с помощью утилиты [kind](https://kind.sigs.k8s.io/).

> [!IMPORTANT]
> Чтобы иметь возможность загружать Docker-образ c провайдером Yandex Cloud из [реестра](https://yandex.cloud/ru/docs/container-registry/concepts/registry) Yandex Container Registry, у управляющего кластера должен быть доступ в интернет. Например, вы можете [настроить NAT-шлюз](https://yandex.cloud/ru/docs/vpc/operations/create-nat-gateway) в подсети управляющего кластера.

## Настройте окружение

1. Установите следующие инструменты:
    * [Go](https://go.dev/doc/install) версии 1.22.0 и выше;
    * [gomock](https://github.com/golang/mock#installation) версии 1.6.0;
    * [docker](https://www.docker.com/) версии 17.03 и выше;
    * [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) версии 1.11.3 и выше;
    * [clusterctl](https://cluster-api.sigs.k8s.io/user/quick-start#install-clusterctl) версии 1.5.0 и выше.

1. Настройте для `kubectl` доступ к управляющему кластеру Kubernetes:
    * [Managed Service for Kubernetes](https://yandex.cloud/ru/docs/managed-kubernetes/operations/connect/#kubectl-connect);
    * [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#interacting-with-your-cluster).

1. Склонируйте репозиторий `cluster-api-provider-yandex` и перейдите в директорию с проектом:

    ```bash
    git clone https://github.com/yandex-cloud/cluster-api-provider-yandex.git
    cd cluster-api-provider-yandex
    ```

## Подготовьте образ ОС для узлов кластера

[Образ](https://yandex.cloud/ru/docs/compute/concepts/image) ОС, который будет развернут на узлах создаваемого кластера, должен быть подготовлен для работы с Kubernetes Cluster API, а также совместим с Compute Cloud.

Вы можете использовать готовый образ ОС на основе Ubuntu 24.04, подготовленный нами для работы с Kubernetes версии 1.31.4. Для этого при [формировании манифеста кластера](#сформируйте-манифесты-кластера) в переменной `YANDEX_CONTROL_PLANE_MACHINE_IMAGE_ID` укажите идентификатор образа `fd8a3kknu25826s8hbq3`.

> [!IMPORTANT]
> Образ создан исключительно в ознакомительных целях, использовать его в промышленной эксплуатации не рекомендуется.

Вы можете подготовить собственный образ ОС, следуя инструкции ниже:

1. [Соберите](https://image-builder.sigs.k8s.io/capi/capi) образ ОС с помощью утилиты [Image Builder](https://github.com/kubernetes-sigs/image-builder).

    См. также [Подготовить образ диска для Compute Cloud](https://yandex.cloud/ru/docs/compute/operations/image-create/custom-image).

1. [Загрузите](https://yandex.cloud/ru/docs/compute/operations/image-create/upload) образ в Compute Cloud и сохраните его идентификатор.

## Получите Docker-образ с провайдером Yandex Cloud

Вы можете [использовать готовый Docker-образ](#использовать-готовый-docker-образ) с провайдером Yandex Cloud из публичного [реестра](https://yandex.cloud/ru/docs/container-registry/concepts/registry) Container Registry или [собрать его самостоятельно](#собрать-docker-образ-из-исходного-кода) из исходного кода.

### Использовать готовый Docker-образ

Добавьте в переменную окружения `IMG` путь к Docker-образу с провайдером Yandex Cloud в публичном реестре:

```bash
export IMG=cr.yandex/crpsjg1coh47p81vh2lc/capy/cluster-api-provider-yandex:latest
```

### Собрать Docker-образ из исходного кода

1. [Создайте](https://yandex.cloud/ru/docs/container-registry/operations/registry/registry-create) реестр Container Registry и сохраните его идентификатор.
1. [Аутентифицируйтесь](https://yandex.cloud/ru/docs/container-registry/operations/authentication#cred-helper) в реестре Container Registry с помощью Docker credential helper.
1. Добавьте в переменную окружения `IMG` путь, по которому собранный Docker-образ будет сохранен в реестре:

    ```bash
    export IMG=cr.yandex/<идентификатор_реестра>/cluster-api-provider-yandex:<тег>
    ```

1. Если вы собираете Docker-образ на компьютере с архитектурой, отличной от [AMD64](https://ru.wikipedia.org/wiki/X86-64), отредактируйте в [Makefile](Makefile) блок `docker-build`:

    ```text
    docker build --platform linux/amd64 -t ${IMG} .
    ```

1. Запустите Docker daemon.

1. Соберите Docker-образ и загрузите его в реестр:

    ```bash
    make docker-build docker-push
    ```

## Установите провайдер Yandex Cloud и провайдер Kubernetes Cluster API

1. Инициализируйте управляющий кластер:

    ```bash
    clusterctl init
    ```

    В управляющий кластер будут установлены основные компоненты Kubernetes Cluster API, а также [cert-manager](https://cert-manager.io/).

1. Создайте в управляющем кластере [CustomResourceDefinitions](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/) для создаваемого кластера:

    ```bash
    make install
    ```

1. Получите список установленных CRD:

    ```bash
    kubectl get crd | grep cluster.x-k8s.io
    ```

    Чтобы получить манифест конкретного CRD, выполните команду:

    ```bash
    kubectl get crd <имя_CRD> \
      --output yaml
    ```

1. Создайте пространство имен для провайдера Yandex Cloud:

    ```bash
    kubectl create namespace capy-system
    ```

1. Создайте секрет с авторизованным ключом сервисного аккаунта Yandex Cloud:

    ```bash
    kubectl create secret generic yc-sa-key \
      --from-file=key=<путь_к_файлу_с_авторизованным_ключом> \
      --namespace capy-system

1. Установите провайдер Yandex Cloud:

    ```bash
    make deploy
    ```

## Сформируйте манифесты кластера

1. Получите идентификаторы ресурсов Yandex Cloud для развертывания кластера:
    * [образ ОС](https://yandex.cloud/ru/docs/compute/operations/image-control/image-control-get-info)
    * [каталог](https://yandex.cloud/ru/docs/resource-manager/operations/folder/get-id)
    * [сеть](https://yandex.cloud/ru/docs/vpc/operations/network-get-info)
    * [подсеть](https://yandex.cloud/ru/docs/vpc/operations/subnet-get-info)

1. Передайте идентификаторы ресурсов в переменные окружения:

    ```bash
    export YANDEX_CONTROL_PLANE_MACHINE_IMAGE_ID=<идентификатор_образа>
    export YANDEX_FOLDER_ID=<идентификатор_каталога>
    export YANDEX_NETWORK_ID=<идентификатор_сети>
    export YANDEX_SUBNET_ID=<идентификатор_подсети>
    export YANDEX_ZONE_ID=<идентификатор_зоны>
    ```

1. Сформируйте манифесты кластера:

    ```bash
    clusterctl generate cluster <имя_создаваемого_кластера> \
      --from templates/cluster-template.yaml > /tmp/capy-cluster.yaml
    ```

    По умолчанию согласно сгенерированному манифесту, для доступа в кластер будет развернут [L7-балансировщик](https://yandex.cloud/ru/docs/application-load-balancer/concepts/application-load-balancer) Application Load Balancer c динамическим внутренним IP-адресом. Вы можете [присвоить L7-балансировщику фиксированный IP-адрес](#Опционально-настройте-эндпоинт-api-сервера).

> [!IMPORTANT]
> После создания кластера, присвоить L7-балансировщику фиксированный IP-адрес будет нельзя.

### (Опционально) Настройте эндпоинт API-сервера

Задайте в менифесте `YandexCluster` следующие параметры для L7-балансировщика:

```yaml
  loadBalancer:
    listener:
      address: <фиксированный_IP-адрес_из_диапазона_подсети>
      subnet:
        id: <идентификатор_подсети>
```

## Разверните кластер

```bash
kubectl apply -f /tmp/capy-cluster.yaml
```

За процессом развертывания кластера можно следить в [консоли управления](https://console.yandex.cloud/) Yandex Cloud, а также в логах пода `capy-controller-manager`:

```bash
kubectl logs <имя_пода_с_capy-controller-manager> \
  --namespace capy-system \
  --follow
```

## Подключитесь к кластеру

Реквизиты для подключения к новому кластеру будут созданы в управляющем кластере в секрете `<имя_создаваемого_кластера>-kubeconfig`.

1. Получите данные из секрета:

    ```bash
    kubectl get secret <имя_создаваемого_кластера>-kubeconfig \
      --output yaml | yq -r '.data.value' | base64 \
      --decode > capy-cluster-config
    ```

1. Передайте на ВМ, находящейся в той же сети, в которой расположен новый кластер, файл с конфигурацией для `kubectl`:

    ```bash
    scp <путь_к_файлу_capy-cluster-config_на_локальном_компьютере> \
    <имя_пользователя>@<публичный_IP-адрес_ВМ>:/home/<имя_пользователя>/.kube/config
    ```

1. [Подключитесь](https://yandex.cloud/ru/docs/compute/operations/vm-connect/ssh) к ВМ по SSH.

1. Подключитесь к новому кластеру:

    ```bash
    kubectl cluster-info
    ```

## Установите в созданный кластер CNI

Чтобы обеспечить сетевую функциональность для подов в новом кластере, установите в него [Container Network Interface](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/), например [Cilium](https://github.com/cilium/cilium) или [Calico](https://github.com/projectcalico/calico).

Подробнее см. в документации:
* [Cilium Quick Installation](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/)
* [Quickstart for Calico on Kubernetes](https://docs.tigera.io/calico/latest/getting-started/kubernetes/quickstart)

## Как удалить созданные ресурсы

### Удалите кластер

```bash
kubectl delete -f /tmp/capy-cluster.yaml
```

### Удалите CRD из управляющего кластера

```bash
make uninstall
```

### Удалите контроллер провайдера Yandex Cloud из управляющего кластера

```sh
make undeploy
```

### Удалите вспомогательные ресурсы Yandex Cloud

Чтобы перестать платить за вспомогательные ресурсы, если вы их создавали, удалите:
* [Группу узлов Managed Service for Kubernetes](https://yandex.cloud/ru/docs/managed-kubernetes/operations/node-group/node-group-delete)
* [Кластер Managed Service for Kubernetes](https://yandex.cloud/ru/docs/managed-kubernetes/operations/kubernetes-cluster/kubernetes-cluster-delete)
* [ВМ Compute Cloud](https://yandex.cloud/ru/docs/compute/operations/vm-control/vm-delete)
* [Образ ОС в Compute Cloud](https://yandex.cloud/ru/docs/compute/operations/image-control/delete)
* [Образ ОС в Object Storage](https://yandex.cloud/ru/docs/storage/operations/objects/delete)
* [Бакет Object Storage](https://yandex.cloud/ru/docs/storage/operations/buckets/delete)

## См. также

* [Apache-2.0 license](LICENSE)
* [Соглашение с контрибьютором](CONTRIBUTING.md)
* [Kubernetes Cluster API](https://cluster-api.sigs.k8s.io/)
* [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)
