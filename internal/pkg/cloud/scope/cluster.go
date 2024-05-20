package scope

import (
	"context"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client        client.Client
	Cluster       *clusterv1.Cluster
	YandexCluster *infrav1.YandexCluster
	YandexClient  yandex.Client
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(ctx context.Context, params ClusterScopeParams) (*ClusterScope, error) {
	if params.Client == nil {
		return nil, errors.New("failed to generate new scope from nil Client")
	}
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.YandexCluster == nil {
		return nil, errors.New("failed to generate new scope from nil YandexCluster")
	}
	if params.YandexClient == nil {
		return nil, errors.New("failed to generate new scope from nil YandexClient")
	}

	helper, err := patch.NewHelper(params.YandexCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &ClusterScope{
		client:        params.Client,
		Cluster:       params.Cluster,
		YandexCluster: params.YandexCluster,
		patchHelper:   helper,
		yandexClient:  params.YandexClient,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	client      client.Client
	patchHelper *patch.Helper

	Cluster       *clusterv1.Cluster
	YandexCluster *infrav1.YandexCluster
	yandexClient  yandex.Client
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.YandexCluster)
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close(ctx context.Context) error {
	return s.PatchObject(ctx)
}

// SetReady sets the YandexCluster Ready Status.
func (m *ClusterScope) SetReady() {
	m.YandexCluster.Status.Ready = true
}

// GetClient gets client for YandexCloud api.
func (m *ClusterScope) GetClient() yandex.Client {
	return m.yandexClient
}

// GetLBType gets type of kubernetes api loadbalancer.
func (m *ClusterScope) GetLBType() infrav1.LoadBalancerType {
	return m.YandexCluster.Spec.LoadBalancer.Type
}
