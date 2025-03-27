package loadbalancer

import (
	"context"
	"fmt"
	"reflect"

	alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/services/loadbalancers/builders"
)

const (
	resourceDeleted    bool = true
	resourceNotDeleted bool = false
)

// reconcileALB reconciles the YandexCloud application load balancer
// and its supporting components.
func (s *Service) reconcileALBService(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("reconciling application load balancer instance")

	targetGroupID, err := s.reconcileALBTargetGroup(ctx)
	if err != nil {
		return err
	}

	backendGroupID, err := s.reconcileALBBackendGroup(ctx, targetGroupID)
	if err != nil {
		return err
	}

	return s.reconcileALB(ctx, backendGroupID)
}

// deleteALB deletes the YandexCloud application load balancers
// and its supporting components.
func (s *Service) deleteALBService(ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("deleting application load balancer instance")

	deleted, err := s.deleteApplicationLoadBalancer(ctx)
	if err != nil || !deleted {
		return resourceNotDeleted, err
	}

	deleted, err = s.deleteALBBackendGroup(ctx)
	if err != nil || !deleted {
		return resourceNotDeleted, err
	}

	deleted, err = s.deleteALBTargetGroup(ctx)
	if err != nil || !deleted {
		return resourceNotDeleted, err
	}

	return resourceDeleted, nil
}

// deleteApplicationLoadBalancer deletes the YandexCloud application load balancer.
func (s *Service) deleteApplicationLoadBalancer(ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("deleting application load balancer instance")

	client := s.scope.GetClient()
	lb, err := client.ALBGetByName(ctx, s.scope.GetFolderID(), s.scope.GetLBName())
	if err != nil {
		return resourceNotDeleted, err
	}

	// The load balancer has already been deleted.
	if lb == nil {
		return resourceDeleted, nil
	}

	// Check the load balancer status. If the load balancer is already being deleted, we do nothing.
	// Create load balancer deletion request otherwise.
	status := lb.GetStatus()
	logger.V(1).Info("application load balancer status", "status", status.String())
	if status == alb.LoadBalancer_DELETING || status == alb.LoadBalancer_STOPPING {
		return resourceNotDeleted, nil
	}

	// The load balancer exists and not being deleted at this moment.
	// We are ok to send deleteion request.
	return resourceNotDeleted, client.ALBDelete(ctx, lb.GetId())
}

// deleteALBTargetGroup deletes an ALB target group.
func (s *Service) deleteALBTargetGroup(ctx context.Context) (bool, error) {
	client := s.scope.GetClient()
	name := builders.NewALBTargetGroupBuilder(s.scope.GetLBSpec()).
		WithLBName(s.scope.GetLBName()).
		GetName()

	tg, err := client.ALBTargetGroupGetByName(ctx, s.scope.GetFolderID(), name)
	if err != nil {
		return resourceNotDeleted, err
	}

	if tg == nil {
		return resourceDeleted, nil
	}

	return resourceNotDeleted, client.ALBTargetGroupDelete(ctx, tg.GetId())
}

// deleteALBBackendGroup deletes an ALB target group.
func (s *Service) deleteALBBackendGroup(ctx context.Context) (bool, error) {
	client := s.scope.GetClient()
	name := builders.NewALBBackendGroupBuilder(s.scope.GetLBSpec()).
		WithLBName(s.scope.GetLBName()).
		GetName()

	bg, err := client.ALBBackendGroupGetByName(ctx, s.scope.GetFolderID(), name)
	if err != nil {
		return resourceNotDeleted, err
	}

	if bg == nil {
		return resourceDeleted, nil
	}

	return resourceNotDeleted, client.ALBBackendGroupDelete(ctx, bg.GetId())
}

