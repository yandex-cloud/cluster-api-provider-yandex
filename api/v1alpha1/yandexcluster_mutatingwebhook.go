package v1alpha1

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"gomodules.xyz/jsonpatch/v2"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//nolint:lll // controller-gen marker
//+kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha1-yandexcluster,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=yandexclusters,versions=v1alpha1,name=mutation.yandexclusters.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

type yandexClusterAdmitter struct {
}

// NewYandexClusterAdmitter returns a new admission.Handler that adds identityRef labels to YandexCluster objects.
func NewYandexClusterAdmitter() admission.Handler {
	return &yandexClusterAdmitter{}
}

const (
	identityLabelPath = "/metadata/labels"
)

func (m *yandexClusterAdmitter) Handle(_ context.Context, req admission.Request) admission.Response {
	yandexCluster, ok := req.Object.Object.(*YandexCluster)
	if !ok || yandexCluster == nil {
		return admission.Errored(http.StatusBadRequest, errors.New("failed to convert runtime Object to YandexCluster"))
	}

	if yandexCluster.Spec.IdentityRef != nil {
		key, val := generateIdentityLabelKeyAndValue(yandexCluster.Spec.IdentityRef.Name, yandexCluster.Spec.IdentityRef.Namespace)
		currentValue := yandexCluster.GetLabels()[key]
		// if the label does not exist or the value is different, we need to add the label
		if currentValue != val {
			patch := jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      fmt.Sprintf("%s/%s", identityLabelPath, key),
				Value:     val,
			}

			return admission.Patched("added label for identityRef", patch)
		}

		return admission.Allowed("identityRef label already exists")
	}

	// from this moment on, identityRef is nil

	// let's check previous state
	oldYandexCluster, ok := req.OldObject.Object.(*YandexCluster)
	if !ok && req.OldObject.Object != nil {
		return admission.Errored(http.StatusBadRequest, errors.New("failed to convert runtime Object to YandexCluster"))
	}

	// if oldYandexCluster is nil, nothing to do
	if oldYandexCluster == nil {
		return admission.Allowed("allowed")
	}

	// if oldYandexCluster.Spec.IdentityRef was not nil, we need to remove the label
	if oldYandexCluster.Spec.IdentityRef != nil {
		// get the key
		key, _ := generateIdentityLabelKeyAndValue(oldYandexCluster.Spec.IdentityRef.Name, oldYandexCluster.Spec.IdentityRef.Namespace)
		// check if the label exists
		if _, ok := oldYandexCluster.GetLabels()[key]; ok {
			patch := jsonpatch.JsonPatchOperation{
				Operation: "remove",
				Path:      fmt.Sprintf("%s/%s", identityLabelPath, key),
			}

			return admission.Patched("removed label for identityRef", patch)
		}
		// no label to remove means nothing to do
	}

	// if identityRef was nil and still is nil, nothing to do
	return admission.Allowed("allowed")
}
