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
	"fmt"
	"time"

	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client/mock_client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/options"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	testNamespace *corev1.Namespace
	e             ClusterTestEnv
	config        options.Config
)

// YandexCluster test utils.
type ClusterTestEnv struct {
	client.Client
	controller               *gomock.Controller
	mockClient               *mock_client.MockClient
	clusterName              string
	machineName              string
	secretName               string
	hostControlPlaneEndpoint string
	portControlPlaneEndpoint int32
	eventuallyTimeout        time.Duration
	reconcileTimeout         time.Duration
}

// CreateNamespace creates the random namespace from prefix.
func (c *ClusterTestEnv) CreateNamespace(ctx context.Context, prefix string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", prefix),
		},
	}
	if err := c.Client.Create(ctx, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

// DeleteNamespace deletes the temporary namespace.
func (c *ClusterTestEnv) DeleteNamespace(ctx context.Context) error {
	return c.Delete(ctx, testNamespace)
}

// YandexCluster controller testsute helper fuctions.

// getEmptyYandexCluster returns an empty Yandex Cluster specification.
func (c *ClusterTestEnv) getEmptyYandexCluster(nsName string) *infrav1.YandexCluster {
	return &infrav1.YandexCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.clusterName,
			Namespace: nsName,
		},
		Spec: infrav1.YandexClusterSpec{},
	}
}

// getYandexCluster returns a minimal Yandex Cluster specification.
func (c *ClusterTestEnv) getYandexCluster(nsName string) *infrav1.YandexCluster {
	return &infrav1.YandexCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.clusterName,
			Namespace: nsName,
		},
		Spec: infrav1.YandexClusterSpec{
			FolderID: "987654321",
			NetworkSpec: infrav1.NetworkSpec{
				ID: "1122334455",
			},
			LoadBalancer: infrav1.LoadBalancerSpec{
				Listener: infrav1.ListenerSpec{
					Subnet: infrav1.SubnetSpec{
						ZoneID: "ru-central-1a",
						ID:     "123456789",
					},
				},
			},
		},
	}
}

// newCAPIClusterWithInfrastructureReference return CAPI CLuster with infrastruture reference.
func (c *ClusterTestEnv) getCAPIClusterWithInfrastructureReference(nsName string) *clusterv1.Cluster {
	return &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.clusterName,
			Namespace: nsName,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				Namespace:  nsName,
				Name:       c.clusterName,
				APIVersion: infrav1.GroupVersion.String(),
				Kind:       "YandexCluster",
			},
		},
	}
}

// newYandexClusterWithOwnerReference returns Yandex Cluster with owner reference.
func (c *ClusterTestEnv) getYandexClusterWithOwnerReference(nsName string) *infrav1.YandexCluster {
	yc := c.getYandexCluster(nsName)
	yc.ObjectMeta = metav1.ObjectMeta{
		Name:      c.clusterName,
		Namespace: nsName,
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       "Cluster",
				Name:       c.clusterName,
				UID:        "some-uid",
			},
		},
	}
	return yc
}

