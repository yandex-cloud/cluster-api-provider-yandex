package loadbalancer

import (
	"context"
	"fmt"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
)

// Reconcile reconciles the loadbalancer instance.
func (s *Service) Reconcile(ctx context.Context) error {
	lbType := s.scope.GetLBType()
	switch lbType {
	case infrav1.LoadBalancerTypeALB:
		return s.reconcileALBService(ctx)
	case infrav1.LoadBalancerTypeNLB:
		return s.reconcileNLBService(ctx)
	default:
		return fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// Delete deletes the loadbalancer instance.
func (s *Service) Delete(ctx context.Context) (bool, error) {
	lbType := s.scope.GetLBType()
	switch lbType {
	case infrav1.LoadBalancerTypeALB:
		return s.deleteALBService(ctx)
	case infrav1.LoadBalancerTypeNLB:
		return s.deleteNLBService(ctx)
	default:
		return false, fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// Describe returns the IP address and port of the load balancer listener.
func (s *Service) Describe(ctx context.Context) (infrav1.LoadBalancerStatus, error) {
	lbType := s.scope.GetLBType()
	switch lbType {
	case infrav1.LoadBalancerTypeALB:
		return s.describeALB(ctx)
	case infrav1.LoadBalancerTypeNLB:
		return s.describeNLB(ctx)
	default:
		return infrav1.LoadBalancerStatus{}, fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// Status returns the load balancer status.
func (s *Service) Status(ctx context.Context) (infrav1.LBStatus, error) {
	lbType := s.scope.GetLBType()
	switch lbType {
	case infrav1.LoadBalancerTypeALB:
		return s.getStatusALB(ctx)
	case infrav1.LoadBalancerTypeNLB:
		return s.getStatusNLB(ctx)
	default:
		return infrav1.LoadBalancerOther, fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// AddTarget adds address to the load balancer target group.
func (s *Service) AddTarget(ctx context.Context, addr, subnetID string) error {
	lbType := s.scope.GetLBType()
	switch lbType {
	case infrav1.LoadBalancerTypeALB:
		return s.addTargetALB(ctx, addr, subnetID)
	case infrav1.LoadBalancerTypeNLB:
		return s.addTargetNLB(ctx, addr, subnetID)
	default:
		return fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// RemoveTarget removes address from the load balancer target group.
func (s *Service) RemoveTarget(ctx context.Context, addr, subnetID string) error {
	lbType := s.scope.GetLBType()
	switch lbType {
	case infrav1.LoadBalancerTypeALB:
		return s.removeTargetALB(ctx, addr, subnetID)
	case infrav1.LoadBalancerTypeNLB:
		return s.removeTargetNLB(ctx, addr, subnetID)
	default:
		return fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}
