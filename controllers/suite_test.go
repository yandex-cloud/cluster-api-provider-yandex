/*
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
*/

package controllers_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yandex-cloud/cluster-api-provider-yandex/controllers"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client/mock_client"
	"go.uber.org/mock/gomock"
	"golang.org/x/tools/go/packages"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	mockCLient *mock_client.MockClient
	k8sClient  client.Client
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
	scheme     = runtime.NewScheme()
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	crdpaths := getFilePathToCAPICRDs()
	crdpaths = append(crdpaths, filepath.Join("..", "config", "crd", "bases"))
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     crdpaths,
		ErrorIfCRDPathMissing: true,
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = infrav1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = clusterv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: ":8085"},
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to create manager")

	gmc := gomock.NewController(GinkgoT())
	mockCLient = mock_client.NewMockClient(gmc)

	err = (&controllers.YandexClusterReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		YandexClient: mockCLient,
	}).SetupWithManager(ctx, mgr)
	Expect(err).ToNot(HaveOccurred(), "unable to create YandexCluster controller")

	err = (&controllers.YandexMachineReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		YandexClient: mockCLient,
	}).SetupWithManager(ctx, mgr)
	Expect(err).ToNot(HaveOccurred(), "unable to create YandexMachine controller")

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFilePathToCAPICRDs loads CAPI CRD and returns path
// for loaded CRD.
func getFilePathToCAPICRDs() []string {
	packageName := "sigs.k8s.io/cluster-api"
	packageConfig := &packages.Config{
		Mode: packages.NeedModule,
	}

	pkgs, err := packages.Load(packageConfig, packageName)
	if err != nil {
		return nil
	}

	pkg := pkgs[0]

	return []string{
		filepath.Join(pkg.Module.Dir, "config", "crd", "bases"),
		filepath.Join(pkg.Module.Dir, "controlplane", "kubeadm", "config", "crd", "bases"),
	}
}