// ReconcileALBTargetGroup reconciles an ALB target group for kubernetes control plane.
// Returns ID of ALB target group.
func (s *Service) reconcileALBTargetGroup(ctx context.Context) (string, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("reconciling application load balancer target group")

	client := s.scope.GetClient()

	builder := builders.NewALBTargetGroupBuilder(s.scope.GetLBSpec()).
		WithCluster(s.scope.Name()).
		WithLBName(s.scope.GetLBName()).
		WithFolder(s.scope.GetFolderID()).
		WithLabels(s.scope.GetLabels())

	tg, err := client.ALBTargetGroupGetByName(ctx, s.scope.GetFolderID(), builder.GetName())
	if err != nil {
		return "", err
	}

	if tg == nil {
		logger.V(1).Info("creating application load balancer target group")
		req, err := builder.Build()
		if err != nil {
			return "", err
		}

		id, err := client.ALBTargetGroupCreate(ctx, req)
		if err != nil {
			return "", err
		}

		logger.Info("application load balancer target group created", "instance id", id)
		return id, nil
	}

	return tg.Id, nil
}

// reconcileALBBackendGroup reconciles an ALB backend group for kubernetes control plane.
// Returns ID of ALB backend group.
func (s *Service) reconcileALBBackendGroup(ctx context.Context, targetGroupID string) (string, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("reconciling application load balancer backend group")

	client := s.scope.GetClient()

	builder := builders.NewALBBackendGroupBuilder(s.scope.GetLBSpec()).
		WithCluster(s.scope.Name()).
		WithLBName(s.scope.GetLBName()).
		WithFolder(s.scope.GetFolderID()).
		WithTargetGroupID(targetGroupID).
		WithLabels(s.scope.GetLabels())

	bg, err := client.ALBBackendGroupGetByName(ctx, s.scope.GetFolderID(), builder.GetName())
	if err != nil {
		return "", err
	}

	if bg == nil {
		logger.V(1).Info("creating application load balancer backend group")
		req, err := builder.Build()
		if err != nil {
			return "", err
		}

		id, err := client.ALBBackendGroupCreate(ctx, req)
		if err != nil {
			return "", err
		}

		logger.Info("application load balancer backend group created", "instance id", id)
		return id, nil
	}

	return bg.Id, nil
}

// reconcileApplicationLoadBalancer reconciles an ALB for kubernetes control plane.
func (s *Service) reconcileALB(ctx context.Context, backendGroupID string) error {
	logger := log.FromContext(ctx)
	logger.V(1).Info("reconciling application load balancer")

	client := s.scope.GetClient()
	builder := builders.NewALBBuilder(s.scope.GetLBSpec()).
		WithCluster(s.scope.Name()).
		WithName(s.scope.GetLBName()).
		WithFolder(s.scope.GetFolderID()).
		WithBackendGroupID(backendGroupID).
		WithNetworkID(s.scope.GetNetworkID()).
		WithLabels(s.scope.GetLabels())

	lb, err := client.ALBGetByName(ctx, s.scope.GetFolderID(), builder.GetName())
	if err != nil {
		return err
	}
	switch {
	case lb != nil && lb.Status == alb.LoadBalancer_ACTIVE && s.scope.GetLBSpec().Listener.Address != "":
		// TODO: add support for external balancers
		lbAddress, lbPort := s.getInternalAddress(lb), s.getInternalPort(lb)
		// TODO: reconile LB listener address/port in https://github.com/yandex-cloud/cluster-api-provider-yandex/issues/17
		if lbAddress != "" && s.scope.GetLBSpec().Listener.Address != lbAddress {
			return fmt.Errorf(
				"load balancer for the YandexCluster %s has an incorrect address %s, expected %s. "+
					"The cluster has become unrecoverable and should be manually deleted",
				s.scope.Name(), lbAddress,
				s.scope.GetLBSpec().Listener.Address)
		}
		if lbPort != 0 && s.scope.GetLBSpec().Listener.Port != lbPort {
			return fmt.Errorf(
				"load balancer for the YandexCluster %s has an incorrect port %d, expected %d. "+
					"The cluster has become unrecoverable and should be manually deleted",
				s.scope.Name(), lbPort,
				s.scope.GetLBSpec().Listener.Port)
		}
	case lb == nil && s.scope.ControlPlaneEndpoint().IsValid() && s.scope.GetLBSpec().Listener.Address == "":
		// if load balancer is not found and cluster ControlPlaneEndpoint is already populated, then we have to recreate cluster.
		return fmt.Errorf("load balancer for the YandexCluster %s not found, the cluster has become unrecoverable and should be manually deleted",
			s.scope.YandexCluster.Name)

	case lb == nil:
		// if load balancer is not found, create it.
		logger.Info("creating application load balancer. It may take a while, please be patient.")
		req, err := builder.Build()
		if err != nil {
			return err
		}

		id, err := client.ALBCreate(ctx, req)
		if err != nil {
			return err
		}
		logger.Info("application loadbalancer created", "instance id", id)
		conditions.MarkTrue(s.scope.YandexCluster, infrav1.ConditionStatusReady)
		return nil
	}

	return nil
}