// setNewALBMocks mocks YandexCloud client API calls on new load balancer reconciliation.
func (c *ClusterTestEnv) setNewALBReconcileMocks(address string) {
	const (
		mockID   string = "123"
		mockName string = "alb"
	)
	albAddress := address
	gomock.InOrder(
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name string, zone string) (*alb.TargetGroup, error) {
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *alb.CreateTargetGroupRequest) (string, error) {
				logFunctionCalls(
					"ALBTargetGroupCreate",
					map[string]interface{}{"req": req},
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
			DoAndReturn(func(_ context.Context, req *alb.CreateBackendGroupRequest) (string, error) {
				logFunctionCalls(
					"ALBBackendGroupCreate",
					map[string]interface{}{"req": req},
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
		e.mockClient.EXPECT().ALBCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, albRequest *alb.CreateLoadBalancerRequest) (string, error) {
				if address == "" {
					albAddress = albRequest.ListenerSpecs[0].EndpointSpecs[0].
						AddressSpecs[0].GetInternalIpv4AddressSpec().GetAddress()
				}
				logFunctionCalls(
					"ALBCreate",
					map[string]interface{}{"albRequest": albRequest},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				loadBalancer := &alb.LoadBalancer{
					Id:     mockID,
					Name:   mockName,
					Status: alb.LoadBalancer_ACTIVE,
				}
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{loadBalancer, nil})
				return loadBalancer, nil
			}),
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id string, name string) (*alb.LoadBalancer, error) {
				loadBalancer := &alb.LoadBalancer{
					Id:     id,
					Name:   name,
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
													Address:  albAddress,
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
}

// setExistingALBMock mocks YandexCloud client API calls on existing load balancer reconciliation.
func (c *ClusterTestEnv) setExistingALBReconcileMocks(address string) {
	const (
		mockID   string = "123"
		mockName string = "alb"
	)
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
													Address:  address,
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
			}).Times(3),
	)
}

// setExistingALBDeleteMocks mocks YandexCloud client API calls on existing application load balancer deletion.
func (c *ClusterTestEnv) setExistingALBDeleteMocks(mockName, mockID string) {
	gomock.InOrder(
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), mockName).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				loadBalancer := &alb.LoadBalancer{
					Id:     mockID,
					Name:   mockName,
					Status: alb.LoadBalancer_ACTIVE,
				}
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{loadBalancer, nil})
				return loadBalancer, nil
			}),
		e.mockClient.EXPECT().ALBDelete(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) error {
				logFunctionCalls(
					"ALBDelete",
					map[string]interface{}{"id": id},
					[]interface{}{nil})
				return nil
			}),
	)
}

// setNonExistingALBDeleteMocks mocks YandexCloud client API calls on non existing application load balancer deletion.
func (c *ClusterTestEnv) setNonExistingALBDeleteMocks(name string) {
	gomock.InOrder(
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), name).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBBackendGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.BackendGroup, error) {
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.TargetGroup, error) {
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
	)
}

// setExistingBackendGroupDeleteMocks mocks YandexCloud client API calls on existing ALB backend group deletion.
func (c *ClusterTestEnv) setExistingBackendGroupDeleteMocks(lbName, bgName, mockID string) {
	gomock.InOrder(
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), lbName).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBBackendGroupGetByName(gomock.Any(), gomock.Any(), bgName).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.BackendGroup, error) {
				backendGroup := &alb.BackendGroup{
					Id:   mockID,
					Name: bgName,
				}
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{backendGroup, nil})
				return backendGroup, nil
			}),
		e.mockClient.EXPECT().ALBBackendGroupDelete(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) error {
				logFunctionCalls(
					"ALBBackendGroupDelete",
					map[string]interface{}{"id": id},
					[]interface{}{nil})
				return nil
			}),
	)
}

// setExistingBackendGroupDeleteMocks mocks YandexCloud client API calls on existing ALB target group deletion.
func (c *ClusterTestEnv) setExistingTargetGroupDeleteMocks(lbName, tgName, mockID string) {
	gomock.InOrder(
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), lbName).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBBackendGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.BackendGroup, error) {
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), tgName).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.TargetGroup, error) {
				targetGroup := &alb.TargetGroup{
					Id:   mockID,
					Name: tgName,
				}
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{targetGroup, nil})
				return targetGroup, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupDelete(gomock.Any(), mockID).DoAndReturn(
			func(_ context.Context, id string) error {
				logFunctionCalls(
					"ALBTargetGroupDelete",
					map[string]interface{}{"id": id},
					[]interface{}{nil})
				return nil
			}),
	)
}

