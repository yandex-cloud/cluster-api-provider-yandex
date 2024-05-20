package compute

import (
	"context"
	"fmt"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	yandex_compute "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	instanceDeleted    bool = true
	instanceNotDeleted bool = false
)

// Reconcile reconciles compute instance.
func (s *Service) Reconcile(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("reconciling compute instance")

	client := s.scope.GetClient()

	// If InstanceID does not set up, we have to create new instance.
	instanceID := s.scope.GetInstanceID()
	if instanceID == "" {
		newInstanceID, err := s.createComputeInstance(ctx, client)
		if err != nil {
			conditions.MarkFalse(s.scope.YandexMachine,
				infrav1.ConditionStatusRunning, infrav1.ConditionStatusNotfound, clusterv1.ConditionSeverityError, err.Error())
			return err
		}

		s.scope.SetProviderID(newInstanceID)
		logger.Info("instance creating")
	}

	// Find compute instance and set status.
	vm, err := client.ComputeGet(ctx, s.scope.GetInstanceID())
	if err != nil {
		conditions.MarkUnknown(s.scope.YandexMachine,
			infrav1.ConditionStatusProvisioning,
			infrav1.ConditionStatusNotfound,
			err.Error())
		conditions.MarkUnknown(s.scope.YandexMachine,
			infrav1.ConditionStatusRunning,
			infrav1.ConditionStatusNotfound,
			err.Error())
		return fmt.Errorf("unable to find instance %v: %w", s.scope.GetInstanceID(), err)
	}

	instanceState := infrav1.InstanceStatus(vm.GetStatus().String())
	if instanceState == infrav1.InstanceStatusRunning {
		instanceAddress, err := s.getInstanceAddress(vm)
		if err != nil {
			conditions.MarkFalse(s.scope.YandexMachine,
				infrav1.ConditionStatusRunning,
				infrav1.ConditionStatusError,
				clusterv1.ConditionSeverityError,
				err.Error())
			return err
		}

		s.scope.SetAddresses(instanceAddress)

		if s.scope.IsControlPlane() {
			logger.V(1).Info("registering control plane instance")
			if err := s.registerControlPlane(ctx, client); err != nil {
				return fmt.Errorf("failed to register control plane: %w", err)
			}
		}
		conditions.MarkTrue(s.scope.YandexMachine, infrav1.ConditionStatusRunning)
	}

	s.scope.SetInstanceStatus(
		infrav1.InstanceStatus(
			vm.GetStatus().String(),
		),
	)

	return nil
}

// Delete deletes a compute instance.
func (s *Service) Delete(ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("deleting instance")
	client := s.scope.GetClient()

	// TODO: bootstap secret deletion handling.
	instanceID := s.scope.GetInstanceID()
	// instance never been created, mark deleted and return.
	if instanceID == "" {
		return instanceDeleted, nil
	}

	// Find compute instance and set status.
	vm, err := client.ComputeGet(ctx, s.scope.GetInstanceID())
	if err != nil {
		// instance already deleted or deleted by someone else.
		logger.Info("unable to find instance")
		if s.scope.IsControlPlane() {
			logger.V(1).Info("deregistering control plane instance")
			if err := s.deregisterControlPlane(ctx, client); err != nil {
				return instanceNotDeleted, fmt.Errorf("failed to deregister control plane: %w", err)
			}
		}
		return instanceDeleted, nil
	}

	// Check the instance status. If instance already shutting down or terminated,
	// do nothing. Otherwise attempt to delete compute instance.
	instanceStatus := infrav1.InstanceStatus(vm.GetStatus().String())
	s.scope.SetInstanceStatus(instanceStatus)

	if instanceStatus == infrav1.InstanceStatusDeleting {
		return instanceNotDeleted, nil
	}

	if s.scope.IsControlPlane() {
		logger.V(1).Info("deregistering control plane instance")
		if err := s.deregisterControlPlane(ctx, client); err != nil {
			return instanceNotDeleted, fmt.Errorf("failed to deregister control plane: %w", err)
		}
	}

	return instanceNotDeleted, client.ComputeDelete(ctx, instanceID)
}

// createComputeInstance creates a virtual machine from YandexCompute specification.
func (s *Service) createComputeInstance(ctx context.Context, client yandex.Client) (string, error) {
	request, err := s.scope.GetCreateInstanceRequest()
	if err != nil {
		return "", err
	}
	return client.ComputeCreate(ctx, request)
}

// getInstanceAddress returns the internal IP address of the instance.
func (s *Service) getInstanceAddress(instance *yandex_compute.Instance) ([]corev1.NodeAddress, error) {
	intfList := instance.GetNetworkInterfaces()
	// TODO: MVP support only one interface with IPv4 Address.
	if len(intfList) != 1 {
		return nil, fmt.Errorf("only one interface supported")
	}

	address := intfList[0].GetPrimaryV4Address().GetAddress()
	if address == "" {
		return nil, fmt.Errorf("only IPv4 address supported")
	}

	return []corev1.NodeAddress{{
		Type:    corev1.NodeInternalIP,
		Address: address,
	}}, nil
}

// registerControlPlane adds control plane instance address to kube api lb target group.
func (s *Service) registerControlPlane(ctx context.Context, client yandex.Client) error {
	targetGroupID := *s.scope.ControlPlaneTargetGroupID()
	lbType := s.scope.ClusterGetter.GetLBType()

	request, err := s.scope.GetLBAddTargetsRequest()
	if err != nil {
		return err
	}

	tg, err := client.LBGetTargetGroup(ctx, targetGroupID, lbType)
	if err != nil {
		return err
	}

	registered, err := s.scope.IsControlPlaneRegistered(tg)
	if err != nil {
		return err
	}
	if registered {
		return nil
	}

	_, err = client.LBAddTarget(ctx, request, lbType)
	return err
}

// deregisterControlPlane removes control plane instance address from kube api lb target group.
func (s *Service) deregisterControlPlane(ctx context.Context, client yandex.Client) error {
	targetGroupID := *s.scope.ControlPlaneTargetGroupID()
	lbType := s.scope.ClusterGetter.GetLBType()

	request, err := s.scope.GetLBRemoveTargetsRequest()
	if err != nil {
		return err
	}

	tg, err := client.LBGetTargetGroup(ctx, targetGroupID, lbType)
	if err != nil {
		return err
	}

	registered, err := s.scope.IsControlPlaneRegistered(tg)
	if err != nil {
		return err
	}
	if !registered {
		return nil
	}

	_, err = client.LBRemoveTarget(ctx, request, lbType)
	return err
}
