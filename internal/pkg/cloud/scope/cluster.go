package scope

import (
	"context"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud"
	"github.com/yandex-cloud/go-sdk/gen/compute"
	"github.com/yandex-cloud/go-sdk/gen/loadbalancer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	CloudFactory  cloud.Factory
	Client        client.Client
	Cluster       *clusterv1.Cluster
	YandexCluster *infrav1.YandexCluster
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(ctx context.Context, params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.YandexCluster == nil {
		return nil, errors.New("failed to generate new scope from nil YandexCluster")
	}
	if params.CloudFactory == nil {
		return nil, errors.New("failed to generate new scope from nil CloudFactory")
	}

	helper, err := patch.NewHelper(params.YandexCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	c, err := params.CloudFactory.NewCloud(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init cloud")
	}

	return &ClusterScope{
		client:        params.Client,
		Cluster:       params.Cluster,
		YandexCluster: params.YandexCluster,
		patchHelper:   helper,
		Cloud:         c,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	client      client.Client
	patchHelper *patch.Helper

	Cluster       *clusterv1.Cluster
	YandexCluster *infrav1.YandexCluster
	Cloud         cloud.Cloud
}

// PatchObject persists the cluster configuration and status.
func (s *ClusterScope) PatchObject() error {
	return s.patchHelper.Patch(context.TODO(), s.YandexCluster)
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close() error {
	return s.PatchObject()
}

// SetReady sets the YandexCluster Ready Status.
func (m *ClusterScope) SetReady() {
	m.YandexCluster.Status.Ready = true
}

// Compute gets client for YandexCloud Compute api.
func (m *ClusterScope) Compute() *compute.Compute {
	return m.Cloud.Compute()
}

// LoadBalancer gets client for YandexCloud LoadBalancer api.
func (m *ClusterScope) LoadBalancer() *loadbalancer.LoadBalancer {
	return m.Cloud.LoadBalancer()
}
