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
	"testing"

	. "github.com/onsi/gomega"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestCLoudScope_GetLBName(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		testName string
		scope    *scope.ClusterScope
		want     string
	}{
		{
			testName: "Return the name of the load balancer, if it is defined in the specification",
			scope: &scope.ClusterScope{
				YandexCluster: &infrav1.YandexCluster{
					Spec: infrav1.YandexClusterSpec{
						LoadBalancer: infrav1.LoadBalancerSpec{Name: "test-lb"},
					},
				},
			},
			want: "test-lb",
		},
		{
			testName: "Return generated name of the load balancer, if name is not defined",
			scope: &scope.ClusterScope{
				Cluster: &v1beta1.Cluster{ObjectMeta: v1.ObjectMeta{Name: "test-cluster"}},
				YandexCluster: &infrav1.YandexCluster{
					Spec: infrav1.YandexClusterSpec{
						LoadBalancer: infrav1.LoadBalancerSpec{},
					}},
			},
			want: "test-cluster-api",
		},
		{
			testName: "Return generated name of the load balancer only with alphnumeric characters",
			scope: &scope.ClusterScope{
				Cluster: &v1beta1.Cluster{ObjectMeta: v1.ObjectMeta{Name: "some_strange_name"}},
				YandexCluster: &infrav1.YandexCluster{
					Spec: infrav1.YandexClusterSpec{
						LoadBalancer: infrav1.LoadBalancerSpec{},
					}},
			},
			want: "somestrangename-api",
		},
		{
			testName: "Return hashed name of the load balancer, if definded name too long",
			scope: &scope.ClusterScope{
				Cluster: &v1beta1.Cluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "some-very-very-very-very-very-very-loooooooooooooooooooooooooong-name"}},
				YandexCluster: &infrav1.YandexCluster{
					Spec: infrav1.YandexClusterSpec{
						LoadBalancer: infrav1.LoadBalancerSpec{},
					}},
			},
			want: "cluster-7de72134c46ee934-api",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(_ *testing.T) {
			got := test.scope.GetLBName()
			g.Expect(got).To(Equal(test.want))
		})
	}
}
