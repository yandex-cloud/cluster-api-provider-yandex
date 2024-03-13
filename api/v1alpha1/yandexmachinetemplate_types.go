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

// YandexMachineTemplateResource describes the data needed to create am YandexMachine from a template.
type YandexMachineTemplateResource struct {
	// Standard object's metadata.
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification of the desired behavior of the machine.
	Spec YandexMachineSpec `json:"spec"`
}

// YandexMachineTemplateSpec defines the desired state of YandexMachineTemplate.
type YandexMachineTemplateSpec struct {
	Template YandexMachineTemplateResource `json:"template"`
}

//+kubebuilder:object:root=true

// YandexMachineTemplate is the Schema for the yandexmachinetemplates API.
type YandexMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec YandexMachineTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// YandexMachineTemplateList contains a list of YandexMachineTemplate.
type YandexMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []YandexMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&YandexMachineTemplate{}, &YandexMachineTemplateList{})
}
