package v1alpha1

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//nolint:lll // controller-gen marker
//+kubebuilder:webhook:verbs=delete,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha1-yandexidentity,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=yandexidentities,versions=v1alpha1,name=mutation.yandexidentities.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

type yandexIdentityAdmitter struct {
	platformClient client.Client
}

// NewYandexIdentityDeletionBlocker returns a new admission.Handler that prevents deletion of YandexIdentity objects
// that are linked to YandexCluster objects.
func NewYandexIdentityDeletionBlocker(platformClient client.Client) admission.Handler {
	return &yandexIdentityAdmitter{platformClient: platformClient}
}

func (m *yandexIdentityAdmitter) Handle(ctx context.Context, req admission.Request) admission.Response {
	identity, ok := req.Object.Object.(*YandexIdentity)
	if !ok {
		return admission.Errored(http.StatusBadRequest, errors.New("failed to convert runtime Object to YandexIdentity"))
	}

	clusterList := YandexClusterList{}
	if err := m.platformClient.List(ctx, &clusterList, client.MatchingLabels(identity.GenerateLabelsForCluster())); err != nil {
		return admission.Errored(
			http.StatusInternalServerError,
			errors.Wrapf(err, "failed to list clusters linked to identity %s", identity.Name),
		)
	}

	if len(clusterList.Items) > 0 {
		return admission.Denied("identity is linked to clusters")
	}

	return admission.Allowed("identity is not linked to any cluster")
}
