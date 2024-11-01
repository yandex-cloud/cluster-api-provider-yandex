package loadbalancer

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
)

// reconcileNLBService reconciles the YandexCloud network load balancer
// and its supporting components.
func (s *Service) reconcileNLBService(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("reconciling network loadbalancer instance")

	return fmt.Errorf("NLB support will be added in future releases, use ALB instead")
}

// deleteNLBService deletes the YandexCloud network load balancer
// and its supporting components.
func (s *Service) deleteNLBService(ctx context.Context) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("deleting network loadbalancer instance")

	return false, fmt.Errorf("NLB support will be added in future releases, use ALB instead")
}

// describeNLB returns the IP address and port of the network load balancer listener.
func (s *Service) describeNLB(ctx context.Context) (infrav1.LoadBalancerStatus, error) {
	_ = ctx
	return infrav1.LoadBalancerStatus{}, fmt.Errorf("NLB support will be added in future releases, use ALB instead")
}

// addTargetNLB adds the IP address to the network load balancer's target group.
func (s *Service) addTargetNLB(ctx context.Context, addr, subnetID string) error {
	//nolint:dogsled // placeholder
	_, _, _ = ctx, addr, subnetID
	return fmt.Errorf("NLB support will be added in future releases, use ALB instead")
}

// removeTargetNLB removes the IP address from the network load balancer's target group.
func (s *Service) removeTargetNLB(ctx context.Context, addr, subnetID string) error {
	//nolint:dogsled // placeholder
	_, _, _ = ctx, addr, subnetID
	return fmt.Errorf("NLB support will be added in future releases, use ALB instead")
}

// isActiveNLB returns true when the network load balancer instance have an ACTIVE status.
func (s *Service) isActiveNLB(ctx context.Context) (bool, error) {
	//nolint:dogsled // placeholder
	_ = ctx
	return false, fmt.Errorf("NLB support will be added in future releases, use ALB instead")
}
