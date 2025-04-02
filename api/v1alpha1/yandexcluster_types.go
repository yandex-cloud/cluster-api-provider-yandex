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
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// ClusterFinalizer allows cleaning up resources associated with
	// YandexCluster before removing it from the apiserver.
	ClusterFinalizer = "yandexcluster.infrastructure.cluster.x-k8s.io"
	// LoadBalancerTypeALB is the name of the application load balancer type.
	LoadBalancerTypeALB LoadBalancerType = "ALB"
	// LoadBalancerTypeNLB is the name of the network load balancer type.
	LoadBalancerTypeNLB LoadBalancerType = "NLB"
)

//+kubebuilder:validation:Required

// Labels defines a map of tags.
// No more than 64 per resource. The string length in characters for each key must be 1-63.
// Each key must match the regular expression [a-z][-_./\@0-9a-z]*.
// The maximum string length in characters for each value is 63.
// Each value must match the regular expression [-_./\@0-9a-z]*.
// More information https://yandex.cloud/docs/overview/concepts/services#labels.
type Labels map[string]string

// LoadBalancerType is a type of a loadbalancer.
// More details about loadbalancer type in YandexCloud docs:
// NLB https://yandex.cloud/ru/services/network-load-balancer .
// ALB https://yandex.cloud/ru/services/application-load-balancer .
type LoadBalancerType string

