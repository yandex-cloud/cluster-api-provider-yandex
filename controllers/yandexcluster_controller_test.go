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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterName              string = "test"
	nsName                   string = "default"
	hostControlPlaneEndpoint string = "ya.ru"
	portControlPlaneEndpoint int32  = 8443
)

// newYandexCluster returns empty Yandex Cluser.
func newYandexCluster() *infrav1.YandexCluster {
	return &infrav1.YandexCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: nsName,
		},
		Spec: infrav1.YandexClusterSpec{
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
				Host: hostControlPlaneEndpoint,
				Port: portControlPlaneEndpoint,
			},
		},
	}
}

// newCAPIClusterWithInfrastructureReference return CAPI CLuster with infrastruture reference.
func newCAPIClusterWithInfrastructureReference() *clusterv1.Cluster {
	return &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: nsName,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				Namespace:  nsName,
				Name:       clusterName,
				APIVersion: infrav1.GroupVersion.String(),
				Kind:       "YandexCluster",
			},
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
				Host: hostControlPlaneEndpoint,
				Port: portControlPlaneEndpoint,
			},
		},
	}
}

// newYandexClusterWithOwnerReference returns Yandex Cluster with owner reference.
func newYandexClusterWithOwnerReference() *infrav1.YandexCluster {
	return &infrav1.YandexCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: nsName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "cluster.x-k8s.io/v1beta1",
					Kind:       "Cluster",
					Name:       clusterName,
					UID:        types.UID("some-uid"),
				},
			},
		},
		Spec: infrav1.YandexClusterSpec{
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
				Host: hostControlPlaneEndpoint,
				Port: portControlPlaneEndpoint,
			},
		},
	}
}

var _ = Describe("YandexCluster Reconciler check", func() {
	BeforeEach(func() {})
	AfterEach(func() {})

	When("Reconcile an empty YandexCluster", func() {
		It("should not error and not requeue the request", func() {
			reconciler := &controllers.YandexClusterReconciler{
				Client: k8sClient,
			}

			yc := newYandexCluster()
			Expect(k8sClient.Create(ctx, yc)).To(Succeed())
			defer func() {
				err := k8sClient.Delete(ctx, yc)
				Expect(err).NotTo(HaveOccurred())
			}()

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKey{
					Namespace: yc.Namespace,
					Name:      yc.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(result.Requeue).To(BeFalse())
		})
	})

	When("Reconcile an YandexCluster owned by CAPI Cluster", func() {
		It("Should not error and set up finalizer", func() {
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

			Eventually(func(g Gomega) []string {
				yc := &infrav1.YandexCluster{}
				nn := types.NamespacedName{
					Name:      clusterName,
					Namespace: nsName,
				}
				Expect(k8sClient.Get(ctx, nn, yc)).Should(Succeed())
				return yc.Finalizers
			}).Should(HaveLen(1))
		})
	})
})
