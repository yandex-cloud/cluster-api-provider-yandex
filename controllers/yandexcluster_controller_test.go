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
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/services/loadbalancers/builders"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/options"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("YandexCluster API check", func() {
	BeforeEach(func() {
		var err error

		e = ClusterTestEnv{
			Client:                   k8sClient,
			clusterName:              "test",
			hostControlPlaneEndpoint: "1.2.3.4",
			portControlPlaneEndpoint: 8443,
			eventuallyTimeout:        3 * time.Second,
			reconcileTimeout:         1 * time.Minute,
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
		testNamespace, err = e.CreateNamespace(ctx, "api-check")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(e.DeleteNamespace(ctx)).To(Succeed())
		e.controller.Finish()
	})

	When("Creating empty an YandexCluster", func() {
		It("should fail", func() {
			Expect(e.Create(ctx, e.getEmptyYandexCluster(testNamespace.Name))).NotTo(Succeed())
		})
	})

	When("Creating minimal an YandexCluster", func() {
		It("should not fail and set default values", func() {
			Expect(e.Create(ctx, e.getYandexCluster(testNamespace.Name))).To(Succeed())

			yc := &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return err == nil
			}, e.eventuallyTimeout).Should(BeTrue())

			Expect(yc.Spec.LoadBalancer.Type).To(Equal(infrav1.LoadBalancerTypeALB))
			Expect(yc.Spec.LoadBalancer.Listener.Internal).To(BeTrue())
			Expect(yc.Spec.LoadBalancer.BackendPort).To(BeNumerically("==", 8443))
			Expect(yc.Spec.LoadBalancer.Healthcheck.HealthcheckThreshold).To(BeNumerically("==", 3))
			Expect(yc.Spec.LoadBalancer.Healthcheck.HealthcheckTimeoutSec).To(BeNumerically("==", 1))
			Expect(yc.Spec.LoadBalancer.Healthcheck.HealthcheckIntervalSec).To(BeNumerically("==", 3))
		})
	})

})

