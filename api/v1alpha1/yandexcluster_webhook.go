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
	"net"
	"net/netip"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AddrType is a address type
type AddrType int

// Possible address types
const (
	Unknown AddrType = iota + 1
	FQDN
	IPV4
	IPV6
)

// log is for logging in this package.
var yandexclusterlog = logf.Log.WithName("yandexcluster-resource")

// SetupWebhookWithManager creates a validation webhook
func (c *YandexCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}

//nolint:lll // controller-gen marker
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-yandexcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=yandexclusters,versions=v1alpha1,name=validation.yandexclusters.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Defaulter = &YandexCluster{}
	_ webhook.Validator = &YandexCluster{}
)

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (c *YandexCluster) Default() {
	yandexclusterlog.Info("default", "name", c.Name)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (c *YandexCluster) ValidateCreate() (admission.Warnings, error) {
	yandexclusterlog.Info("validate create", "name", c.Name)
	var allErrs field.ErrorList

	if c.Spec.LoadBalancer.Type == LoadBalancerTypeNLB {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancer", "type"),
				c.Spec.LoadBalancer.Type, "network load balancer support will be added in future releases, use application load balancer instead"),
		)
	}

	if c.Spec.LoadBalancer.Listener.Address != "" && !isIPv4(c.Spec.LoadBalancer.Listener.Address) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancer", "listener", "address"),
				c.Spec.LoadBalancer.Listener.Address, "field must be a valid IPv4 address"),
		)
	}

	if !reflect.DeepEqual(c.Spec.ControlPlaneEndpoint, clusterv1.APIEndpoint{}) {
		allErrs = append(allErrs, isControlPlaneEndpointValid(c.Spec.ControlPlaneEndpoint)...)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(GroupVersion.WithKind("YandexCluster").GroupKind(), c.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (c *YandexCluster) ValidateUpdate(oldRaw runtime.Object) (admission.Warnings, error) {
	yandexclusterlog.Info("validate update", "name", c.Name)
	var allErrs field.ErrorList
	old, ok := oldRaw.(*YandexCluster)
	if !ok {
		return nil, apierrors.NewBadRequest("failed to convert runtime Object to YandexCluster")
	}

	if !reflect.DeepEqual(old.Spec.LoadBalancer, c.Spec.LoadBalancer) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancer"), c.Spec.LoadBalancer, "field is immutable"),
		)
	}

	// We allow you to change the ControlPlaneEndpoint only if this field has not been set before.
	// In all other cases, this field is immutable.
	if !reflect.DeepEqual(c.Spec.ControlPlaneEndpoint, old.Spec.ControlPlaneEndpoint) {
		if reflect.DeepEqual(old.Spec.ControlPlaneEndpoint, clusterv1.APIEndpoint{}) {
			allErrs = append(allErrs, isControlPlaneEndpointValid(c.Spec.ControlPlaneEndpoint)...)
		} else {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("spec", "controlPlaneEndpoint"), c.Spec.ControlPlaneEndpoint, "field is immutable"),
			)
		}
	}
	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(GroupVersion.WithKind("YandexCluster").GroupKind(), c.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (c *YandexCluster) ValidateDelete() (admission.Warnings, error) {
	yandexclusterlog.Info("validate delete", "name", c.Name)
	return nil, nil
}

// getAddrType returns type of address
func getAddrType(s string) AddrType {
	parsedIP, err := netip.ParseAddr(s)
	if err != nil {
		if _, errfqdn := net.LookupHost(s); errfqdn == nil {
			return FQDN
		}
		return Unknown

	}
	if parsedIP.Is4() {
		return IPV4
	}
	if parsedIP.Is6() {
		return IPV6
	}
	return Unknown
}

// isIPv4 checks if address is an ipv4 address
func isIPv4(s string) bool {
	return getAddrType(s) == IPV4
}

// isIPv4orFQDN checks if address is an ipv4 address or FQDN
func isIPv4orFQDN(s string) bool {
	result := getAddrType(s)
	return result == IPV4 || result == FQDN
}

// isControlPlaneEndpointValid checks if ControlPlaneEndpoint has valid fields
func isControlPlaneEndpointValid(cp clusterv1.APIEndpoint) field.ErrorList {
	var errs field.ErrorList
	if cp.Host == "" || cp.Port == 0 {
		errs = append(errs,
			field.Invalid(field.NewPath("spec", "controlPlaneEndpoint"),
				cp.Port, "fields have to be not empty"),
		)
	}
	if !isIPv4orFQDN(cp.Host) {
		errs = append(errs,
			field.Invalid(field.NewPath("spec", "controlPlaneEndpoint", "host"),
				cp.Host, "field has to be IPv4 or FQDN"),
		)
	}

	if cp.Port < 1 || cp.Port > 65535 {
		errs = append(errs,
			field.Invalid(field.NewPath("spec", "controlPlaneEndpoint", "port"),
				cp.Port, "field has to be a network port from range 1-65535"),
		)
	}
	return errs
}