// setNewTargetGroupErrorMocks mocks YandexCloud client API calls on ALB target group creation with API errors.
func (c *ClusterTestEnv) setNewTargetGroupErrorMocks() {
	const mockID string = "123"

	gomock.InOrder(
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.TargetGroup, error) {
				err := fmt.Errorf("target group get error")
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, err})
				return nil, err
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.TargetGroup, error) {
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, targetGroup *alb.CreateTargetGroupRequest) (*alb.TargetGroup, error) {
				err := fmt.Errorf("target group create error")
				logFunctionCalls(
					"ALBTargetGroupCreate",
					map[string]interface{}{"targetGroup": targetGroup},
					[]interface{}{nil, err})
				return nil, err
			}),
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
				err := fmt.Errorf("backend group get error")
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, err})
				return nil, err
			}),
	)
}

// setNewBackendGroupErrorMocks mocks YandexCloud client API calls on ALB backend group creation with API errors.
func (c *ClusterTestEnv) setNewBackendGroupErrorMocks() {
	const (
		mockID   string = "123"
		mockName string = "alb"
	)
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
				err := fmt.Errorf("backend group get error")
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, err})
				return nil, err
			}),
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
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBBackendGroupCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *alb.CreateBackendGroupRequest) (string, error) {
				err := fmt.Errorf("backend group create error")
				logFunctionCalls(
					"ALBBackendGroupCreate",
					map[string]interface{}{"req": req},
					[]interface{}{"", err})
				return "", err
			}),
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
				logFunctionCalls(
					"ALBBackendGroupGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBBackendGroupCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *alb.CreateBackendGroupRequest) (string, error) {
				logFunctionCalls(
					"ALBBackendGroupCreate",
					map[string]interface{}{"req": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				err := fmt.Errorf("alb get error")
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, err})
				return nil, err
			}),
	)
}

// setNewALBErrorMocks mocks YandexCloud client API calls on ALB creation with API errors.
func (c *ClusterTestEnv) setNewALBErrorMocks(address string) {
	const (
		mockID   string = "123"
		mockName string = "alb"
	)
	albAddress := address
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
				err := fmt.Errorf("alb get error")
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, err})
				return nil, err
			}),

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
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, alb *alb.CreateLoadBalancerRequest) (string, error) {
				err := fmt.Errorf("alb create error")
				logFunctionCalls(
					"ALBCreate",
					map[string]interface{}{"alb": alb},
					[]interface{}{"", err})
				return "", err
			}),

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
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{nil, nil})
				return nil, nil
			}),
		e.mockClient.EXPECT().ALBCreate(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, albRequest *alb.CreateLoadBalancerRequest) (string, error) {
				if address == "" {
					albAddress = albRequest.ListenerSpecs[0].EndpointSpecs[0].
						AddressSpecs[0].GetInternalIpv4AddressSpec().GetAddress()
				}
				logFunctionCalls(
					"ALBCreate",
					map[string]interface{}{"albRequest": albRequest},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				loadBalancer := &alb.LoadBalancer{
					Id:     mockID,
					Name:   mockName,
					Status: alb.LoadBalancer_ACTIVE,
				}
				logFunctionCalls(
					"ALBGetByName",
					map[string]interface{}{"id": id, "name": name},
					[]interface{}{loadBalancer, nil})
				return loadBalancer, nil
			}),
		e.mockClient.EXPECT().ALBGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id, name string) (*alb.LoadBalancer, error) {
				loadBalancer := &alb.LoadBalancer{
					Id:     id,
					Name:   name,
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
													Address:  albAddress,
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
			}))
}

// YandexMachine controller testsute helper fuctions.

// getYandexMachine returns minimal YandexMachine specification.
func (c *ClusterTestEnv) getYandexMachine(nsName string) *infrav1.YandexMachine {
	return &infrav1.YandexMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.machineName,
			Namespace: nsName,
		},
		Spec: infrav1.YandexMachineSpec{
			Resources: infrav1.Resources{
				Memory: resource.MustParse("1Gi"),
				Cores:  1,
			},
			BootDisk: &infrav1.Disk{
				Size:    resource.MustParse("100Gi"),
				ImageID: "imageid",
			},
			NetworkInterfaces: []infrav1.NetworkInterface{{
				SubnetID: "subnetid",
			}},
		},
	}
}

