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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows cleaning up resources associated with
	// YandexMachine before removing it from the apiserver.
	MachineFinalizer = "yandexmachine.infrastructure.cluster.x-k8s.io"
)

// YandexMachineSpec defines the desired state of YandexMachine.
type YandexMachineSpec struct {
	// ProviderID is the unique identifier as specified by the cloud provider.
	ProviderID *string `json:"providerID,omitempty"`

	// FolderID is the identifier of YandexCloud folder.
	FolderID string `json:"folderID"`

	// ZoneID is the identifier of YandexCloud availability zone.
	// +optional
	ZoneID *string `json:"zoneID,omitempty"`

	// PlatformID is the identifier of YandexCloud current CPU model.
	// For example: standard-v1, standard-v2, standard-v3, highfreq-v3
	// With GPU: gpu-standard-v1, gpu-standard-v2, gpu-standard-v3, standard-v3-t4
	// More information https://cloud.yandex.ru/ru/docs/compute/concepts/vm-platforms .
	// +optional
	PlatformID *string `json:"platformID,omitempty"`

	// TargetGroupID is the identifier of LoadBalancer TargetGroup.
	// Only for ControlPlane.
	// +optional
	TargetGroupID *string `json:"targetGroupID,omitempty"`

	// BootDisk is boot storage configuration for YandexCloud VM.
	BootDisk Disk `json:"bootDisk"`

	// Resources represents different types of YandexCloud VM instance resources
	Resources Resources `json:"resources"`

	// NetworkInterfaces is a network interfaces configurations for YandexCloud VM
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

// NetworkInterface defines the network interface configuration of YandexCloud VM.
type NetworkInterface struct {
	// SubnetID is the identifier of subnetwork to use for this instance.
	SubnetID string `json:"subnetID"`

	// PublicIP is set to true if public IP for YandexCloud VM is needed.
	// +optional
	PublicIP *bool `json:"publicIP,omitempty"`
}

// Resources defines the YandexCloud VM resources, like cores, memory etc.
type Resources struct {
	// Memory is an amount of RAM memory for YandexCloud VM
	// Allows to specify k,M,G... or Ki,Mi,Gi... suffixes
	// For more information see https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity .
	Memory resource.Quantity `json:"memory"`

	// Cores is the number of cpu cores for YandexCloud VM.
	Cores int64 `json:"cores"`

	// CoreFraction is baseline level of CPU performance with the ability to burst performance above that baseline level.
	// This field sets baseline performance for each core.
	// For more information see https://yandex.cloud/en/docs/compute/concepts/performance-levels
	// +optional
	CoreFraction *int64 `json:"coreFraction,omitempty"`

	// GPUs is the number of GPUs available for YandexCloud VM.
	// +optional
	GPUs *int64 `json:"gpus,omitempty"`
}

// Disk defines YandexCloud VM disk configuration
type Disk struct {
	// Type is the disk storage type for YandexCloud VM
	// Possible values: network-ssd, network-hdd, network-ssd-nonreplicated, network-ssd-io-m3
	// More information https://cloud.yandex.ru/ru/docs/compute/concepts/disk .
	// +optional
	Type *string `json:"type,omitempty"`

	// Size is an disk size
	// Allows to specify k,M,G... or Ki,Mi,Gi... suffixes
	// For more information see https://pkg.go.dev/k8s.io/apimachinery/pkg/api/resource#Quantity .
	Size resource.Quantity `json:"size"`

	// ImageID is the identifier for OS image of YandexCloud VM.
	ImageID string `json:"imageID"`
}

// YandexMachineStatus defines the observed state of YandexMachine.
type YandexMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the YandexCloud instance associated addresses.
	// +optional
	Addresses []corev1.NodeAddress `json:"addresses,omitempty"`

	// InstanceStatus is the status of the Yandex instance for this machine.
	// +optional
	InstanceStatus *InstanceStatus `json:"instanceState,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the YandexMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// YandexMachine is the Schema for the yandexmachines API.
type YandexMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   YandexMachineSpec   `json:"spec,omitempty"`
	Status YandexMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// YandexMachineList contains a list of YandexMachine.
type YandexMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []YandexMachine `json:"items"`
}

// GetConditions returns the list of conditions for an Yandex Machine API object.
func (ym *YandexMachine) GetConditions() clusterv1.Conditions {
	return ym.Status.Conditions
}

// SetConditions will set the given conditions on an Yandex Machine API object.
func (ym *YandexMachine) SetConditions(conditions clusterv1.Conditions) {
	ym.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&YandexMachine{}, &YandexMachineList{})
}
