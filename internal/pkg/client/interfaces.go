package client

import (
	"context"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/loadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

// Compute defines interfaces for YandexCloud Compute operations
type Compute interface {
	ComputeGet(ctx context.Context, id string) (*compute.Instance, error)
	ComputeCreate(ctx context.Context, req *compute.CreateInstanceRequest) (string, error)
	ComputeDelete(ctx context.Context, id string) (bool, error)
}

// LoadBalancer defines interfaces for YandexCloud LoadBalancer operations
type LoadBalancer interface {
	LBAddTarget(ctx context.Context, req *loadbalancer.AddTargetsRequest) (*operation.Operation, error)
	LBRemoveTarget(ctx context.Context, req *loadbalancer.RemoveTargetsRequest) (*operation.Operation, error)
}

// Client defines interfaces for YandexCloud api
type Client interface {
	Compute
	LoadBalancer
	Close(ctx context.Context) error
}