// getYandexMachineWithOwnerRef returns YandexMachine specification with CAPI Machine owner reference.
func (c *ClusterTestEnv) getYandexMachineWithOwnerRef(nsName string) *infrav1.YandexMachine {
	ym := c.getYandexMachine(nsName)
	ym.ObjectMeta.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       "Machine",
		Name:       c.machineName,
		UID:        types.UID("uid"),
	}}

	return ym
}

// getMachine returns CAPI Machine specification.
func (c *ClusterTestEnv) getMachine(nsName string) *clusterv1.Machine {
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: c.clusterName,
			},
			Name:      c.machineName,
			Namespace: nsName,
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: c.clusterName,
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: ptr.To(c.secretName),
			},
		},
	}
}

// getMachineWithInfrastructureRef returns CAPI Machine with infrastructure reference.
func (c *ClusterTestEnv) getMachineWithInfrastructureRef(nsName string) *clusterv1.Machine {
	cm := c.getMachine(nsName)
	cm.Spec.InfrastructureRef = corev1.ObjectReference{
		Kind:       "YandexMachine",
		Namespace:  nsName,
		Name:       c.machineName,
		APIVersion: infrav1.GroupVersion.String(),
	}
	cm.Spec.Bootstrap = clusterv1.Bootstrap{
		DataSecretName: ptr.To(c.secretName),
	}

	return cm
}

// getCPMachineWithInfrastructureRef returns controlplane CAPI Machine with infrastructure reference.
func (c *ClusterTestEnv) getCPMachineWithInfrastructureRef(nsName string) *clusterv1.Machine {
	cm := c.getMachineWithInfrastructureRef(nsName)
	cm.ObjectMeta.Labels[clusterv1.MachineControlPlaneLabel] = ""

	return cm
}

// getBootstrapSecret returns mock secret for YandexMachine bootstrap.
func (c *ClusterTestEnv) getBootstrapSecret(nsName string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.secretName,
			Namespace: nsName,
		},
		Data: map[string][]byte{
			"value": []byte("verysecretdata"),
		},
	}
}

// getReconcileRequest return runtime controller reconciliation request for a resource name.
func (c *ClusterTestEnv) getReconcileRequest(namespace, name string) ctrl.Request {
	return ctrl.Request{NamespacedName: client.ObjectKey{Namespace: namespace, Name: name}}
}

// setNewYandexMachineReconcileMocks mocks the YandexClient API calls on YandexMachine reconciliation.
func (c *ClusterTestEnv) setNewYandexMachineReconcileMocks(address string) {
	const mockID string = "123"

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_STARTING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: address,
							},
						},
					},
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
	)
}

// setNewYandexMachineErrorReconcileMocks mocks the YandexClient API calls on YandexMachine reconciliation with API errors.
func (c *ClusterTestEnv) setNewYandexMachineErrorReconcileMocks() {
	const mockID string = "123"

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				err := fmt.Errorf("compute creation error")
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{"", err})
				return "", err
			}),
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_STARTING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
	)
}

// setYandexMachineNotFoundReconcileMocks mocks the YandexClient API calls on YandexMachine reconciliation with NotFound API error.
func (c *ClusterTestEnv) setYandexMachineNotFoundReconcileMocks() {
	const mockID string = "123"
	notFoundError := status.Error(codes.NotFound, "instance not found")

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{"", notFoundError})
				return nil, notFoundError
			}),
	)
}

