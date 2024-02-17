package scope

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud"
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
	ProviderIDPrefix = "yandexcloud://"
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
		return nil, errors.New("client is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("machine is required when creating a MachineScope")
	}
	if params.YandexMachine == nil {
		return nil, errors.New("yandexmachine is required when creating a MachineScope")
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
	if m.Machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

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
