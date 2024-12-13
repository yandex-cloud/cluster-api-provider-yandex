/*
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

package controllers //nolint:testpackage // private variables access

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client/mock_client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	loadbalancer "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/services/loadbalancers"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("YandexMachine reconciliation check", func() {
	BeforeEach(func() {
		var err error

		e = ClusterTestEnv{
			Client:           k8sClient,
			clusterName:      "machine-check-cluster",
			machineName:      "test",
			secretName:       "test",
			reconcileTimeout: 3 * time.Second,
		}
		//+kubebuilder:scaffold:webhook
		e.controller = gomock.NewController(GinkgoT())
		e.mockClient = mock_client.NewMockClient(e.controller)
		e.mockClientBuilder = mock_client.NewMockBuilder(e.controller)
		e.mockClientBuilder.EXPECT().GetDefaultClient(gomock.Any()).
			DoAndReturn(func(_ context.Context) (*mock_client.MockClient, error) {
				return e.mockClient, nil
			}).AnyTimes()
		e.mockClient.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()
		testNamespace, err = e.CreateNamespace(ctx, "machine-check")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(e.DeleteNamespace(ctx)).To(Succeed())
		e.controller.Finish()
	})

	When("Creating an empty YandexMachine", func() {
		It("should not fail", func() {
			Expect(e.Create(ctx, e.getYandexMachine(testNamespace.Name))).To(Succeed())
		})
	})

	When("Creating an YandexMachine", func() {
		It("should not reconcile YandexMachine without parent CAPI Machine", func() {
			ym := e.getYandexMachine(testNamespace.Name)
			Expect(e.Create(ctx, ym)).To(Succeed())

			reconciler := &YandexMachineReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
			}

			result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should not reconcile YandexMachine without parent YandexCluster", func() {
			Expect(e.Create(ctx, e.getMachineWithInfrastructureRef(testNamespace.Name))).To(Succeed())
			ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
			Expect(e.Create(ctx, ym)).To(Succeed())

			reconciler := &YandexMachineReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
			}

			result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should not reconcile YandexMachine with paused CAPI Cluster", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			cc.Spec.Paused = true
			Expect(e.Create(ctx, cc)).To(Succeed())
			Expect(e.Create(ctx, e.getYandexClusterWithOwnerReference(testNamespace.Name))).To(Succeed())
			Expect(e.Create(ctx, e.getMachineWithInfrastructureRef(testNamespace.Name))).To(Succeed())
			ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
			Expect(e.Create(ctx, ym)).To(Succeed())

			reconciler := &YandexMachineReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
			}
			result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
		})
	})

	It("should not error and get ready status eventually", func() {
		Expect(e.Create(ctx, e.getCAPIClusterWithInfrastructureReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getYandexClusterWithOwnerReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getMachineWithInfrastructureRef(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getBootstrapSecret(testNamespace.Name))).To(Succeed())
		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		Expect(e.Create(ctx, ym)).To(Succeed())

		reconciler := &YandexMachineReconciler{
			Client:              k8sClient,
			YandexClientBuilder: e.mockClientBuilder,
		}

		addr := "1.2.3.4"
		e.setNewYandexMachineReconcileMocks(addr)
		result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).NotTo(HaveOccurred())

		// On the first reconcile the YandexMachine have to be in STARTING status.
		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err = e.Get(ctx, key, ym)
			return (err == nil &&
				ym.Status.InstanceStatus != nil &&
				*ym.Status.InstanceStatus == infrav1.InstanceStatusStarting)
		}, e.reconcileTimeout).Should(BeTrue())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		Expect(ym.Status.Ready).To(BeFalse())
		Expect(ym.GetFinalizers()).To(Equal([]string{infrav1.MachineFinalizer}))

		result, err = reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).NotTo(HaveOccurred())

		// On the second reconcile the YandexMachine have to be READY.
		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err := e.Get(ctx, key, ym)
			return (err == nil && ym.Status.Ready == true)
		}, e.reconcileTimeout).Should(BeTrue())
		Expect(result.Requeue).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
		Expect(ym.Status.Addresses[0].Address).To(Equal(addr))
	})

	It("should not error and add controlplane node to application load balancer in cluster with ALB", func() {
		Expect(e.Create(ctx, e.getCAPIClusterWithInfrastructureReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getYandexClusterWithOwnerReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getCPMachineWithInfrastructureRef(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getBootstrapSecret(testNamespace.Name))).To(Succeed())
		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		Expect(e.Create(ctx, ym)).To(Succeed())

		addr := "1.2.3.4"
		tgName := "targetgroup"
		e.setNewCPYandexMachineReconcileMocks(addr, tgName)
		reconciler := &YandexMachineReconciler{
			Client:              k8sClient,
			YandexClientBuilder: e.mockClientBuilder,
		}

		result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).NotTo(HaveOccurred())

		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err := e.Get(ctx, key, ym)
			return (err == nil && ym.Status.Ready == true)
		}, e.reconcileTimeout).Should(BeTrue())

		Expect(result.Requeue).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
		Expect(ym.Status.Addresses[0].Address).To(Equal(addr))
	})

	It("should error and start machine creation again on api error", func() {
		Expect(e.Create(ctx, e.getCAPIClusterWithInfrastructureReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getYandexClusterWithOwnerReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getMachineWithInfrastructureRef(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getBootstrapSecret(testNamespace.Name))).To(Succeed())
		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		Expect(e.Create(ctx, ym)).To(Succeed())

		reconciler := &YandexMachineReconciler{
			Client:              k8sClient,
			YandexClientBuilder: e.mockClientBuilder,
		}

		e.setNewYandexMachineErrorReconcileMocks()
		// got machine creation api error.
		result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).To(HaveOccurred())
		Expect(result.RequeueAfter).To(BeZero())
		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err = e.Get(ctx, key, ym)
			return (err == nil &&
				ym.Status.Conditions != nil &&
				ym.Status.Conditions[0].Reason == infrav1.ConditionStatusNotfound)
		}, e.reconcileTimeout).Should(BeTrue())
		Expect(ym.Spec.ProviderID).To(BeNil())
		Expect(ym.Status.Ready).To(BeFalse())

		// Machine creation API call ok,  YandexMachine have to be in STARTING status.
		result, err = reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err = e.Get(ctx, key, ym)
			return (err == nil &&
				ym.Status.InstanceStatus != nil &&
				*ym.Status.InstanceStatus == infrav1.InstanceStatusStarting)
		}, e.reconcileTimeout).Should(BeTrue())
		Expect(ym.Status.Ready).To(BeFalse())
		Expect(ym.GetFinalizers()).To(Equal([]string{infrav1.MachineFinalizer}))
	})

	It("should error and retry to add node to ALB target group on load balancer api error", func() {
		Expect(e.Create(ctx, e.getCAPIClusterWithInfrastructureReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getYandexClusterWithOwnerReference(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getCPMachineWithInfrastructureRef(testNamespace.Name))).To(Succeed())
		Expect(e.Create(ctx, e.getBootstrapSecret(testNamespace.Name))).To(Succeed())
		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		Expect(e.Create(ctx, ym)).To(Succeed())

		addr := "1.2.3.4"
		tgName := "targetgroup"
		e.setNewCPYandexMachineErrorReconcileMocks(addr, tgName)
		reconciler := &YandexMachineReconciler{
			Client:              k8sClient,
			YandexClientBuilder: e.mockClientBuilder,
		}

		// target group add error.
		result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).To(HaveOccurred())

		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err = e.Get(ctx, key, ym)
			return (err == nil && ym.Status.Ready == false)
		}, e.reconcileTimeout).Should(BeTrue())

		Expect(result.Requeue).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
		Expect(ym.Status.Addresses[0].Address).To(Equal(addr))

		// target group add succeed.
		result, err = reconciler.Reconcile(ctx, e.getReconcileRequest(ym.Namespace, ym.Name))
		Expect(err).NotTo(HaveOccurred())

		ym = &infrav1.YandexMachine{}
		Eventually(func() bool {
			key := client.ObjectKey{
				Name:      e.machineName,
				Namespace: testNamespace.Name,
			}
			err := e.Get(ctx, key, ym)
			return (err == nil && ym.Status.Ready == true)
		}, e.reconcileTimeout).Should(BeTrue())

		Expect(result.Requeue).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
		Expect(ym.Status.Addresses[0].Address).To(Equal(addr))
	})
})

var _ = Describe("YandexMachine deletions checks", func() {
	BeforeEach(func() {
		var err error

		e = ClusterTestEnv{
			Client:           k8sClient,
			clusterName:      "machine-delete-cluster",
			machineName:      "test",
			secretName:       "test",
			reconcileTimeout: 3 * time.Second,
		}
		//+kubebuilder:scaffold:webhook
		e.controller = gomock.NewController(GinkgoT())
		e.mockClient = mock_client.NewMockClient(e.controller)
		e.mockClientBuilder = mock_client.NewMockBuilder(e.controller)
		e.mockClientBuilder.EXPECT().GetDefaultClient(gomock.Any()).
			DoAndReturn(func(_ context.Context) (*mock_client.MockClient, error) {
				return e.mockClient, nil
			}).AnyTimes()
		e.mockClient.EXPECT().Close(gomock.Any()).Return(nil).AnyTimes()
		testNamespace, err = e.CreateNamespace(ctx, "machine-delete")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(e.DeleteNamespace(ctx)).To(Succeed())
		e.controller.Finish()
	})

	When("Deleting an ordinary YandexMachine", func() {
		It("should delete YandexMachine if ProviderID does not set", func() {
			ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
			controllerutil.AddFinalizer(ym, infrav1.MachineFinalizer)

			reconciler := &YandexMachineReconciler{
				Client:              e.Client,
				YandexClientBuilder: e.mockClientBuilder,
			}

			clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
				Client:        e.Client,
				Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
				YandexCluster: e.getYandexClusterWithOwnerReference(testNamespace.Name),
				Builder:       e.mockClientBuilder,
			})
			Expect(err).NotTo(HaveOccurred())

			machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
				Client:        e.Client,
				Machine:       e.getMachineWithInfrastructureRef(testNamespace.Name),
				LoadBalancer:  loadbalancer.New(clusterScope),
				ClusterGetter: clusterScope,
				YandexMachine: ym,
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := reconciler.reconcileDelete(ctx, machineScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())
		})
	})

	It("should delete YandexMachine if providerID is set, but YandexCloud VM does not exists", func() {
		const id string = "123"
		notFoundError := status.Error(codes.NotFound, "instance not found")

		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		controllerutil.AddFinalizer(ym, infrav1.MachineFinalizer)

		reconciler := &YandexMachineReconciler{
			Client:              e.Client,
			YandexClientBuilder: e.mockClientBuilder,
		}

		clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
			Client:        e.Client,
			Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
			YandexCluster: e.getYandexClusterWithOwnerReference(testNamespace.Name),
			Builder:       e.mockClientBuilder,
		})
		Expect(err).NotTo(HaveOccurred())

		machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
			Client:        e.Client,
			Machine:       e.getMachineWithInfrastructureRef(testNamespace.Name),
			LoadBalancer:  loadbalancer.New(clusterScope),
			ClusterGetter: clusterScope,
			YandexMachine: ym,
		})
		Expect(err).NotTo(HaveOccurred())
		machineScope.SetProviderID(id)

		// There is no YandexCloud VM with this id.
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), id).Return(nil, notFoundError)
		result, err := reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Requeue).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
	})

	It("should delete YandexMachine if providerID is set and YandexCloud VM does exists", func() {
		const id string = "123"

		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		controllerutil.AddFinalizer(ym, infrav1.MachineFinalizer)

		reconciler := &YandexMachineReconciler{
			Client:              e.Client,
			YandexClientBuilder: e.mockClientBuilder,
		}

		clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
			Client:        e.Client,
			Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
			YandexCluster: e.getYandexClusterWithOwnerReference(testNamespace.Name),
			Builder:       e.mockClientBuilder,
		})
		Expect(err).NotTo(HaveOccurred())

		machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
			Client:        e.Client,
			Machine:       e.getMachineWithInfrastructureRef(testNamespace.Name),
			LoadBalancer:  loadbalancer.New(clusterScope),
			ClusterGetter: clusterScope,
			YandexMachine: ym,
		})
		Expect(err).NotTo(HaveOccurred())
		machineScope.SetProviderID(id)

		// Send compute deletion request.
		e.setYandexMachineDeleteMocks(id)
		result, err := reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		status := machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusRunning))

		// Get DELETING status.
		result, err = reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		status = machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusDeleting))

		// YandexCloud VM deleted.
		result, err = reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(controllerutil.ContainsFinalizer(machineScope.YandexMachine, infrav1.ClusterFinalizer)).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
	})

	It("should not delete YandexMachine until receive NOT_FOUND return code from YC compute API", func() {
		const id string = "123"

		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		controllerutil.AddFinalizer(ym, infrav1.MachineFinalizer)

		reconciler := &YandexMachineReconciler{
			Client:              e.Client,
			YandexClientBuilder: e.mockClientBuilder,
		}

		clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
			Client:        e.Client,
			Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
			YandexCluster: e.getYandexClusterWithOwnerReference(testNamespace.Name),
			Builder:       e.mockClientBuilder,
		})
		Expect(err).NotTo(HaveOccurred())

		machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
			Client:        e.Client,
			Machine:       e.getMachineWithInfrastructureRef(testNamespace.Name),
			LoadBalancer:  loadbalancer.New(clusterScope),
			ClusterGetter: clusterScope,
			YandexMachine: ym,
		})
		Expect(err).NotTo(HaveOccurred())
		machineScope.SetProviderID(id)

		// Send compute deletion request.
		e.setYandexMachineDeleteErrorMocks(id)
		result, err := reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		status := machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusRunning))

		// Get DELETING status.
		result, err = reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		status = machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusDeleting))

		// When we are get error from YC API on instance request and error status
		// does not equal to NOT_FOUND error, we are should not delete the YandexMachine.
		// We should return with API error and restart reconciliation flow.
		result, err = reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).To(HaveOccurred())
		Expect(ym.GetFinalizers()).To(Equal([]string{infrav1.MachineFinalizer}))
		status = machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusDeleting))
	})

	It("should delete control plane YandexMachine and remove it from load balancer if YandexCloud VM does exists", func() {
		const (
			id      string = "123"
			address string = "1.2.3.4"
		)

		ym := e.getYandexMachineWithOwnerRef(testNamespace.Name)
		controllerutil.AddFinalizer(ym, infrav1.MachineFinalizer)
		ym.Status.Addresses = append(ym.Status.Addresses, corev1.NodeAddress{
			Address: address,
		})

		reconciler := &YandexMachineReconciler{
			Client:              e.Client,
			YandexClientBuilder: e.mockClientBuilder,
		}

		clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
			Client:        e.Client,
			Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
			YandexCluster: e.getYandexClusterWithOwnerReference(testNamespace.Name),
			Builder:       e.mockClientBuilder,
		})
		Expect(err).NotTo(HaveOccurred())

		// No API defaults here, so we have to set load balancer type
		clusterScope.YandexCluster.Spec.LoadBalancer.Type = infrav1.LoadBalancerTypeALB
		machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
			Client:        e.Client,
			Machine:       e.getCPMachineWithInfrastructureRef(testNamespace.Name),
			LoadBalancer:  loadbalancer.New(clusterScope),
			ClusterGetter: clusterScope,
			YandexMachine: ym,
		})
		Expect(err).NotTo(HaveOccurred())
		machineScope.SetProviderID(id)

		// Deregister from ALB and send compute deletion request.
		e.setCPYandexMachineDeleteMocks(id, address)
		result, err := reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		status := machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusRunning))

		// Get DELETING status.
		result, err = reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(RequeueDuration))
		status = machineScope.GetInstanceStatus()
		Expect(status).NotTo(BeNil())
		Expect(*status).To(Equal(infrav1.InstanceStatusDeleting))

		// YandexCloud VM deleted.
		result, err = reconciler.reconcileDelete(ctx, machineScope)
		Expect(err).NotTo(HaveOccurred())
		Expect(controllerutil.ContainsFinalizer(machineScope.YandexMachine, infrav1.ClusterFinalizer)).To(BeFalse())
		Expect(result.RequeueAfter).To(BeZero())
	})

})
