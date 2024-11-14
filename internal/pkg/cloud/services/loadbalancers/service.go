package loadbalancer

import (
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
)

// Service implements instances reconciler.
type Service struct {
	scope *scope.ClusterScope
}

var _ cloud.Reconciler = &Service{}

// New returns a new load balancer service.
func New(scp *scope.ClusterScope) *Service {
	return &Service{
		scope: scp,
	}
}
