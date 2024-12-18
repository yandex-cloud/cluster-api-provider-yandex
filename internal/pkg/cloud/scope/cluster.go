package scope

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)

const (
	maxNameLength = 64
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	Client        client.Client
	Cluster       *clusterv1.Cluster
	Builder       yandex.Builder
	YandexCluster *infrav1.YandexCluster
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
	if params.Builder == nil {
		return nil, errors.New("failed to generate new scope from nil ClientBuilder")
	}

	// Get Yandex Client for cluster
	yandexClient, err := getYandexClient(ctx, params)
	if err != nil {
		return nil, err
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
		yandexClient:  yandexClient,
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
func (c *ClusterScope) PatchObject(ctx context.Context) error {
	return c.patchHelper.Patch(ctx, c.YandexCluster)
}

// Close closes the current scope persisting the cluster configuration and status.
func (c *ClusterScope) Close(ctx context.Context) error {
	// first path the object
	if err := c.PatchObject(ctx); err != nil {
		return err
	}

	// close the client, since we've build it inside the scope
	return c.yandexClient.Close(ctx)
}

// Name returns the CAPI cluster name.
func (c *ClusterScope) Name() string {
	return c.Cluster.GetName()
}

// SetReady sets the YandexCluster Ready Status.
func (c *ClusterScope) SetReady() {
	c.YandexCluster.Status.Ready = true
}

// GetClient gets client for YandexCloud API.
func (c *ClusterScope) GetClient() yandex.Client {
	return c.yandexClient
}

// GetLBType gets type of kubernetes API load balancer.
func (c *ClusterScope) GetLBType() infrav1.LoadBalancerType {
	return c.YandexCluster.Spec.LoadBalancer.Type
}

// GetFolderID gets the cluster Folder ID.
func (c *ClusterScope) GetFolderID() string {
	return c.YandexCluster.Spec.FolderID
}

// GetNetworkID gets the Yandex Network ID.
func (c *ClusterScope) GetNetworkID() string {
	return c.YandexCluster.Spec.NetworkSpec.ID
}

// GetLabels gets the set of cluster tags.
func (c *ClusterScope) GetLabels() infrav1.Labels {
	return c.YandexCluster.Spec.Labels
}

// ControlPlaneEndpoint gets the cluster API endpoit.
func (c *ClusterScope) ControlPlaneEndpoint() clusterv1.APIEndpoint {
	return c.YandexCluster.Spec.ControlPlaneEndpoint
}

// GetLBName returns the load balancer name.
func (c *ClusterScope) GetLBName() string {
	if c.YandexCluster.Spec.LoadBalancer.Name == "" {
		return c.generateName()
	}

	return c.YandexCluster.Spec.LoadBalancer.Name
}

// GetLBSpec returns the load balancer specification.
func (c *ClusterScope) GetLBSpec() infrav1.LoadBalancerSpec {
	return c.YandexCluster.Spec.LoadBalancer
}

// generateName generates a resource name via:
// 1. concatenating the cluster name to the suffix provided
// 2. computing a hash for name, if resulting name length greater than
// maxNameLength characters.
func (c *ClusterScope) generateName() string {
	prefix := c.clearString(c.Name())
	name := fmt.Sprintf("%s-api", prefix)

	if len(name) < maxNameLength {
		return name
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(c.Name())))
	return fmt.Sprintf("cluster-%s-api", hash[:16])
}

// clearString remove all non alphnumeric characters from input.
func (c *ClusterScope) clearString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

// AppendLabels appends labels to the cluster.
func (c *ClusterScope) AppendLabels(l map[string]string) {
	if c.Cluster.Labels == nil {
		c.Cluster.Labels = make(map[string]string)
	}

	for k, v := range l {
		c.Cluster.Labels[k] = v
	}
}

func (c *ClusterScope) UpdateIndentityLabels() {
	if c.YandexCluster.Spec.IdentityRef == nil {
		return
	}

	if c.YandexCluster.Labels == nil {
		c.YandexCluster.Labels = make(map[string]string)
	}

	c.YandexCluster.Labels["yandexidentity/"+c.YandexCluster.Spec.IdentityRef.Namespace] = c.YandexCluster.Spec.IdentityRef.Name
}

// getYandexClient returns Yandex Cloud client.
func getYandexClient(ctx context.Context, params ClusterScopeParams) (yandex.Client, error) {
	if params.YandexCluster.Spec.IdentityRef != nil {
		identity := &infrav1.YandexIdentity{}
		if err := params.Client.Get(ctx, params.YandexCluster.Spec.IdentityRef.NamespacedName(), identity); err != nil {
			return nil, errors.Wrapf(err, "failed to get identity %s/%s",
				params.YandexCluster.Spec.IdentityRef.Namespace, params.YandexCluster.Spec.IdentityRef.Name)
		}

		yc, err := params.Builder.GetClientFromSecret(ctx,
			params.Client, identity.Spec.SecretName, params.YandexCluster.Spec.IdentityRef.Namespace, identity.Spec.KeyName)
		if err == nil {
			return yc, nil
		}
		// no need to return error here, as we can fall back to default client
	}

	// Fall back to default client if no identity is provided
	yc, err := params.Builder.GetDefaultClient(ctx)
	if err != nil {
		return nil, err
	}

	return yc, nil
}
