package loadbalancers

import (
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
)

// Service implements instances reconciler.
type Service struct {
	scope *scope.MachineScope
}

var _ cloud.Reconciler = &Service{}

// New returns a new service given the YandexCloud api client.
func New(scp *scope.MachineScope) *Service {
	return &Service{
		scope: scp,
	}
}
