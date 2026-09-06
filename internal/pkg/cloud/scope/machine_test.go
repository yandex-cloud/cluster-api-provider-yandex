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

package scope_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/services/loadbalancers"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMachineScope_ProviderID(t *testing.T) {
	g := NewWithT(t)

	t.Run("GetProviderID should return an empty string if CAPY has not yet set its value", func(_ *testing.T) {
		scp := scope.MachineScope{
			YandexMachine: &infrav1.YandexMachine{},
		}
		g.Expect(scp.GetProviderID()).To(BeEmpty())
	})

	t.Run("SetProviderID should set the providerID field in the required format", func(_ *testing.T) {
		scp := scope.MachineScope{
			YandexMachine: &infrav1.YandexMachine{},
		}
		scp.SetProviderID("test-machine")
		g.Expect(scp.GetProviderID()).To(Equal("yandex://test-machine"))
	})

	t.Run("GetInstanceID should return YandexCloud VM instance id from providerID", func(_ *testing.T) {
		scp := scope.MachineScope{
			YandexMachine: &infrav1.YandexMachine{},
		}
		scp.SetProviderID("test-machine")
		g.Expect(scp.GetInstanceID()).To(Equal("test-machine"))
	})
}

func TestMachineScope_GetBootstrapData(t *testing.T) {
	g := NewWithT(t)
	want := "bootstrap-data"

	goodSecret := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Namespace: "test", Name: "test-secret"},
		Data:       map[string][]byte{"value": []byte(want)},
	}
	scp, err := fakeScopeWithSecret(&goodSecret)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(scp).ToNot(BeNil())

	get, err := scp.GetBootstrapData()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(get).To(Equal(want))

	badSecret := corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Namespace: "test", Name: "test-secret"},
		Data:       map[string][]byte{"asdf": []byte(want)},
	}
	scp, err = fakeScopeWithSecret(&badSecret)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(scp).ToNot(BeNil())

	get, err = scp.GetBootstrapData()
	g.Expect(err).To(HaveOccurred())
	g.Expect(get).To(BeZero())
}

func TestMachineScope_GetInstanceReq_SchedulingPolicy(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		preemptible *bool
		want        bool
	}{
		{
			name:        "preemptible is not set, VM should not be preemptible",
			preemptible: nil,
			want:        false,
		},
		{
			name:        "preemptible is false, VM should not be preemptible",
			preemptible: ptr.To(false),
			want:        false,
		},
		{
			name:        "preemptible is true, VM should be preemptible",
			preemptible: ptr.To(true),
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			secret := corev1.Secret{
				ObjectMeta: v1.ObjectMeta{Namespace: "test", Name: "test-secret"},
				Data:       map[string][]byte{"value": []byte("bootstrap-data")},
			}
			scp, err := fakeScopeWithSecret(&secret)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(scp).ToNot(BeNil())

			scp.YandexMachine.Spec.Preemptible = tt.preemptible

			req, err := scp.GetInstanceReq()
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(req.GetSchedulingPolicy()).To(Equal(&compute.SchedulingPolicy{Preemptible: tt.want}))
		})
	}
}

// fakeScopeWithSecret creates machine scope with fakeclient.
func fakeScopeWithSecret(secret *corev1.Secret) (*scope.MachineScope, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := infrav1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	if err := k8sClient.Create(context.TODO(), secret); err != nil {
		return nil, err
	}

	cm := &v1beta1.Machine{
		Spec: v1beta1.MachineSpec{
			Bootstrap: v1beta1.Bootstrap{
				DataSecretName: &secret.Name,
			},
		},
	}
	ym := &infrav1.YandexMachine{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "test",
			Name:      "ym-test",
		},
		Spec: infrav1.YandexMachineSpec{
			ZoneID:     ptr.To("ru-central1-d"),
			PlatformID: ptr.To("standard-v3"),
			BootDisk: &infrav1.Disk{
				TypeID:  ptr.To("network-ssd"),
				Size:    resource.MustParse("10Gi"),
				ImageID: "test-image",
			},
			Resources: infrav1.Resources{
				Memory: resource.MustParse("2Gi"),
				Cores:  2,
			},
			NetworkInterfaces: []infrav1.NetworkInterface{{SubnetID: "test-subnet"}},
		},
	}
	clusterScope := &scope.ClusterScope{
		YandexCluster: &infrav1.YandexCluster{
			Spec: infrav1.YandexClusterSpec{FolderID: "test-folder"},
		},
	}

	scp, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:        k8sClient,
		Machine:       cm,
		LoadBalancer:  loadbalancer.New(clusterScope),
		ClusterGetter: clusterScope,
		YandexMachine: ym,
	})
	if err != nil {
		return nil, err
	}

	return scp, nil
}