var _ = Describe("YandexCluster reconciliation check", func() {
	BeforeEach(func() {
		var err error

		e = ClusterTestEnv{
			Client:                   k8sClient,
			clusterName:              "test",
			hostControlPlaneEndpoint: "1.2.3.4",
			portControlPlaneEndpoint: 8443,
			eventuallyTimeout:        3 * time.Second,
			reconcileTimeout:         1 * time.Minute,
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
		testNamespace, err = e.CreateNamespace(ctx, "reconcile-check")
		Expect(err).ToNot(HaveOccurred())

		config = options.Config{
			ReconcileTimeout: e.reconcileTimeout,
		}
	})

	AfterEach(func() {
		Expect(e.DeleteNamespace(ctx)).To(Succeed())
		e.controller.Finish()
	})

	When("Creating YandexCluster", func() {
		It("should not reconcile an YandexCluster without parent CAPI Cluster", func() {
			yc := e.getYandexCluster(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client: k8sClient,
				Config: config,
			}
			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: yc.Namespace,
					Name:      yc.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should not reconcile an YandexCluster with paused CAPI Cluster", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			cc.Spec.Paused = true
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client: k8sClient,
				Config: config,
			}

			result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(yc.Namespace, yc.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should add finalizer to the YandexCluster and requeue", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}

			result, err := reconciler.Reconcile(ctx, e.getReconcileRequest(yc.Namespace, yc.Name))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && len(yc.GetFinalizers()) > 0)
			}, e.eventuallyTimeout).Should(BeTrue())
		})

		It("should create load balancer if it does not exists and set ready status", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}

			// reconciler sets finalizer here.
			req := e.getReconcileRequest(yc.Namespace, yc.Name)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			ip := "1.2.3.4"
			e.setNewALBReconcileMocks(ip)
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && yc.Status.Ready)
			}, e.eventuallyTimeout).Should(BeTrue())

			Expect(yc.Spec.ControlPlaneEndpoint.Host).To(Equal(ip))
			Expect(yc.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(8443)))
			Expect(yc.Status.LoadBalancer.ListenerAddress).To(Equal(ip))
			Expect(yc.Status.LoadBalancer.ListenerPort).To(Equal(int32(8443)))
		})

		It("should create load balancer with correct listener address if it does not exists and set ready status", func() {
			ip := "1.2.3.4"

			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			yc.Spec.LoadBalancer.Listener.Address = ip
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}

			// reconciler sets finalizer here.
			req := e.getReconcileRequest(yc.Namespace, yc.Name)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			e.setNewALBReconcileMocks("")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && yc.Status.Ready)
			}, e.eventuallyTimeout).Should(BeTrue())

			Expect(yc.Spec.ControlPlaneEndpoint.Host).To(Equal(ip))
			Expect(yc.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(8443)))
			Expect(yc.Status.LoadBalancer.ListenerAddress).To(Equal(ip))
			Expect(yc.Status.LoadBalancer.ListenerPort).To(Equal(int32(8443)))
		})

		It("should set ready status and controlplane endpoint when load balancer exists and listener spec empty", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}
			req := e.getReconcileRequest(yc.Namespace, yc.Name)

			// reconciler sets finalizer here.
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			ip := "1.2.3.4"
			e.setExistingALBReconcileMocks(ip)
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && yc.Status.Ready)
			}, e.eventuallyTimeout).Should(BeTrue())

			Expect(yc.Spec.ControlPlaneEndpoint.Host).To(Equal(ip))
			Expect(yc.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(8443)))
			Expect(yc.Status.LoadBalancer.ListenerAddress).To(Equal(ip))
			Expect(yc.Status.LoadBalancer.ListenerPort).To(Equal(int32(8443)))
		})

		It("should error when load balancer exists and listener spec not empty and differ from existed load balancer", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			yc.Spec.LoadBalancer.Listener.Address = "1.1.1.1"
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}
			req := e.getReconcileRequest(yc.Namespace, yc.Name)

			// reconciler sets finalizer here.
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			existLoadBalancerIP := "1.2.3.4"
			mockID, mockName := "123", "alb"

			gomock.InOrder(
				e.mockClient.EXPECT().
					ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, id, name string) (*alb.TargetGroup, error) {
						targetGroup := &alb.TargetGroup{
							Id:   mockID,
							Name: mockName,
						}
						logFunctionCalls(
							"ALBTargetGroupGetByName",
							map[string]interface{}{"id": id, "name": name},
							[]interface{}{targetGroup, nil})
						return targetGroup, nil
					}),
				e.mockClient.EXPECT().ALBBackendGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, id, name string) (*alb.BackendGroup, error) {
						backendGroup := &alb.BackendGroup{
							Id:   mockID,
							Name: mockName,
						}
						logFunctionCalls(
							"ALBBackendGroupGetByName",
							map[string]interface{}{"id": id, "name": name},
							[]interface{}{backendGroup, nil})
						return backendGroup, nil
					}),
				e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
						loadBalancer := &alb.LoadBalancer{
							Id:     mockID,
							Name:   mockName,
							Status: alb.LoadBalancer_ACTIVE,
							Listeners: []*alb.Listener{
								{
									Endpoints: []*alb.Endpoint{
										{
											Ports: []int64{8443},
											Addresses: []*alb.Address{
												{
													Address: &alb.Address_InternalIpv4Address{
														InternalIpv4Address: &alb.InternalIpv4Address{
															Address:  existLoadBalancerIP,
															SubnetId: id,
														},
													},
												},
											},
										},
									},
								},
							},
						}
						logFunctionCalls(
							"ALBGetByName",
							map[string]interface{}{"id": id, "name": name},
							[]interface{}{loadBalancer, nil})
						return loadBalancer, nil
					}),
			)

			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should error if controlplaneendpoint is set and load balancer listener address doesn't exist", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			controlPlaneEndpointIP := "1.1.1.1"
			yc.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
				Host: controlPlaneEndpointIP,
				Port: 8443,
			}
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}
			req := e.getReconcileRequest(yc.Namespace, yc.Name)

			// reconciler sets finalizer here.
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// mocks
			mockID := "123"
			gomock.InOrder(
				e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, id, name string) (*alb.TargetGroup, error) {
						logFunctionCalls(
							"ALBTargetGroupGetByName",
							map[string]interface{}{"id": id, "name": name},
							[]interface{}{nil, nil})
						return nil, nil
					}),
				e.mockClient.EXPECT().ALBTargetGroupCreate(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, targetGroup *alb.CreateTargetGroupRequest) (string, error) {
						logFunctionCalls(
							"ALBTargetGroupCreate",
							map[string]interface{}{"targetGroup": targetGroup},
							[]interface{}{mockID, nil})
						return mockID, nil
					}),
				e.mockClient.EXPECT().ALBBackendGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, id, name string) (*alb.BackendGroup, error) {
						logFunctionCalls(
							"ALBBackendGroupGetByName",
							map[string]interface{}{"id": id, "name": name},
							[]interface{}{nil, nil})
						return nil, nil
					}),
				e.mockClient.EXPECT().ALBBackendGroupCreate(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, backendGroup *alb.CreateBackendGroupRequest) (string, error) {
						logFunctionCalls(
							"ALBBackendGroupCreate",
							map[string]interface{}{"backendGroup": backendGroup},
							[]interface{}{mockID, nil})
						return mockID, nil
					}),
				e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
						logFunctionCalls(
							"ALBGetByName",
							map[string]interface{}{"id": id, "name": name},
							[]interface{}{nil, nil})
						return nil, nil
					}),
			)

			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should not change controlplaneendpoint when it exist and load balancer listener address exists", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			controlPlaneEndpointIP := "1.2.3.4"
			ip := "1.2.3.4"
			yc.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
				Host: controlPlaneEndpointIP,
				Port: 8443,
			}
			yc.Spec.LoadBalancer.Listener.Address = ip
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}
			req := e.getReconcileRequest(yc.Namespace, yc.Name)

			// reconciler sets finalizer here.
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// mock should return ip address from cluster load balancer spec
			e.setNewALBReconcileMocks("")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && yc.Status.Ready)
			}, e.eventuallyTimeout).Should(BeTrue())

			Expect(yc.Spec.ControlPlaneEndpoint.Host).To(Equal(controlPlaneEndpointIP))
			Expect(yc.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(8443)))
			Expect(yc.Status.LoadBalancer.ListenerAddress).To(Equal(ip))
			Expect(yc.Status.LoadBalancer.ListenerPort).To(Equal(int32(8443)))
		})

	})

	When("Creating ALB", func() {
		It("should error and retry to create ALB target group on api error", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}

			// reconciler sets finalizer here.
			req := e.getReconcileRequest(yc.Namespace, yc.Name)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			e.setNewTargetGroupErrorMocks()
			// got a target group get error.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			// got a target group create error.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			// got a backend group get error.
			// We call mock with an error only to exit from the Reconcile().
			// This allows us not to prepare mock calls for the full cycle of creating ALB
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should error and retry to create ALB backend group on api error", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}

			// reconciler sets finalizer here.
			req := e.getReconcileRequest(yc.Namespace, yc.Name)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			e.setNewBackendGroupErrorMocks()
			// got a backend group get error.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			// got a backend group create error.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			// got an alb get error.
			// We call mock with an error only to exit from the Reconcile().
			// This allows us not to prepare mock calls for the full cycle of creating ALB
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should error and retry to create ALB on api error", func() {
			cc := e.getCAPIClusterWithInfrastructureReference(testNamespace.Name)
			Expect(e.Create(ctx, cc)).To(Succeed())
			yc := e.getYandexClusterWithOwnerReference(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client:              k8sClient,
				YandexClientBuilder: e.mockClientBuilder,
				Config:              config,
			}

			// reconciler sets finalizer here.
			req := e.getReconcileRequest(yc.Namespace, yc.Name)
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			ip := "1.2.3.4"
			e.setNewALBErrorMocks(ip)
			// got an ALB get error.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			// got an ALB create error.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			// ALB created.
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && yc.Status.Ready)
			}, e.eventuallyTimeout).Should(BeTrue())

			Expect(yc.Spec.ControlPlaneEndpoint.Host).To(Equal(ip))
			Expect(yc.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(8443)))
			Expect(yc.Status.LoadBalancer.ListenerAddress).To(Equal(ip))
			Expect(yc.Status.LoadBalancer.ListenerPort).To(Equal(int32(8443)))
		})
	})
})

