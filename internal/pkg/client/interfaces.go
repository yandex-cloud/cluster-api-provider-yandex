package client

import (
	"context"

	alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	nlb "github.com/yandex-cloud/go-genproto/yandex/cloud/loadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	kube "sigs.k8s.io/controller-runtime/pkg/client"
)

// Compute defines interface for YandexCloud Compute operations.
type Compute interface {
	ComputeGet(ctx context.Context, id string) (*compute.Instance, error)
	ComputeCreate(ctx context.Context, req *compute.CreateInstanceRequest) (string, error)
	ComputeDelete(ctx context.Context, id string) error
}

// ApplicationLoadBalancer defines interface for YandexCloud ALB operations.
type ApplicationLoadBalancer interface {
	ALBAddTarget(ctx context.Context, req *alb.AddTargetsRequest) (*operation.Operation, error)
	ALBGetTargetGroup(ctx context.Context, targetGroupID string) (*alb.TargetGroup, error)
	ALBRemoveTarget(ctx context.Context, req *alb.RemoveTargetsRequest) (*operation.Operation, error)
	ALBTargetGroupCreate(ctx context.Context, req *alb.CreateTargetGroupRequest) (string, error)
	ALBTargetGroupDelete(ctx context.Context, id string) error
	ALBTargetGroupGet(ctx context.Context, id string) (*alb.TargetGroup, error)
	ALBTargetGroupGetByName(ctx context.Context, id, name string) (*alb.TargetGroup, error)
	ALBBackendGroupCreate(ctx context.Context, req *alb.CreateBackendGroupRequest) (string, error)
	ALBBackendGroupDelete(ctx context.Context, id string) error
	ALBBackendGroupGet(ctx context.Context, id string) (*alb.BackendGroup, error)
	ALBBackendGroupGetByName(ctx context.Context, id, name string) (*alb.BackendGroup, error)
	ALBCreate(ctx context.Context, req *alb.CreateLoadBalancerRequest) (string, error)
	ALBDelete(ctx context.Context, id string) error
	ALBGet(ctx context.Context, id string) (*alb.LoadBalancer, error)
	ALBGetByName(ctx context.Context, id, name string) (*alb.LoadBalancer, error)
}

// NetworkLoadBalancer defines interface for YandexCloud NLB operations.
type NetworkLoadBalancer interface {
	NLBAddTarget(ctx context.Context, req *nlb.AddTargetsRequest) (*operation.Operation, error)
	NLBGetTargetGroup(ctx context.Context, targetGroupID string) (*nlb.TargetGroup, error)
	NLBRemoveTarget(ctx context.Context, req *nlb.RemoveTargetsRequest) (*operation.Operation, error)
}

// Client defines interface for YandexCloud API.
type Client interface {
	Compute
	ApplicationLoadBalancer
	NetworkLoadBalancer
	Close(ctx context.Context) error
}

//go:generate mockgen -build_flags=--mod=mod -package mock_client -destination=mock_client/builder.go . Builder

// Builder defines interfaces for YandexClientBuilder
type Builder interface {
	// GetClientFromSecret returns YandexClient with secretName and keyName
	GetClientFromSecret(ctx context.Context, cl kube.Client, secretName string, keyName string) (Client, error)

	// GetDefaultClient returns YandexClient with defaultKey
	GetDefaultClient(ctx context.Context) (Client, error)

	// GetClientFromKey returns YandexClient with key
	GetClientFromKey(ctx context.Context, key string) (Client, error)
}