func (s *Service) getInternalAddress(lb *alb.LoadBalancer) string {
	return lb.Listeners[0].GetEndpoints()[0].Addresses[0].GetInternalIpv4Address().Address
}

func (s *Service) getInternalPort(lb *alb.LoadBalancer) int32 {
	return int32(lb.Listeners[0].GetEndpoints()[0].Ports[0])
}

// describeALB returns the IP address and port of the application load balancer listener.
func (s *Service) describeALB(ctx context.Context) (infrav1.LoadBalancerStatus, error) {
	lb, err := s.scope.GetClient().ALBGetByName(
		ctx,
		s.scope.GetFolderID(),
		s.scope.GetLBName(),
	)
	if err != nil {
		return infrav1.LoadBalancerStatus{}, err
	}

	// TODO: external address support.
	status := infrav1.LoadBalancerStatus{
		ListenerAddress: s.getInternalAddress(lb),
		ListenerPort:    s.getInternalPort(lb),
	}
	return status, nil
}

// addTargetALB adds the IP address to the load balancer's target group.
func (s *Service) addTargetALB(ctx context.Context, ipAddress, subnetID string) error {
	builder := builders.NewALBTargetGroupBuilder(s.scope.GetLBSpec()).
		WithCluster(s.scope.Name()).
		WithLBName(s.scope.GetLBName()).
		WithIP(ipAddress).
		WithSubnetID(subnetID).
		WithFolder(s.scope.GetFolderID())

	tg, err := s.scope.GetClient().ALBTargetGroupGetByName(ctx, s.scope.GetFolderID(), builder.GetName())
	if err != nil {
		return err
	}

	if tg == nil {
		return nil
	}

	if s.isAddressRegisteredALB(ipAddress, subnetID, tg) {
		return nil
	}

	req := builder.WithTargetGroupID(tg.Id).BuildAddTargetRequest(ipAddress)
	_, err = s.scope.GetClient().ALBAddTarget(ctx, req)
	return err
}

// removeTargetALB removes the IP address from the load balancer's target group.
func (s *Service) removeTargetALB(ctx context.Context, ipAddress, subnetID string) error {
	builder := builders.NewALBTargetGroupBuilder(s.scope.GetLBSpec()).
		WithCluster(s.scope.Name()).
		WithLBName(s.scope.GetLBName()).
		WithIP(ipAddress).
		WithSubnetID(subnetID).
		WithFolder(s.scope.GetFolderID())

	tg, err := s.scope.GetClient().ALBTargetGroupGetByName(ctx, s.scope.GetFolderID(), builder.GetName())
	if err != nil {
		return err
	}

	if tg == nil {
		return nil
	}

	if !s.isAddressRegisteredALB(ipAddress, subnetID, tg) {
		return nil
	}

	req := builder.WithTargetGroupID(tg.Id).BuildRemoveTargetRequest(ipAddress)
	_, err = s.scope.GetClient().ALBRemoveTarget(ctx, req)
	return err
}

// isAddressRegisteredALB checks that the instance address is already registered to the ALB target group.
func (s *Service) isAddressRegisteredALB(addr, subnetID string, tg *alb.TargetGroup) bool {
	for _, target := range tg.Targets {
		ipAddress := &alb.Target_IpAddress{IpAddress: addr}
		if reflect.DeepEqual(target.AddressType, ipAddress) && target.SubnetId == subnetID {
			return true
		}
	}
	return false
}

// isActiveALB returns true when the application load balancer instance have an ACTIVE status.
func (s *Service) isActiveALB(ctx context.Context) (bool, error) {
	lb, err := s.scope.GetClient().ALBGetByName(ctx, s.scope.GetFolderID(), s.scope.GetLBName())
	if err != nil {
		return false, err
	}

	if lb.GetStatus() == alb.LoadBalancer_ACTIVE {
		return true, nil
	}

	return false, nil
}
