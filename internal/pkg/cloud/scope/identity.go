package scope

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// IdentityScopeParams defines the input parameters used to create a new IdentityScope.
type IdentityScopeParams struct {
	Client             client.Client
	YandexClientGetter yandex.YandexClientGetter
	YandexIdentity     *infrav1.YandexIdentity
}

// IdentityScope defines a scope defined around a YandexIdentity.
type IdentityScope struct {
	Identity           *infrav1.YandexIdentity
	client             client.Client
	patchHelper        *patch.Helper
	yandexClientGetter yandex.YandexClientGetter
	secret             *corev1.Secret
}

// NewIdentityScope creates a new IdentityScope.
func NewIdentityScope(params IdentityScopeParams) (*IdentityScope, error) {
	if params.Client == nil {
		return nil, errors.New("failed to generate new Identity scope from nil Client")
	}

	if params.YandexClientGetter == nil {
		return nil, errors.New("failed to generate new Identity scope from nil Builder")
	}

	if params.YandexIdentity == nil {
		return nil, errors.New("failed to generate new Identity scope from nil YandexIdentity")
	}

	helper, err := patch.NewHelper(params.YandexIdentity, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper for Identity scope")
	}

	return &IdentityScope{
		client:             params.Client,
		Identity:           params.YandexIdentity,
		yandexClientGetter: params.YandexClientGetter,
		patchHelper:        helper,
	}, nil
}

// getSecret returns the secret of the identity
func (s *IdentityScope) getSecret(ctx context.Context) (*corev1.Secret, error) {
	if s.secret != nil {
		return s.secret, nil
	}

	secret := &corev1.Secret{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: s.Identity.Spec.SecretName, Namespace: s.Identity.Namespace}, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to get secret %s/%s", s.Identity.Namespace, s.Identity.Spec.SecretName)
	}

	s.secret = secret

	return secret, nil
}

func (s *IdentityScope) getIdentityKey(ctx context.Context) (string, error) {
	secret, err := s.getSecret(ctx)
	if err != nil {
		return "", err
	}

	identityKey, ok := secret.Data[s.Identity.Spec.KeyName]
	if !ok {
		return "", errors.Errorf("key %s not found in secret %s/%s", s.Identity.Spec.KeyName, s.Identity.Namespace, s.Identity.Spec.SecretName)
	}

	return string(identityKey), nil
}

// CheckConnectWithIdentity checks if the identity can be used to connect to YandexCloud
func (s *IdentityScope) CheckConnectWithIdentity(ctx context.Context) error {
	identityKey, err := s.getIdentityKey(ctx)
	if err != nil {
		return err
	}

	// Check if YandexClient can be created
	yandexClient, err := s.yandexClientGetter.GetFromKey(ctx, string(identityKey))
	if err != nil {
		return errors.Wrap(err, "failed to create YandexClient")
	}

	// TODO: Check permissions? Test connection?

	// We don't care about the client close result
	yandexClient.Close(ctx)

	return nil
}

// SetSecretFinalizerAndOwner sets the secret finalizer on the identity secret
func (s *IdentityScope) SetSecretFinalizerAndOwner(ctx context.Context) error {
	identitySecret, err := s.getSecret(ctx)
	if err != nil {
		return err
	}

	originalSecret := identitySecret.DeepCopy()
	if controllerutil.AddFinalizer(identitySecret, s.Identity.GenerateSecretFinalizer()) ||
		!hasOwnerReferenceToObject(identitySecret.GetOwnerReferences(), s.Identity) {

		if err := controllerutil.SetOwnerReference(s.Identity, identitySecret, s.client.Scheme()); err != nil {
			return errors.Wrapf(err, "failed to set owner reference on secret %s/%s", identitySecret.Namespace, identitySecret.Name)
		}

		if err := s.client.Patch(ctx, identitySecret, client.MergeFrom(originalSecret)); err != nil {
			return errors.Wrapf(err, "failed to add finalizer to secret %s/%s", identitySecret.Namespace, identitySecret.Name)
		}
	}

	return nil
}

// RemoveSecretFinalizer removes the finalizer from the secret of the identity.
func (s *IdentityScope) RemoveSecretFinalizer(ctx context.Context) error {
	identitySecret, err := s.getSecret(ctx)
	if err != nil {
		return err
	}

	originalSecret := identitySecret.DeepCopy()
	if controllerutil.RemoveFinalizer(identitySecret, s.Identity.GenerateSecretFinalizer()) {
		// we don't remove owner reference here, because it's not necessary
		if err := s.client.Patch(ctx, identitySecret, client.MergeFrom(originalSecret)); err != nil {
			return errors.Wrapf(err, "failed to remove finalizer from secret %s/%s", identitySecret.Namespace, identitySecret.Name)
		}
	}

	return nil
}

// hasOwnerReferenceToObject checks if the object set as owner reference in the list.
func hasOwnerReferenceToObject(ownerReferences []metav1.OwnerReference, obj client.Object) bool {
	for _, ownerReference := range ownerReferences {
		if ownerReference.UID == obj.GetUID() {
			return true
		}
	}

	return false
}

// UpdateLinkedClusters updates the linked clusters of the identity.
func (s *IdentityScope) UpdateLinkedClusters(ctx context.Context) error {
	clusterList := infrav1.YandexClusterList{}
	if err := s.client.List(ctx, &clusterList, s.generateIdentityLabelSelector()); err != nil {
		return errors.Wrapf(err, "failed to list clusters linked to identity %s", s.Identity.Name)
	}

	linkedClusters := []string{}
	for _, cluster := range clusterList.Items {
		linkedClusters = append(linkedClusters, cluster.Namespace+"/"+cluster.Name)
	}

	s.Identity.Status.LinkedClusters = linkedClusters

	return nil
}

// IsLinkedToCluster checks if the identity is linked to any cluster.
func (s *IdentityScope) IsLinkedToCluster(ctx context.Context) (bool, error) {
	if err := s.UpdateLinkedClusters(ctx); err != nil {
		return false, err
	}

	return len(s.Identity.Status.LinkedClusters) > 0, nil
}

// generateIdentityLabelSelector generates a label selector for the identity.
func (s *IdentityScope) generateIdentityLabelSelector() client.MatchingLabels {
	return client.MatchingLabels(s.Identity.GenerateLabelsForCluster())
}

// PersistIndentityChanges persists the identity changes to the API server.
func (s *IdentityScope) PersistIndentityChanges(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.Identity)
}

// IsSecretChanged checks if the secret of the identity has changed.
func (s *IdentityScope) IsSecretChanged(ctx context.Context) (bool, error) {
	if s.Identity.Status.KeyHash == "" {
		return true, nil
	}

	h, err := s.getIdentityKeyHash(ctx)
	if err != nil {
		return false, err
	}

	return s.Identity.Status.KeyHash != h, nil
}

// SetSecretHash sets the secret hash to the identity status.
func (s *IdentityScope) SetSecretHash(ctx context.Context) error {
	h, err := s.getIdentityKeyHash(ctx)
	if err != nil {
		return err
	}

	s.Identity.Status.KeyHash = h

	return nil
}

// getIdentityKeyHash returns the hash of the identity key in the secret.
func (s *IdentityScope) getIdentityKeyHash(ctx context.Context) (string, error) {
	key, err := s.getIdentityKey(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(key))), nil
}
