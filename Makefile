-include .env
export

GOMAXPROCS ?= 3
CI_MERGE_REQUEST_DIFF_BASE_SHA ?= master

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.25.x
TESTBINS = $(shell pwd)/tests/bin
ENVTEST = $(TESTBINS)/setup-envtest
ENVTEST_BINS = $(TESTBINS)/envtest-binaries
CONTROLLER_GEN = $(TESTBINS)/controller-gen
MOCKGEN = $(TESTBINS)/mockgen
KUSTOMIZE = $(TESTBINS)/kustomize
PATH := $(abspath $(TESTBINS)):$(PATH)

.PHONY: all
all: lint-dev test-dev

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen mockgen ## Generate code containing mocks and DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	@go generate ./...

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	(cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG})
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	@GOBIN=$(TESTBINS) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0

.PHONY: mockgen
mockgen: ## Download controller-gen locally if necessary.
	@GOBIN=$(TESTBINS) go install go.uber.org/mock/mockgen@v0.4.0

.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	@GOBIN=$(TESTBINS) go install sigs.k8s.io/kustomize/kustomize/v5@v5.3.0

.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	@GOBIN=$(TESTBINS) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.17

.PHONY: golangci-lint-local
golangci-lint-local: ## Download golangci-lint locally if necessary.
ifeq (, $(shell which golangci-lint))
	cd $(shell go env GOPATH) && wget -O - -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.59.1
GOLANGCI-LINT-LOCAL=$(shell go env GOPATH)/bin/golangci-lint
else
GOLANGCI-LINT-LOCAL=$(shell which golangci-lint)
endif

lint-dev: golangci-lint-local 
	git -C /tmp/golangci-lint-config pull || git clone ssh://git@gitlab.tcsbank.ru:7999/k8s/golang-template.git /tmp/golangci-lint-config; \
	${GOLANGCI-LINT-LOCAL} -c /tmp/golangci-lint-config/.golangci.kubebuilder.yml run --concurrency ${GOMAXPROCS} --new-from-rev ${CI_MERGE_REQUEST_DIFF_BASE_SHA} --timeout=10m ./... -v ; \

.PHONY: test
test: manifests generate envtest ## Run tests locally.
	@ENV=test CLUSTER_NAME=test KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir $(ENVTEST_BINS) --arch amd64 )" go test -v --count=1  ./...
