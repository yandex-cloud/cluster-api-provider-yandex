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
)

const (
	// IdentityFinalizer allows cleaning up resources associated with
	// IdentityFinalizer before removing it from the apiserver.
	IdentityFinalizer = "yandexidentity.infrastructure.cluster.x-k8s.io"
)

// YandexIdentitySpec defines the desired state of YandexIdentity.
type YandexIdentitySpec struct {
	// Name is the name of a secret in the same namespace as YandexIdentity.
	// The secret must contain a key, which contains an valid iamkey.Key as of github.com/yandex-cloud/go-sdk/iamkey.
	// +kubebuilder:validation:Required
	SecretName string `json:"secretname"`

	// KeyName is the name of the key in the secret. Default is `YandexCloudSAKey`.
	// +kubebuilder:validation:Optional
	KeyName string `json:"keyname" default:"YandexCloudSAKey"`
}

// YandexIdentityStatus defines the observed state of YandexIdentity.
type YandexIdentityStatus struct {
	// Ready is true when the secret is checked and resource is ready.
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// YandexIdentity is the Schema for the YandexIdentitys API.
type YandexIdentity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   YandexIdentitySpec   `json:"spec,omitempty"`
	Status YandexIdentityStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// YandexIdentityList contains a list of YandexIdentity.
type YandexIdentityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []YandexIdentity `json:"items"`
}

func init() {
	SchemeBuilder.Register(&YandexIdentity{}, &YandexIdentityList{})
}
