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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	}

	scp, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:        k8sClient,
		Machine:       cm,
		LoadBalancer:  loadbalancer.New(&scope.ClusterScope{}),
		ClusterGetter: &scope.ClusterScope{},
		YandexMachine: ym,
	})
	if err != nil {
		return nil, err
	}

	return scp, nil
}
