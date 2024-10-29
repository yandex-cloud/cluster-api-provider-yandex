package client

//go:generate mockgen -build_flags=--mod=mod -package mock_client -destination=mock_client/client.go . Client

import (
	"context"
	"encoding/json"
	"fmt"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/loadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

// YandexClient is a wrapper for YandexCloud SDK.
// Implements Client interface.
type YandexClient struct {
	Compute
	LoadBalancer
	sdk *ycsdk.SDK
}

// GetClient returns authentificated YandexClient.
func GetClient(ctx context.Context, key string) (Client, error) {
	var SAKey iamkey.Key

	if err := json.Unmarshal([]byte(key), &SAKey); err != nil {
		return nil, err
	}

	credentials, err := ycsdk.ServiceAccountKey(&SAKey)
	if err != nil {
		return nil, err
	}

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{Credentials: credentials})
	if err != nil {
		return nil, err
	}

	return &YandexClient{sdk: sdk}, nil
}

// ComputeGet returns YandexCloud Compute Instance by instance ID.
func (c *YandexClient) ComputeGet(ctx context.Context, id string) (*compute.Instance, error) {
	return c.sdk.Compute().Instance().Get(ctx, &compute.GetInstanceRequest{
		InstanceId: id,
	})
}

// ComputeCreate send compute creation request to YandexCloud and return Compute Instance ID.
func (c *YandexClient) ComputeCreate(ctx context.Context, req *compute.CreateInstanceRequest) (string, error) {
	op, err := c.sdk.Compute().Instance().Create(ctx, req)
	if err != nil {
		return "", err
	}

	ci, err := c.sdk.WrapOperation(op, err)
	if err != nil {
		return "", err
	}

	meta, err := ci.Metadata()
	if err != nil {
		return "", err
	}

	return meta.(*compute.CreateInstanceMetadata).InstanceId, nil
}

// ComputeDelete sends compute delete request to YandexCloud and return operation status.
func (c *YandexClient) ComputeDelete(ctx context.Context, id string) error {
	_, err := c.sdk.Compute().Instance().Delete(ctx, &compute.DeleteInstanceRequest{
		InstanceId: id,
	})

	return err
}

// LBAddTarget adds target to Yandex LB TargetGroup.
func (c *YandexClient) LBAddTarget(ctx context.Context, req any, lbType infrav1.LoadBalancerType) (*operation.Operation, error) {
	switch lbType {
	case infrav1.ApplicationLoadBalancer:
		request, ok := req.(*apploadbalancer.AddTargetsRequest)
		if !ok {
			return nil, fmt.Errorf("can't convert request to application loadbalancer AddTargetRequest")
		}
		return c.ALBAddTarget(ctx, request)

	case infrav1.NetworkLoadBalancer:
		request, ok := req.(*loadbalancer.AddTargetsRequest)
		if !ok {
			return nil, fmt.Errorf("can't convert request to network loadbalancer AddTargetRequest")
		}
		return c.NLBAddTarget(ctx, request)
	default:
		return nil, fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// LBRemoveTarget removes target from Yandex LB TargetGroup.
func (c *YandexClient) LBRemoveTarget(ctx context.Context, req any, lbType infrav1.LoadBalancerType) (*operation.Operation, error) {
	switch lbType {
	case infrav1.ApplicationLoadBalancer:
		request, ok := req.(*apploadbalancer.RemoveTargetsRequest)
		if !ok {
			return nil, fmt.Errorf("can't convert request to application loadbalancer RemoveTargetRequest")
		}
		return c.ALBRemoveTarget(ctx, request)

	case infrav1.NetworkLoadBalancer:
		request, ok := req.(*loadbalancer.RemoveTargetsRequest)
		if !ok {
			return nil, fmt.Errorf("can't convert request to network loadbalancer RemoveTargetRequest")
		}
		return c.NLBRemoveTarget(ctx, request)
	default:
		return nil, fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// LBGetTargetGroup gets target group from Yandex.
func (c *YandexClient) LBGetTargetGroup(ctx context.Context, targetGroupID string, lbType infrav1.LoadBalancerType) (any, error) {
	switch lbType {
	case infrav1.ApplicationLoadBalancer:
		return c.ALBGetTargetGroup(ctx, targetGroupID)
	case infrav1.NetworkLoadBalancer:
		return c.NLBGetTargetGroup(ctx, targetGroupID)
	default:
		return nil, fmt.Errorf("unknown loadbalancer type: %v", lbType)
	}
}

// Close shutdowns YandexCloud SDK and closes all open connections.
func (c *YandexClient) Close(ctx context.Context) error {
	return c.sdk.Shutdown(ctx)
}
