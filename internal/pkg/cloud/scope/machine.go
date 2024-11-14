package scope

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ProviderIDPrefix will be appended to the beginning of Yandex Cloud resource IDs to form the Kubernetes Provider ID.
	ProviderIDPrefix = "yandex://"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	Client        client.Client
	ClusterGetter cloud.ClusterGetter
	LoadBalancer  cloud.LoadBalancer
	Machine       *clusterv1.Machine
	YandexMachine *infrav1.YandexMachine
}

// NewMachineScope is meant to be called for each reconcile iteration.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.YandexMachine == nil {
		return nil, errors.New("YandexMachine is required when creating a MachineScope")
	}
	if params.LoadBalancer == nil {
		return nil, errors.New("load balancer is required when creating a MachineScope")
	}

	helper, err := patch.NewHelper(params.YandexMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		client:      params.Client,
		patchHelper: helper,

		ClusterGetter: params.ClusterGetter,
		LoadBalancer:  params.LoadBalancer,
		Machine:       params.Machine,
		YandexMachine: params.YandexMachine,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	client      client.Client
	patchHelper *patch.Helper

	ClusterGetter cloud.ClusterGetter
	LoadBalancer  cloud.LoadBalancer
	Machine       *clusterv1.Machine
	YandexMachine *infrav1.YandexMachine
}

// PatchObject persists the cluster configuration and status.
func (m *MachineScope) PatchObject() error {
	return m.patchHelper.Patch(context.TODO(), m.YandexMachine)
}

// Close closes the current scope persisting the cluster configuration and status.
func (m *MachineScope) Close() error {
	return m.PatchObject()
}

// Name returns the YandexMachine name.
func (m *MachineScope) Name() string {
	return m.YandexMachine.Name
}

// Namespace returns the namespace name.
func (m *MachineScope) Namespace() string {
	return m.YandexMachine.Namespace
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// SetReady sets the YandexMachine Ready Status.
func (m *MachineScope) SetReady() {
	m.YandexMachine.Status.Ready = true
}

// GetProviderID returns the YandexMachine providerID from the spec.
func (m *MachineScope) GetProviderID() string {
	if m.YandexMachine.Spec.ProviderID != nil {
		return *m.YandexMachine.Spec.ProviderID
	}
	return ""
}

// SetProviderID sets the YandexMachine providerID in spec.
func (m *MachineScope) SetProviderID(instanceID string) {
	pid := fmt.Sprintf("%s%s", ProviderIDPrefix, instanceID)
	m.YandexMachine.Spec.ProviderID = ptr.To(pid)
}

// SetAddresses sets the addresses field on the YandexMachine.
func (m *MachineScope) SetAddresses(addressList []corev1.NodeAddress) {
	m.YandexMachine.Status.Addresses = addressList
}

// GetAddresses gets the addresses field of the YandexMachine.
func (m *MachineScope) GetAddresses() []corev1.NodeAddress {
	return m.YandexMachine.Status.Addresses
}

// GetInstanceStatus returns the YandexMachine instance status.
func (m *MachineScope) GetInstanceStatus() *infrav1.InstanceStatus {
	return m.YandexMachine.Status.InstanceStatus
}

// SetInstanceStatus sets the YandexMachine instance status.
func (m *MachineScope) SetInstanceStatus(v infrav1.InstanceStatus) {
	m.YandexMachine.Status.InstanceStatus = &v
}

// GetBootstrapData returns the bootstrap data from the secret in the Machine's bootstrap.dataSecretName.
func (m *MachineScope) GetBootstrapData() (string, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: m.Namespace(), Name: *m.Machine.Spec.Bootstrap.DataSecretName}
	if err := m.client.Get(context.TODO(), key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for YandexMachine %s/%s", m.Namespace(), m.Name())
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return string(value), nil
}

// SetFailureMessage sets the YandexMachine status failure message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.YandexMachine.Status.FailureMessage = ptr.To(v.Error())
}