var _ = Describe("YandexCluster deletion check", func() {
	BeforeEach(func() {
		var err error
		e = ClusterTestEnv{
			Client:                   k8sClient,
			clusterName:              "test",
			hostControlPlaneEndpoint: "1.2.3.4",
			portControlPlaneEndpoint: 8443,
			eventuallyTimeout:        3 * time.Second,
			reconcileTimeout:         1 * time.Minute,
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
		testNamespace, err = e.CreateNamespace(ctx, "deletion-check")
		Expect(err).ToNot(HaveOccurred())

		config = options.Config{
			ReconcileTimeout: e.reconcileTimeout,
		}
	})

	AfterEach(func() {
		Expect(e.DeleteNamespace(ctx)).To(Succeed())
		e.controller.Finish()
	})

	When("Deleting YandexCluster", func() {
		It("should not delete an YandexCluster without finalizer", func() {
			yc := e.getYandexCluster(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client: k8sClient,
				Config: config,
			}

			clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
				Client:        e.Client,
				Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
				YandexCluster: yc,
				Builder:       e.mockClientBuilder,
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := reconciler.reconcileDelete(ctx, clusterScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(clusterScope.Close(ctx)).Error().NotTo(HaveOccurred())

			// Check that the YandexCluster still exists after reconciliation.
			yc = &infrav1.YandexCluster{}
			Eventually(func() bool {
				key := client.ObjectKey{
					Name:      e.clusterName,
					Namespace: testNamespace.Name,
				}
				err := e.Get(ctx, key, yc)
				return (err == nil && yc.Name == e.clusterName)
			}, e.eventuallyTimeout).Should(BeTrue())
		})

		It("should delete an YandexCluster when load balancer in YandexCloud does not exists", func() {
			yc := e.getYandexCluster(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client: k8sClient,
				Config: config,
			}

			clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
				Client:        e.Client,
				Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
				YandexCluster: yc,
				Builder:       e.mockClientBuilder,
			})
			Expect(err).NotTo(HaveOccurred())
			controllerutil.AddFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer)

			// No resources exists in YandexCloud.
			e.setNonExistingALBDeleteMocks("test-api")
			result, err := reconciler.reconcileDelete(ctx, clusterScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutil.ContainsFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer)).To(BeFalse())
			Expect(result.Requeue).To(BeFalse())
			Expect(clusterScope.Close(ctx)).Error().NotTo(HaveOccurred())
		})

		It("should delete an YandexCluster and remove load balancer from YandexCloud, if it exists", func() {
			yc := e.getYandexCluster(testNamespace.Name)
			Expect(e.Create(ctx, yc)).To(Succeed())

			reconciler := &YandexClusterReconciler{
				Client: k8sClient,
				Config: config,
			}

			clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
				Client:        e.Client,
				Cluster:       e.getCAPIClusterWithInfrastructureReference(testNamespace.Name),
				YandexCluster: yc,
				Builder:       e.mockClientBuilder,
			})
			Expect(err).NotTo(HaveOccurred())

			controllerutil.AddFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer)
			lbName := clusterScope.GetLBName()
			lbID := "123"

			e.setExistingALBDeleteMocks(lbName, lbID)
			result, err := reconciler.reconcileDelete(ctx, clusterScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(RequeueDuration))

			// Backend group deletion.
			bgID := lbID
			bgName := builders.NewALBBackendGroupBuilder(clusterScope.GetLBSpec()).
				WithLBName(clusterScope.GetLBName()).
				GetName()
			e.setExistingBackendGroupDeleteMocks(lbName, bgName, bgID)

			result, err = reconciler.reconcileDelete(ctx, clusterScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(RequeueDuration))

			// Target group deletion.
			tgID := lbID
			tgName := builders.NewALBTargetGroupBuilder(clusterScope.GetLBSpec()).
				WithLBName(clusterScope.GetLBName()).
				GetName()

			e.setExistingTargetGroupDeleteMocks(lbName, tgName, tgID)
			result, err = reconciler.reconcileDelete(ctx, clusterScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(RequeueDuration))

			// No load balancer resources exists in YandexCloud at this moment.
			// We have to finish delete reconciliation and remove finalizer from YandexCLuster.
			e.setNonExistingALBDeleteMocks(lbName)
			result, err = reconciler.reconcileDelete(ctx, clusterScope)
			Expect(err).NotTo(HaveOccurred())
			Expect(controllerutil.ContainsFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer)).To(BeFalse())
			Expect(result.Requeue).To(BeFalse())
			Expect(clusterScope.Close(ctx)).Error().NotTo(HaveOccurred())
		})
	})
})
