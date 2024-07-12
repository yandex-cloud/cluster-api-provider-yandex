package scope

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud"
	yandex_alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	yandex_compute "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	yandex_nlb "github.com/yandex-cloud/go-genproto/yandex/cloud/loadbalancer/v1"
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
	ProviderIDPrefix         = "yandex://"
	defaultZoneID     string = "ru-central1-d"
	defaultPlatformID string = "standard-v3"
	defaultDiskTypeID string = "network-ssd"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	Client        client.Client
	ClusterGetter cloud.ClusterGetter
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

	helper, err := patch.NewHelper(params.YandexMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
	return &MachineScope{
		client:      params.Client,
		patchHelper: helper,

		ClusterGetter: params.ClusterGetter,
		Machine:       params.Machine,
		YandexMachine: params.YandexMachine,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	client      client.Client
	patchHelper *patch.Helper

	ClusterGetter cloud.ClusterGetter
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

// Zone returns the YandexCloud availability zone for the YandexMachine.
func (m *MachineScope) Zone() string {
	return *m.YandexMachine.Spec.ZoneID
}

// IsControlPlane returns true if the machine is a control plane.
func (m *MachineScope) IsControlPlane() bool {
	return util.IsControlPlaneMachine(m.Machine)
}

// SetReady sets the YandexMachine Ready Status.
func (m *MachineScope) SetReady() {
	m.YandexMachine.Status.Ready = true
}

// SetNotReady unsets the YandexMachine Ready Status.
func (m *MachineScope) SetNotReady() {
	m.YandexMachine.Status.Ready = false
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
	m.YandexMachine.Spec.ProviderID = ptr.To[string](pid)
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

// ControlPlaneTargetGroupID returns the control-plane target group ID.
func (m *MachineScope) ControlPlaneTargetGroupID() *string {
	return m.YandexMachine.Spec.TargetGroupID
}

// SetFailureMessage sets the YandexMachine status failure message.
func (m *MachineScope) SetFailureMessage(v error) {
	m.YandexMachine.Status.FailureMessage = ptr.To[string](v.Error())
}

// SetFailureReason sets the YandexMachine status failure reason.
func (m *MachineScope) SetFailureReason(v capierrors.MachineStatusError) {
	m.YandexMachine.Status.FailureReason = &v
}

// GetClient gets client for YandexCloud api.
func (m *MachineScope) GetClient() yandex.Client {
	return m.ClusterGetter.GetClient()
}

// HasFailed returns the failure state of the machine scope.
func (m *MachineScope) HasFailed() bool {
	return m.YandexMachine.Status.FailureReason != nil || m.YandexMachine.Status.FailureMessage != nil
}

// GetInstanceID returns the Yandex Machine ID by parsing the scope's providerID.
func (m *MachineScope) GetInstanceID() string {
	return parseProviderID(m.GetProviderID())
}

// ParseProviderID parses a string to a Yandex Machine ID, removing the cloud identification prefix.
func parseProviderID(id string) string {
	return strings.TrimPrefix(id, ProviderIDPrefix)
}

// IsControlPlaneRegistered checks that control plane instance address exists in kube api lb target group.
func (m *MachineScope) IsControlPlaneRegistered(tg any) (bool, error) {
	addresses := m.GetAddresses()

	address := addresses[0].Address
	subnetID := m.YandexMachine.Spec.NetworkInterfaces[0].SubnetID

	switch m.ClusterGetter.GetLBType() {
	case infrav1.ApplicationLoadBalancer:
		targetGroup, ok := tg.(*yandex_alb.TargetGroup)
		if !ok {
			return false, fmt.Errorf("can't convert target group to application loadbalancer TargetGroup")
		}
		for _, target := range targetGroup.Targets {
			ipAddress := &yandex_alb.Target_IpAddress{
				IpAddress: address,
			}
			if reflect.DeepEqual(target.AddressType, ipAddress) && target.SubnetId == subnetID {
				return true, nil
			}
		}
		return false, nil
	case infrav1.NetworkLoadBalancer:
		targetGroup, ok := tg.(*yandex_nlb.TargetGroup)
		if !ok {
			return false, fmt.Errorf("can't convert target group to network loadbalancer TargetGroup")
		}
		for _, target := range targetGroup.Targets {
			if target.Address == address && target.SubnetId == subnetID {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown loadbalancer type: %v", m.ClusterGetter.GetLBType())
	}
}

// GetLBTargets generates list of loadbalancer.Target objects for Yandex Cloud API request.
func (m *MachineScope) GetLBTargets() (any, error) {
	addresses := m.GetAddresses()
	if len(addresses) == 0 {
		return nil, fmt.Errorf("no addresses for controlplane instance registering")
	}
	address := addresses[0].Address
	subnetID := m.YandexMachine.Spec.NetworkInterfaces[0].SubnetID

	switch m.ClusterGetter.GetLBType() {
	case infrav1.ApplicationLoadBalancer:
		target := []*yandex_alb.Target{
			{
				SubnetId: subnetID,
				AddressType: &yandex_alb.Target_IpAddress{
					IpAddress: address,
				},
			},
		}
		return target, nil
	case infrav1.NetworkLoadBalancer:
		target := []*yandex_nlb.Target{
			{
				SubnetId: subnetID,
				Address:  address,
			},
		}
		return target, nil
	default:
		return nil, fmt.Errorf("unknown loadbalancer type: %v", m.ClusterGetter.GetLBType())
	}
}

// GetLBRemoveTargetsRequest generates RemoveTargetsRequest for Yandex Cloud loadbalancer API.
func (m *MachineScope) GetLBRemoveTargetsRequest() (any, error) {
	targets, err := m.GetLBTargets()
	if err != nil {
		return nil, err
	}
	switch m.ClusterGetter.GetLBType() {
	case infrav1.ApplicationLoadBalancer:
		albTargets, ok := targets.([]*yandex_alb.Target)
		if !ok {
			return nil, fmt.Errorf("can't convert targets to application loadbalancer Target")
		}
		request := &yandex_alb.RemoveTargetsRequest{
			TargetGroupId: *m.ControlPlaneTargetGroupID(),
			Targets:       albTargets,
		}
		return request, nil
	case infrav1.NetworkLoadBalancer:
		nlbTargets, ok := targets.([]*yandex_nlb.Target)
		if !ok {
			return nil, fmt.Errorf("can't convert targets to network loadbalancer Target")
		}
		request := &yandex_nlb.RemoveTargetsRequest{
			TargetGroupId: *m.ControlPlaneTargetGroupID(),
			Targets:       nlbTargets,
		}
		return request, nil
	default:
		return nil, fmt.Errorf("unknown loadbalancer type: %v", m.ClusterGetter.GetLBType())
	}
}

// GetLBAddTargetsRequest generates AddTargetsRequest for Yandex Cloud loadbalancer API.
func (m *MachineScope) GetLBAddTargetsRequest() (any, error) {
	targets, err := m.GetLBTargets()
	if err != nil {
		return nil, err
	}
	switch m.ClusterGetter.GetLBType() {
	case infrav1.ApplicationLoadBalancer:
		albTargets, ok := targets.([]*yandex_alb.Target)
		if !ok {
			return nil, fmt.Errorf("can't convert targets to application loadbalancer Target")
		}
		request := &yandex_alb.AddTargetsRequest{
			TargetGroupId: *m.ControlPlaneTargetGroupID(),
			Targets:       albTargets,
		}
		return request, nil
	case infrav1.NetworkLoadBalancer:
		nlbTargets, ok := targets.([]*yandex_nlb.Target)
		if !ok {
			return nil, fmt.Errorf("can't convert targets to network loadbalancer Target")
		}
		request := &yandex_nlb.AddTargetsRequest{
			TargetGroupId: *m.ControlPlaneTargetGroupID(),
			Targets:       nlbTargets,
		}
		return request, nil
	default:
		return nil, fmt.Errorf("unknown loadbalancer type: %v", m.ClusterGetter.GetLBType())
	}
}

// GetCreateInstanceRequest generates CreateInstanceRequest for Yandex Cloud API
func (m *MachineScope) GetCreateInstanceRequest() (*yandex_compute.CreateInstanceRequest, error) {
	bootstrapData, err := m.GetBootstrapData()
	if err != nil {
		return nil, err
	}

	memory, ok := m.YandexMachine.Spec.Resources.Memory.AsInt64()
	if !ok {
		return nil, errors.New("failed to parse instance's memory from yandex machine specification")
	}
	resourcesSpec := &yandex_compute.ResourcesSpec{
		Cores:  m.YandexMachine.Spec.Resources.Cores,
		Memory: memory,
	}
	if m.YandexMachine.Spec.Resources.GPUs != nil {
		resourcesSpec.Gpus = *m.YandexMachine.Spec.Resources.GPUs
	}
	if m.YandexMachine.Spec.Resources.CoreFraction != nil {
		resourcesSpec.CoreFraction = *m.YandexMachine.Spec.Resources.CoreFraction
	}

	platformID := defaultPlatformID
	if m.YandexMachine.Spec.PlatformID != nil {
		platformID = *m.YandexMachine.Spec.PlatformID
	}
	networkInterfacesSpecs := make([]*yandex_compute.NetworkInterfaceSpec, 0)

	for _, networkInterface := range m.YandexMachine.Spec.NetworkInterfaces {
		networkInterfaceSpec := &yandex_compute.NetworkInterfaceSpec{
			SubnetId:             networkInterface.SubnetID,
			PrimaryV4AddressSpec: &yandex_compute.PrimaryAddressSpec{},
		}
		if networkInterface.HasPublicIP != nil && *networkInterface.HasPublicIP {
			networkInterfaceSpec.PrimaryV4AddressSpec = &yandex_compute.PrimaryAddressSpec{
				OneToOneNatSpec: &yandex_compute.OneToOneNatSpec{
					IpVersion: yandex_compute.IpVersion_IPV4,
				},
			}
		}
		networkInterfacesSpecs = append(networkInterfacesSpecs, networkInterfaceSpec)
	}

	zoneID := defaultZoneID
	if m.YandexMachine.Spec.ZoneID != nil {
		zoneID = *m.YandexMachine.Spec.ZoneID
	}

	bootDiskTypeID := defaultDiskTypeID
	if m.YandexMachine.Spec.BootDisk.TypeID != nil {
		bootDiskTypeID = *m.YandexMachine.Spec.BootDisk.TypeID
	}
	bootDiskSize, ok := m.YandexMachine.Spec.BootDisk.Size.AsInt64()
	if !ok {
		return nil, errors.New("failed to parse instance's boot disk size from yandex machine specification")
	}

	return &yandex_compute.CreateInstanceRequest{
		FolderId:   m.YandexMachine.Spec.FolderID,
		Name:       m.YandexMachine.GetName(),
		ZoneId:     zoneID,
		PlatformId: platformID,
		Metadata: map[string]string{
			"user-data": bootstrapData,
		},
		Labels: map[string]string{
			"managed-by": "capy-controller-manager",
			"purpose":    "capy-test",
		},
		Hostname:      m.YandexMachine.GetName(),
		ResourcesSpec: resourcesSpec,
		BootDiskSpec: &yandex_compute.AttachedDiskSpec{
			AutoDelete: true,
			Disk: &yandex_compute.AttachedDiskSpec_DiskSpec_{
				DiskSpec: &yandex_compute.AttachedDiskSpec_DiskSpec{
					TypeId: bootDiskTypeID,
					Size:   bootDiskSize,
					Source: &yandex_compute.AttachedDiskSpec_DiskSpec_ImageId{
						ImageId: m.YandexMachine.Spec.BootDisk.ImageID,
					},
				},
			},
		},
		NetworkInterfaceSpecs: networkInterfacesSpecs,
	}, nil
}
