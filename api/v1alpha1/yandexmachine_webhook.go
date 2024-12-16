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
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var log = logf.Log.WithName("yandexmachine-resource")

// SetupWebhookWithManager creates an YandexMachine validation webhook.
func (ym *YandexMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(ym).
		Complete()
}

//nolint:lll // controller-gen marker
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-yandexmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=yandexmachines,verbs=create;update,versions=v1alpha1,name=validation.yandexmachines.infrastructure.cluster.x-k8s.io,admissionReviewVersions=v1beta1

var (
	_ webhook.Defaulter = &YandexMachine{}
	_ webhook.Validator = &YandexMachine{}
)

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (ym *YandexMachine) Default() {
	log.Info("default", "name", ym.Name)
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (ym *YandexMachine) ValidateCreate() (admission.Warnings, error) {
	log.Info("validate create", "name", ym.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (ym *YandexMachine) ValidateUpdate(oldRaw runtime.Object) (admission.Warnings, error) {
	log.Info("validate update", "name", ym.Name)

	newYandexMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ym)
	if err != nil {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("YandexMachine").GroupKind(), ym.Name, field.ErrorList{
			field.InternalError(nil, errors.Wrap(err, "failed to convert new YandexMachine to unstructured object")),
		})
	}
	oldYandexMachine, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oldRaw)
	if err != nil {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("YandexMachine").GroupKind(), ym.Name, field.ErrorList{
			field.InternalError(nil, errors.Wrap(err, "failed to convert old YandexMachine to unstructured object")),
		})
	}

	newYandexMachineSpec := newYandexMachine["spec"].(map[string]interface{})
	oldYandexMachineSpec := oldYandexMachine["spec"].(map[string]interface{})

	// allow changes to providerID field.
	delete(oldYandexMachineSpec, "providerID")
	delete(newYandexMachineSpec, "providerID")

	if !reflect.DeepEqual(oldYandexMachineSpec, newYandexMachineSpec) {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("YandexMachine").GroupKind(), ym.Name, field.ErrorList{
			field.Forbidden(field.NewPath("spec"), "cannot be modified"),
		})
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (ym *YandexMachine) ValidateDelete() (admission.Warnings, error) {
	log.Info("validate delete", "name", ym.Name)
	return nil, nil
}
