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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// ClusterFinalizer allows cleaning up resources associated with
	// YandexCluster before removing it from the apiserver.
	ClusterFinalizer = "yandexcluster.infrastructure.cluster.x-k8s.io"
)

// YandexClusterSpec defines the desired state of YandexCluster.
type YandexClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// LoadBalancer is a loadbalancer configuration for the kubernetes cluster api.
	// +optional
	LoadBalancer LoadBalancerSpec `json:"loadBalancer"`
}

// LoadBalancerSpec is a loadbalancer configuration for the kubernetes cluster api.
type LoadBalancerSpec struct {
	// Type is a type of a loadbalancer, possible values are: NLB and ALB.
	// More details about loadbalancer type in YandexCloud docs:
	// NLB https://yandex.cloud/ru/services/network-load-balancer .
	// ALB https://yandex.cloud/ru/services/application-load-balancer .
	// +optional
	Type LoadBalancerType `json:"type"`
}

// LoadBalancerType is a type of a loadbalancer.
type LoadBalancerType string

const (
	// ApplicationLoadBalancer is the string representing an Yandex Cloud Application LoadBalancer.
	ApplicationLoadBalancer LoadBalancerType = "ALB"
	// NetworkLoadBalancer is the string representing an Yandex Cloud Network LoadBalancer.
	NetworkLoadBalancer LoadBalancerType = "NLB"
)

// YandexClusterStatus defines the observed state of YandexCluster.
type YandexClusterStatus struct {
	// Ready is true when the provider resource is ready.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// YandexCluster is the Schema for the yandexclusters API.
type YandexCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   YandexClusterSpec   `json:"spec,omitempty"`
	Status YandexClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// YandexClusterList contains a list of YandexCluster.
type YandexClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []YandexCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&YandexCluster{}, &YandexClusterList{})
}
