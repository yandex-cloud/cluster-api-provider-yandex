package client

import (
	"context"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

// Compute defines interfaces for YandexCloud Compute operations
type Compute interface {
	ComputeGet(ctx context.Context, id string) (*compute.Instance, error)
	ComputeCreate(ctx context.Context, req *compute.CreateInstanceRequest) (string, error)
	ComputeDelete(ctx context.Context, id string) error
}

// LoadBalancer defines interfaces for YandexCloud LoadBalancer operations
type LoadBalancer interface {
	LBAddTarget(ctx context.Context, req any, lbType infrav1.LoadBalancerType) (*operation.Operation, error)
	LBRemoveTarget(ctx context.Context, req any, lbType infrav1.LoadBalancerType) (*operation.Operation, error)
	LBGetTargetGroup(ctx context.Context, targetGroupID string, lbType infrav1.LoadBalancerType) (any, error)
}

// Client defines interfaces for YandexCloud api
type Client interface {
	Compute
	LoadBalancer
	Close(ctx context.Context) error
}