// setNewCPYandexMachineReconcileMock mocks the YandexClient API calls on YandexMachine with controlplane role reconciliation.
func (c *ClusterTestEnv) setNewCPYandexMachineReconcileMocks(address, targetGroup string) {
	const mockID string = "123"

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: address,
							},
						},
					},
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name, zone string) (*alb.TargetGroup, error) {
				tg := &alb.TargetGroup{
					Id:   mockID,
					Name: targetGroup,
				}
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{tg, nil})
				return tg, nil
			}),
		e.mockClient.EXPECT().ALBAddTarget(gomock.Any(), &alb.AddTargetsRequest{
			TargetGroupId: mockID,
			Targets: []*alb.Target{
				{
					SubnetId: "subnetid",
					AddressType: &alb.Target_IpAddress{
						IpAddress: address,
					},
				},
			},
		}).DoAndReturn(func(_ context.Context, req *alb.AddTargetsRequest) (*operation.Operation, error) {
			logFunctionCalls(
				"ALBAddTarget",
				map[string]interface{}{"request": req},
				[]interface{}{&operation.Operation{}, nil})
			return &operation.Operation{}, nil
		}),
	)
}

// setNewWithoutTargetGroupCPYandexMachineReconcileMocks mocks the YandexClient API calls on YandexMachine with controlplane role reconciliation without having Target Group.
func (c *ClusterTestEnv) setNewWithoutTargetGroupCPYandexMachineReconcileMocks(address string) {
	const mockID string = "123"

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: address,
							},
						},
					},
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name, zone string) (*alb.TargetGroup, error) {
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{nil, nil})
				return nil, nil
			}),
	)
}

// setNewCPYandexMachineErrorReconcileMock mocks the YandexClient API calls on controlplane role reconciliation with API error.
func (c *ClusterTestEnv) setNewCPYandexMachineErrorReconcileMocks(address, targetGroup string) {
	const mockID string = "123"

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, req *compute.CreateInstanceRequest) (string, error) {
				logFunctionCalls(
					"ComputeCreate",
					map[string]interface{}{"request": req},
					[]interface{}{mockID, nil})
				return mockID, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: address,
							},
						},
					},
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name, zone string) (*alb.TargetGroup, error) {
				tg := &alb.TargetGroup{
					Id:   mockID,
					Name: targetGroup,
				}
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{tg, nil})
				return tg, nil
			}),
		e.mockClient.EXPECT().ALBAddTarget(gomock.Any(), &alb.AddTargetsRequest{
			TargetGroupId: mockID,
			Targets: []*alb.Target{
				{
					SubnetId: "subnetid",
					AddressType: &alb.Target_IpAddress{
						IpAddress: address,
					},
				},
			},
		}).DoAndReturn(func(_ context.Context, req *alb.AddTargetsRequest) (*operation.Operation, error) {
			err := fmt.Errorf("alb target group add error")
			logFunctionCalls(
				"ALBAddTarget",
				map[string]interface{}{"request": req},
				[]interface{}{&operation.Operation{}, err})
			return &operation.Operation{}, err
		}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: address,
							},
						},
					},
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name, zone string) (*alb.TargetGroup, error) {
				tg := &alb.TargetGroup{
					Id:   mockID,
					Name: targetGroup,
				}
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{tg, nil})
				return tg, nil
			}),
		e.mockClient.EXPECT().ALBAddTarget(gomock.Any(), &alb.AddTargetsRequest{
			TargetGroupId: mockID,
			Targets: []*alb.Target{
				{
					SubnetId: "subnetid",
					AddressType: &alb.Target_IpAddress{
						IpAddress: address,
					},
				},
			},
		}).DoAndReturn(func(_ context.Context, req *alb.AddTargetsRequest) (*operation.Operation, error) {
			logFunctionCalls(
				"ALBAddTarget",
				map[string]interface{}{"request": req},
				[]interface{}{&operation.Operation{}, nil})
			return &operation.Operation{}, nil
		}),
	)
}

// setYandexMachineDeleteMocks mocks the YandexClient API calls on YandexMachine delete.
func (c *ClusterTestEnv) setYandexMachineDeleteMocks(mockID string) {
	notFoundError := status.Error(codes.NotFound, "instance not found")

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ComputeDelete(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) error {
				logFunctionCalls(
					"ComputeDelete",
					map[string]interface{}{"id": id},
					[]interface{}{nil})
				return nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_DELETING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{nil, notFoundError})
				return nil, notFoundError
			}),
	)
}

