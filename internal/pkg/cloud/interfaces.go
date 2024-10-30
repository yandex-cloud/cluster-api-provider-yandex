package cloud

import (
	"context"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
)

// Reconciler is a generic interface used by components offering a type of service.
type Reconciler interface {
	Reconcile(ctx context.Context) error
	Delete(ctx context.Context) (bool, error)
}

// LoadBalancerSetter is an interface which can add and remove client to/from load balancer target group.
type LoadBalancerSetter interface {
	AddTarget(ctx context.Context, addr, subnetID string) error
	RemoveTarget(ctx context.Context, addr, subnetID string) error
}

// LoadBalancerGetter is an interface which can get load balancer information.
type LoadBalancerGetter interface {
	Describe(ctx context.Context) (infrav1.LoadBalancerStatus, error)
	Status(ctx context.Context) (infrav1.LBStatus, error)
}

// LoadBalancer is an interface which provides ALB/NLB management.
type LoadBalancer interface {
	LoadBalancerGetter
	LoadBalancerSetter
}

// ClusterGetter is an interface which can get cluster information.
type ClusterGetter interface {
	GetNetworkID() string
	GetAdditionalLabels() infrav1.Labels
	GetClient() yandex.Client
	GetLBType() infrav1.LoadBalancerType
	GetLBSpec() infrav1.LoadBalancerSpec
	GetLBName() string
	GetFolderID() string
}

// ClusterSetter is an interface which can set cluster information.
type ClusterSetter interface {
	SetReady()
}

// Cluster is an interface which can get and set cluster information.
type Cluster interface {
	ClusterGetter
	ClusterSetter
}
