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
	// IdentityFinalizer allows cleaning up resources associated with
	// IdentityFinalizer before removing it from the apiserver.
	IdentityFinalizer = "yandexidentity.infrastructure.cluster.x-k8s.io"
)

//+kubebuilder:validation:Required

// YandexIdentitySpec defines the desired state of YandexIdentity.
type YandexIdentitySpec struct {
	// Name is the name of a secret in the same namespace as YandexIdentity.
	// The secret must contain a key, which contains an valid iamkey.Key as of github.com/yandex-cloud/go-sdk/iamkey.
	// +required
	SecretName string `json:"secretname"`

	// KeyName is the name of the key in the secret. Default is `YandexCloudSAKey`.
	// +optional
	KeyName string `json:"keyname" default:"YandexCloudSAKey"`
}

// YandexIdentityStatus defines the observed state of YandexIdentity.
type YandexIdentityStatus struct {
	// Ready is true when the secret is checked and resource is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// KeyHash is the hash of the secret.
	// +optional
	KeyHash string `json:"keyhash,omitempty"`

	// Conditions is a list of conditions and their status.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// Linked clusters
	// +optional
	LinkedClusters []string `json:"linkedClusters,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// YandexIdentity is the Schema for the YandexIdentities API.
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

// GenerateSecretFinalizer returns the finalizer string for the YandexIdentity API object.
func (i *YandexIdentity) GenerateSecretFinalizer() string {
	return i.Name + "/" + IdentityFinalizer
}

// GetConditions returns the list of conditions for an YandexIdentity API object.
func (i *YandexIdentity) GetConditions() clusterv1.Conditions {
	return i.Status.Conditions
}

// SetConditions will set the given conditions on an YandexIdentity API object.
func (i *YandexIdentity) SetConditions(conditions clusterv1.Conditions) {
	i.Status.Conditions = conditions
}

// GenerateLabelsForCluster returns the labels that should be applied to the cluster object.
func (i *YandexIdentity) GenerateLabelsForCluster() map[string]string {
	key, value := generateIdentityLabelKeyAndValue(i.Name, i.Namespace)

	return map[string]string{
		key: value,
	}
}

func generateIdentityLabelKeyAndValue(name, namespace string) (string, string) {
	return "yandexidentity/" + namespace, name
}

func init() {
	SchemeBuilder.Register(&YandexIdentity{}, &YandexIdentityList{})
}