// setYandexMachineDeleteErrorMocks mocks the YandexClient API calls on YandexMachine delete with API error.
func (c *ClusterTestEnv) setYandexMachineDeleteErrorMocks(mockID string) {
	unavailableError := status.Error(codes.Unavailable, "instance unavailable")

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ComputeDelete(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) error {
				logFunctionCalls(
					"ComputeDelete",
					map[string]interface{}{"id": id},
					[]interface{}{nil})
				return nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_DELETING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{nil, unavailableError})
				return nil, unavailableError
			}),
	)
}

// setCPYandexMachineDeleteMocks mocks the YandexClient API calls on control plane YandexMachine delete.
func (c *ClusterTestEnv) setCPYandexMachineDeleteMocks(mockID, mockAddress string) {
	notFoundError := status.Error(codes.NotFound, "instance not found")

	gomock.InOrder(
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: mockAddress,
							},
						},
					},
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().
			ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name, zone string) (*alb.TargetGroup, error) {
				tg := &alb.TargetGroup{
					Id:   mockID,
					Name: "targetgroup",
					Targets: []*alb.Target{{
						SubnetId: "subnetid",
						AddressType: &alb.Target_IpAddress{
							IpAddress: mockAddress,
						},
					}},
				}
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{tg, nil})
				return tg, nil
			}),
		e.mockClient.EXPECT().ALBRemoveTarget(gomock.Any(), &alb.RemoveTargetsRequest{
			TargetGroupId: mockID,
			Targets: []*alb.Target{
				{
					SubnetId: "subnetid",
					AddressType: &alb.Target_IpAddress{
						IpAddress: mockAddress,
					},
				},
			},
		}).DoAndReturn(func(_ context.Context, req *alb.RemoveTargetsRequest) (*operation.Operation, error) {
			logFunctionCalls(
				"ALBRemoveTarget",
				map[string]interface{}{"request": req},
				[]interface{}{&operation.Operation{}, nil})
			return &operation.Operation{}, nil
		}),
		e.mockClient.EXPECT().ComputeDelete(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) error {
				logFunctionCalls(
					"ComputeDelete",
					map[string]interface{}{"id": id},
					[]interface{}{nil})
				return nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				instance := &compute.Instance{
					Name:   c.machineName,
					Id:     mockID,
					Status: compute.Instance_DELETING,
				}
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{instance, nil})
				return instance, nil
			}),
		e.mockClient.EXPECT().ComputeGet(gomock.Any(), mockID).
			DoAndReturn(func(_ context.Context, id string) (*compute.Instance, error) {
				logFunctionCalls(
					"ComputeGet",
					map[string]interface{}{"id": id},
					[]interface{}{nil, notFoundError})
				return nil, notFoundError
			}),
		e.mockClient.EXPECT().
			ALBTargetGroupGetByName(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, name, zone string) (*alb.TargetGroup, error) {
				tg := &alb.TargetGroup{
					Id:   mockID,
					Name: "targetgroup",
					Targets: []*alb.Target{{
						SubnetId: "subnetid",
						AddressType: &alb.Target_IpAddress{
							IpAddress: mockAddress,
						},
					}},
				}
				logFunctionCalls(
					"ALBTargetGroupGetByName",
					map[string]interface{}{"name": name, "zone": zone},
					[]interface{}{tg, nil})
				return tg, nil
			}),
		e.mockClient.EXPECT().ALBRemoveTarget(gomock.Any(), &alb.RemoveTargetsRequest{
			TargetGroupId: mockID,
			Targets: []*alb.Target{
				{
					SubnetId: "subnetid",
					AddressType: &alb.Target_IpAddress{
						IpAddress: mockAddress,
					},
				},
			},
		}).DoAndReturn(func(_ context.Context, req *alb.RemoveTargetsRequest) (*operation.Operation, error) {
			logFunctionCalls(
				"ALBRemoveTarget",
				map[string]interface{}{"request": req},
				[]interface{}{&operation.Operation{}, nil})
			return &operation.Operation{}, nil
		}),
	)
}
