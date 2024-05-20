/*
Copyright 2024 The Kubernetes Authors.

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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/controllers"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	machineName string = "test"
	secretName  string = "datasecret"
)

// newCAPIMachine returns CAPI Machine.
func newCAPIMachine() *clusterv1.Machine {
	return &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			},
			Name:      machineName,
			Namespace: nsName,
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: clusterName,
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: ptr.To[string]("datasecretname"),
			},
		},
	}
}

// newCAPIMachineWithInfrastructureRef returns CAPI Machine with infrastructure reference.
func newCAPIMachineWithInfrastructureRef() *clusterv1.Machine {
	cm := newCAPIMachine()
	cm.Spec.InfrastructureRef = corev1.ObjectReference{
		Kind:       "YandexMachine",
		Namespace:  nsName,
		Name:       machineName,
		APIVersion: infrav1.GroupVersion.String(),
	}
	cm.Spec.Bootstrap = clusterv1.Bootstrap{
		DataSecretName: ptr.To[string](secretName),
	}

	return cm
}

// newDataSecret returns Secret with filled Data field
func newDataSecret() *corev1.Secret {
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: nsName,
		},
		Data: map[string][]byte{
			"value": []byte("verysecret,data"),
		},
	}
	return &s
}

// newYandexMachine returns Yandex Machine.
func newYandexMachine() *infrav1.YandexMachine {
	return &infrav1.YandexMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: nsName,
		},
		Spec: infrav1.YandexMachineSpec{
			FolderID: "folderid",
			Resources: infrav1.Resources{
				Cores:  1,
				Memory: resource.MustParse("1Gi"),
			},
			BootDisk: infrav1.Disk{
				Size:    resource.MustParse("10Gi"),
				ImageID: "imageid",
			},
			NetworkInterfaces: []infrav1.NetworkInterface{
				{
					SubnetID: "subnetid",
				},
			},
		},
	}
}

// newYandexMachineWithOwnerReference returns Yandex Machine with owner reference.
func newYandexMachineWithOwnerReference() *infrav1.YandexMachine {
	ym := newYandexMachine()
	ym.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "cluster.x-k8s.io/v1beta1",
			Kind:       "Machine",
			Name:       machineName,
			UID:        types.UID("some-uid"),
		},
	}
	return ym
}

var _ = Describe("YandexMachine Reconciler check", func() {
	BeforeEach(func() {})
	AfterEach(func() {})

	When("Reconcile an empty YandexMachine", func() {
		It("should not error with minimal set up", func() {
			reconciler := &controllers.YandexMachineReconciler{
				Client: k8sClient,
			}

			ym := newYandexMachine()
			Expect(k8sClient.Create(ctx, ym)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, ym)
				Expect(err).NotTo(HaveOccurred())
			}()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: ym.Namespace,
					Name:      ym.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
			time.Sleep(5 * time.Second)
		})
	})

	When("Reconcile an YandexMachine owned by CAPI Machine", func() {
		It("should not error and get ready status eventually", func() {
			mockCLient.EXPECT().ComputeCreate(gomock.Any(), gomock.Any()).Return("machine-uid", nil).Times(1)
			mockCLient.EXPECT().ComputeGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("some error")).Times(1)
			mockCLient.EXPECT().ComputeGet(gomock.Any(), "machine-uid").Return(
				&compute.Instance{
					Name:   machineName,
					Id:     "machine-uid",
					Status: compute.Instance_RUNNING,
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							PrimaryV4Address: &compute.PrimaryAddress{
								Address: "1.2.3.4",
							},
						},
					},
				}, nil).Times(3)
			mockCLient.EXPECT().ComputeGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("some error")).AnyTimes()
			mockCLient.EXPECT().ComputeDelete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			cc := newCAPIClusterWithInfrastructureReference()
			Expect(k8sClient.Create(ctx, cc)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, cc)
				Expect(err).NotTo(HaveOccurred())
			}()

			yc := newYandexClusterWithOwnerReference()
			Expect(k8sClient.Create(ctx, yc)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, yc)
				Expect(err).NotTo(HaveOccurred())
			}()

			s := newDataSecret()
			Expect(k8sClient.Create(ctx, s)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, s)
				Expect(err).NotTo(HaveOccurred())
			}()

			cm := newCAPIMachineWithInfrastructureRef()
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, cm)
				Expect(err).NotTo(HaveOccurred())
			}()

			ym := newYandexMachineWithOwnerReference()
			Expect(k8sClient.Create(ctx, ym)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, ym)
				Expect(err).NotTo(HaveOccurred())
			}()

			Eventually(func(g Gomega) bool {
				ym := &infrav1.YandexMachine{}
				nn := types.NamespacedName{
					Name:      clusterName,
					Namespace: nsName,
				}
				Expect(k8sClient.Get(ctx, nn, ym)).Should(Succeed())
				return ym.Status.Ready
			}).Should(BeTrue())
		})
	})

})
