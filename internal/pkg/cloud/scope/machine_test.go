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
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	loadbalancer "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/services/loadbalancers"
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

func TestMachineScope_GetZoneID(t *testing.T) {
	g := NewWithT(t)

	zoneA := "ru-central1-a"
	zoneB := "ru-central1-b"

	tests := []struct {
		testName    string
		scope       scope.MachineScope
		want        string
		wantErr     bool
		errContains string
	}{
		{
			testName: "Return zone from YandexMachine.Spec.ZoneID if set",
			scope: scope.MachineScope{
				Machine:       &v1beta1.Machine{},
				YandexMachine: &infrav1.YandexMachine{Spec: infrav1.YandexMachineSpec{ZoneID: &zoneA}},
			},
			want: "ru-central1-a",
		},
		{
			testName: "Return zone from Machine.Spec.FailureDomain if ZoneID not set",
			scope: scope.MachineScope{
				Machine:       &v1beta1.Machine{Spec: v1beta1.MachineSpec{FailureDomain: &zoneB}},
				YandexMachine: &infrav1.YandexMachine{},
			},
			want: "ru-central1-b",
		},
		{
			testName: "YandexMachine.Spec.ZoneID takes priority over FailureDomain",
			scope: scope.MachineScope{
				Machine:       &v1beta1.Machine{Spec: v1beta1.MachineSpec{FailureDomain: &zoneB}},
				YandexMachine: &infrav1.YandexMachine{Spec: infrav1.YandexMachineSpec{ZoneID: &zoneA}},
			},
			want: "ru-central1-a",
		},
		{
			testName: "Return error if neither ZoneID nor FailureDomain is set",
			scope: scope.MachineScope{
				Machine:       &v1beta1.Machine{},
				YandexMachine: &infrav1.YandexMachine{},
			},
			wantErr:     true,
			errContains: "zone not specified",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(_ *testing.T) {
			got, err := test.scope.GetZoneID()
			if test.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(got).To(Equal(test.want))
			}
		})
	}
}

// fakeClusterGetter implements cloud.ClusterGetter for testing.
type fakeClusterGetter struct {
	failureDomains []infrav1.FailureDomainSpec
}

func (f *fakeClusterGetter) GetFailureDomains() []infrav1.FailureDomainSpec {
	return f.failureDomains
}

func (f *fakeClusterGetter) GetFolderID() string                 { return "" }
func (f *fakeClusterGetter) GetClient() yandex.Client            { return nil }
func (f *fakeClusterGetter) GetNetworkID() string                { return "" }
func (f *fakeClusterGetter) GetLabels() infrav1.Labels           { return nil }
func (f *fakeClusterGetter) GetLBType() infrav1.LoadBalancerType { return "" }
func (f *fakeClusterGetter) GetLBSpec() infrav1.LoadBalancerSpec { return infrav1.LoadBalancerSpec{} }
func (f *fakeClusterGetter) GetLBName() string                   { return "" }

func TestMachineScope_GetSubnetID(t *testing.T) {
	g := NewWithT(t)

	zoneB := "ru-central1-b"
	zoneC := "ru-central1-c"

	tests := []struct {
		testName    string
		scope       scope.MachineScope
		want        string
		wantErr     bool
		errContains string
	}{
		{
			testName: "Return subnet from NetworkInterfaces if set",
			scope: scope.MachineScope{
				Machine: &v1beta1.Machine{},
				YandexMachine: &infrav1.YandexMachine{
					Spec: infrav1.YandexMachineSpec{
						NetworkInterfaces: []infrav1.NetworkInterface{
							{SubnetID: "explicit-subnet-id"},
						},
					},
				},
			},
			want: "explicit-subnet-id",
		},
		{
			testName: "Return subnet from failure domains based on zone",
			scope: scope.MachineScope{
				Machine:       &v1beta1.Machine{Spec: v1beta1.MachineSpec{FailureDomain: &zoneB}},
				YandexMachine: &infrav1.YandexMachine{},
				ClusterGetter: &fakeClusterGetter{
					failureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
						{ZoneID: "ru-central1-b", SubnetID: "subnet-b"},
						{ZoneID: "ru-central1-d", SubnetID: "subnet-d"},
					},
				},
			},
			want: "subnet-b",
		},
		{
			testName: "Return error if zone not found in failure domains",
			scope: scope.MachineScope{
				Machine:       &v1beta1.Machine{Spec: v1beta1.MachineSpec{FailureDomain: &zoneC}},
				YandexMachine: &infrav1.YandexMachine{},
				ClusterGetter: &fakeClusterGetter{
					failureDomains: []infrav1.FailureDomainSpec{
						{ZoneID: "ru-central1-a", SubnetID: "subnet-a"},
					},
				},
			},
			wantErr:     true,
			errContains: "no subnet found for zone",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(_ *testing.T) {
			got, err := test.scope.GetSubnetID()
			if test.wantErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(test.errContains))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(got).To(Equal(test.want))
			}
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
