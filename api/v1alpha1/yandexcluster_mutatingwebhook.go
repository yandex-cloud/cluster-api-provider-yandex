package v1alpha1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//nolint:lll // controller-gen marker
//+kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha1-yandexcluster-identitylink,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=yandexclusters,versions=v1alpha1,name=mutation.yandexclusters.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

type yandexClusterAdmitter struct {
	decoder *admission.Decoder
}

// NewYandexClusterAdmitter returns a new admission.Handler that adds identityRef labels to YandexCluster objects.
func NewYandexClusterAdmitter(decoder *admission.Decoder) admission.Handler {
	return &yandexClusterAdmitter{decoder: decoder}
}

// Handle updates identityRef labels in YandexCluster objects.
func (m *yandexClusterAdmitter) Handle(_ context.Context, req admission.Request) admission.Response {
	yandexclusterlog.V(10).Info("mutate identity ref on YandexCluster change")

	if req.Operation == admissionv1.Delete {
		// we only care about create and update operations
		return admission.Allowed("allowed")
	}

	yandexCluster := &YandexCluster{}
	if err := m.decoder.DecodeRaw(req.Object, yandexCluster); err != nil {
		yandexclusterlog.Error(err, "failed to decode request")

		return admission.Errored(http.StatusBadRequest, errors.Wrap(err, "failed to decode request"))
	}

	if yandexCluster.Spec.IdentityRef != nil {
		key, val := generateIdentityLabelKeyAndValue(yandexCluster.Spec.IdentityRef.Name, yandexCluster.Spec.IdentityRef.Namespace)
		currentValue := yandexCluster.GetLabels()[key]

		yandexclusterlog.Info("identityRef label", "key", key, "value", val, "currentValue", currentValue)
		// if the label does not exist or the value is different, we need to add the label
		if currentValue != val {
			upd := yandexCluster.DeepCopy()
			if upd.Labels == nil {
				upd.Labels = map[string]string{}
			}

			// add new label
			upd.Labels[key] = val

			// remove old label if it exists
			if currentValue != "" {
				delete(upd.Labels, currentValue)
			}

			rawUpdate, err := json.Marshal(upd)
			if err != nil {
				admission.Errored(http.StatusInternalServerError, errors.Wrap(err, "failed to marshal object"))
			}

			yandexclusterlog.Info("updating identityRef label", "key", key, "value", val)

			return admission.PatchResponseFromRaw(req.Object.Raw, rawUpdate)
		}

		yandexclusterlog.Info("identityRef label already exists", "key", key, "value", val)

		return admission.Allowed("identityRef label already exists")
	}

	yandexclusterlog.Info("identityRef is nil")

	// from this moment on, identityRef is nil
	// if identityRef is nil, nothing to do
	if req.Operation == admissionv1.Create {
		yandexclusterlog.Info("identityRef is nil, nothing to do")

		return admission.Allowed("allowed")
	}

	// let's check previous state
	oldYandexCluster := &YandexCluster{}
	if err := m.decoder.DecodeRaw(req.OldObject, oldYandexCluster); err != nil {
		yandexclusterlog.Error(err, "failed to decode request")

		return admission.Errored(http.StatusBadRequest, errors.Wrap(err, "failed to decode request"))
	}

	// if oldYandexCluster.Spec.IdentityRef was not nil, we need to remove the label
	if oldYandexCluster.Spec.IdentityRef != nil {
		// get the key
		key, _ := generateIdentityLabelKeyAndValue(oldYandexCluster.Spec.IdentityRef.Name, oldYandexCluster.Spec.IdentityRef.Namespace)

		yandexclusterlog.Info("removing identityRef label", "key", key)

		// check if the label exists
		if _, ok := oldYandexCluster.GetLabels()[key]; ok {
			upd := yandexCluster.DeepCopy()

			delete(upd.Labels, key)

			rawUpdate, err := json.Marshal(upd)
			if err != nil {
				admission.Errored(http.StatusInternalServerError, errors.Wrap(err, "failed to marshal object"))
			}

			yandexclusterlog.Info("removing identityRef label", "key", key)

			return admission.PatchResponseFromRaw(req.Object.Raw, rawUpdate)
		}
		// no label to remove means nothing to do
	}

	yandexclusterlog.Info("identityRef was nil and still is nil, nothing to do")

	// if identityRef was nil and still is nil, nothing to do
	return admission.Allowed("allowed")
}
