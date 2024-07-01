package cloud

import (
	"context"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	computegrpc "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	corev1 "k8s.io/api/core/v1"

	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
)

// Reconciler is a generic interface used by components offering a type of service.
type Reconciler interface {
	Reconcile(ctx context.Context) error
	Delete(ctx context.Context) (bool, error)
}

// ClusterGetter is an interface which can get cluster information.
type ClusterGetter interface {
	GetClient() yandex.Client
	GetLBType() infrav1.LoadBalancerType
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

// MachineGetter is an interface which can get machine information.
type MachineGetter interface {
	GetClient() yandex.Client
	Name() string
	ControlPlaneTargetGroupID() string
	IsControlPlane() bool
	GetInstanceID() *string
	GetInstanceStatus() infrav1.InstanceStatus
	GetBootstrapData() (string, error)
	GetInstanceReq() (*computegrpc.CreateInstanceRequest, error)
	GetAddresses() []corev1.NodeAddress
}

// MachineSetter is an interface which can set machine information.
type MachineSetter interface {
	SetReady()
	SetNotReady()
	SetProviderID()
	SetAddresses(addressList []corev1.NodeAddress)
	SetInstanceStatus(v infrav1.InstanceStatus)
}

// Machine is an interface which can get and set machine information.
type Machine interface {
	MachineGetter
	MachineSetter
}