// SetFailureReason sets the YandexMachine status failure reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.YandexMachine.Status.FailureReason = &v
}

// GetClient gets client for YandexCloud api.
func (m *MachineScope) GetClient() yandex.Client {
	return m.ClusterGetter.GetClient()
}

// GetInstanceID returns the Yandex Machine ID by parsing the scope's providerID.
func (m *MachineScope) GetInstanceID() string {
	return parseProviderID(m.GetProviderID())
}

// ParseProviderID parses a string to a Yandex Machine ID, removing the cloud identification prefix.
func parseProviderID(id string) string {
	return strings.TrimPrefix(id, ProviderIDPrefix)
}

// GetInstanceReq returns YandexCloud compute instance creation request.
func (m *MachineScope) GetInstanceReq() (*compute.CreateInstanceRequest, error) {
	bootstrapData, err := m.GetBootstrapData()
	if err != nil {
		return nil, err
	}

	memory, ok := m.YandexMachine.Spec.Resources.Memory.AsInt64()
	if !ok {
		return nil, errors.New("failed to parse instance's memory from yandex machine specification")
	}
	resourcesSpec := &compute.ResourcesSpec{
		Cores:  m.YandexMachine.Spec.Resources.Cores,
		Memory: memory,
	}
	if m.YandexMachine.Spec.Resources.GPUs != nil {
		resourcesSpec.Gpus = *m.YandexMachine.Spec.Resources.GPUs
	}
	if m.YandexMachine.Spec.Resources.CoreFraction != nil {
		resourcesSpec.CoreFraction = *m.YandexMachine.Spec.Resources.CoreFraction
	}

	networkInterfacesSpecs := make([]*compute.NetworkInterfaceSpec, 0)

	for _, networkInterface := range m.YandexMachine.Spec.NetworkInterfaces {
		networkInterfaceSpec := &compute.NetworkInterfaceSpec{
			SubnetId:             networkInterface.SubnetID,
			PrimaryV4AddressSpec: &compute.PrimaryAddressSpec{},
		}
		if networkInterface.HasPublicIP != nil && *networkInterface.HasPublicIP {
			networkInterfaceSpec.PrimaryV4AddressSpec = &compute.PrimaryAddressSpec{
				OneToOneNatSpec: &compute.OneToOneNatSpec{
					IpVersion: compute.IpVersion_IPV4,
				},
			}
		}
		networkInterfacesSpecs = append(networkInterfacesSpecs, networkInterfaceSpec)
	}
	bootDiskSize, ok := m.YandexMachine.Spec.BootDisk.Size.AsInt64()
	if !ok {
		return nil, errors.New("failed to parse instance's boot disk size from yandex machine specification")
	}

	return &compute.CreateInstanceRequest{
		FolderId:   m.ClusterGetter.GetFolderID(),
		Name:       m.YandexMachine.GetName(),
		ZoneId:     *m.YandexMachine.Spec.ZoneID,
		PlatformId: *m.YandexMachine.Spec.PlatformID,
		Metadata: map[string]string{
			"user-data": bootstrapData,
		},
		Labels:        m.getMachineLabels(),
		Hostname:      m.YandexMachine.GetName(),
		ResourcesSpec: resourcesSpec,
		BootDiskSpec: &compute.AttachedDiskSpec{
			AutoDelete: true,
			Disk: &compute.AttachedDiskSpec_DiskSpec_{
				DiskSpec: &compute.AttachedDiskSpec_DiskSpec{
					TypeId: *m.YandexMachine.Spec.BootDisk.TypeID,
					Size:   bootDiskSize,
					Source: &compute.AttachedDiskSpec_DiskSpec_ImageId{
						ImageId: m.YandexMachine.Spec.BootDisk.ImageID,
					},
				},
			},
		},
		NetworkInterfaceSpecs: networkInterfacesSpecs,
	}, nil
}