// YandexClusterSpec defines the desired state of YandexCluster.
type YandexClusterSpec struct {
	// NetworkSpec encapsulates all things related to Yandex network.
	NetworkSpec NetworkSpec `json:"network,omitempty"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// Once set, the value cannot be changed.
	// Do not set it manually when creating YandexCluster as CAPY will set this for you
	// after creating load balancer based on LoadBalancer specification.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// FolderID is the identifier of YandexCloud folder to deploy the cluster to.
	// +required
	// +kubebuilder:validation:MinLength=1
	FolderID string `json:"folderID"`

	// LoadBalancer is a loadbalancer configuration for the kubernetes cluster API.
	// +required
	LoadBalancer LoadBalancerSpec `json:"loadBalancer"`

	// Labels is an optional set of labels to add to Yandex resources managed by the CAPY provider.
	// +optional
	Labels Labels `json:"labels,omitempty"`

	// IdentityRef is a reference to a YandexIdentity resource.
	// +optional
	IdentityRef *IdentityReference `json:"identityRef,omitempty"`
}

// LoadBalancerSpec is a loadbalancer configuration for the kubernetes cluster API.
type LoadBalancerSpec struct {
	// Type is a type of a loadbalancer, possible values are: NLB and ALB.
	// If Type not provided, loadbalancer type will be set to the ALB.
	// +optional
	// +kubebuilder:default=ALB
	// +kubebuilder:validation:Enum:=ALB;NLB
	Type LoadBalancerType `json:"type,omitempty"`

	// Name sets the name of the ALB load balancer. The name must be unique within your set of
	// load balancers for the folder, must have a minimum 3 and maximum of 63 characters,
	// must contain only alphanumeric characters or hyphens, and cannot begin or end with a hyphen.
	// Once set, the value cannot be changed.
	// +kubebuilder:validation:MinLength:=3
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern=`([a-z]([-a-z0-9]{0,61}[a-z0-9])?)?`
	// +optional
	Name string `json:"name,omitempty"`

	// ListenerSpec is a listener configuration for the load balancer.
	// +required
	Listener ListenerSpec `json:"listener"`

	// Load balancer backend port. Acceptable values are 1 to 65535, inclusive.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8443
	// +optional
	BackendPort int32 `json:"backendPort,omitempty"`

	// +optional
	// +kubebuilder:default={}
	Healthcheck HealtcheckSpec `json:"healthcheck,omitempty"`

	// SecurityGroups sets the security groups ID used by the load balancer.
	// If SecurityGroups not provided, new security group will be created for the load balancer.
	// More information https://yandex.cloud/ru/docs/vpc/concepts/security-groups.
	// +optional
	SecurityGroups []string `json:"securityGroups,omitempty"`
}

// ListenerSpec is a load balancer listener configuration for the kubernetes cluster api.
// More information https://yandex.cloud/ru/docs/application-load-balancer/concepts/application-load-balancer#listener.
type ListenerSpec struct {
	// load balancer listener ip address.
	// +optional
	Address string `json:"address,omitempty"`

	// load balancer listener port. Acceptable values are 1 to 65535, inclusive.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8443
	// +optional
	Port int32 `json:"port,omitempty"`

	// If Internal value is true, then a private IP will be used for the listener address.
	// +kubebuilder:default=true
	// +optional
	Internal bool `json:"internal,omitempty"`

	// Load balancer listener will be located in this subnet.
	// More information https://yandex.cloud/ru/docs/vpc/concepts/network#subnet.
	// +required
	Subnet SubnetSpec `json:"subnet"`
}

// SubnetSpec configures an Yandex Subnet.
type SubnetSpec struct {
	// ZoneID is the identifier of YandexCloud availability zone where the subnet resides.
	ZoneID string `json:"zoneID,omitempty"`

	// ID defines a unique identificator of the subnet to be used.
	ID string `json:"id,omitempty"`
}

// HealtcheckSpec configures load balancer healthchecks.
type HealtcheckSpec struct {
	// +optional
	// +kubebuilder:default=1
	HealthcheckTimeoutSec int `json:"healthcheckTimeoutSec,omitempty"`
	// +optional
	// +kubebuilder:default=3
	HealthcheckIntervalSec int `json:"healthcheckIntervalSec,omitempty"`
	// +optional
	// +kubebuilder:default=3
	HealthcheckThreshold int `json:"healthcheckThreshold,omitempty"`
}

// NetworkSpec encapsulates all things related to Yandex network.
type NetworkSpec struct {
	// ID is the unique identificator of the cloud network to be used.
	// More information https://yandex.cloud/ru/docs/vpc/concepts/network.
	ID string `json:"id,omitempty"`
}

// IdentityReference is a reference to a YandexIdentity resource.
type IdentityReference struct {
	// Name is the name of the YandexIdentity resource.
	Name string `json:"name"`

	// Namespace is the namespace of the YandexIdentity resource.
	Namespace string `json:"namespace"`
}

// NamespacedName returns the namespaced name of the YandexIdentity resource.
func (ir *IdentityReference) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: ir.Namespace,
		Name:      ir.Name,
	}
}

// YandexClusterStatus defines the observed state of YandexCluster.
type YandexClusterStatus struct {
	// Ready is true when the provider resource is ready.
	// +kubebuilder:default=false
	Ready        bool                 `json:"ready"`
	LoadBalancer LoadBalancerStatus   `json:"loadBalancerStatus,omitempty"`
	Conditions   clusterv1.Conditions `json:"conditions,omitempty"`
}

// LoadBalancerStatus encapsulates load balancer resources.
type LoadBalancerStatus struct {
	// The name of the load balancer.
	// +optional
	Name string `json:"name,omitempty"`

	// ListenerAddress is the IPV4 l address assigned to the load balancer listener,
	// created for the API Server.
	// +optional
	ListenerAddress string `json:"listenerAddress,omitempty"`

	// ListenerPort is the port assigned to the load balancer listener, created for the API Server.
	// +optional
	ListenerPort int32 `json:"listenerPort,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//nolint: lll // controller-gen markers
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this YandexCluster belongs"
//nolint: lll // controller-gen markers
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for YandexCloud instances"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.controlPlaneEndpoint",description="API Endpoint"

// YandexCluster is the Schema for the yandexclusters API.
type YandexCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   YandexClusterSpec   `json:"spec,omitempty"`
	Status YandexClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// YandexClusterList contains a list of YandexCluster.
type YandexClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []YandexCluster `json:"items"`
}

// GetConditions returns the list of conditions for an YandexCluster API object.
func (yc *YandexCluster) GetConditions() clusterv1.Conditions {
	return yc.Status.Conditions
}

// SetConditions will set the given conditions on an YandexCluster API object.
func (yc *YandexCluster) SetConditions(conditions clusterv1.Conditions) {
	yc.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&YandexCluster{}, &YandexClusterList{})
}
