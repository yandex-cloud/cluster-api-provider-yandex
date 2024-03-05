package client

import (
	"context"
	"encoding/json"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/loadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

// YandexClient is a wrapper for YandexCloud SDK
type YandexClient struct {
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

// ComputeDelete send compute delete request to Yandex Cloud and return operation status.
func (c *YandexClient) ComputeDelete(ctx context.Context, id string) (bool, error) {
	op, err := c.sdk.Compute().Instance().Delete(ctx, &compute.DeleteInstanceRequest{
		InstanceId: id,
	})

	return op.Done, err
}

// LBAddTarget adds target to YandexCloud NLB TargetGroup.
func (c *YandexClient) LBAddTarget(ctx context.Context, req *loadbalancer.AddTargetsRequest) (*operation.Operation, error) {
	return c.sdk.LoadBalancer().TargetGroup().AddTargets(ctx, req)
}

// LBRemoveTarget removes target from YandexCloud NLB TargetGroup.
func (c *YandexClient) LBRemoveTarget(ctx context.Context, req *loadbalancer.RemoveTargetsRequest) (*operation.Operation, error) {
	return c.sdk.LoadBalancer().TargetGroup().RemoveTargets(ctx, req)
}

// Close shutdowns YandexCloud SDK and closes all open connections.
func (c *YandexClient) Close(ctx context.Context) error {
	return c.sdk.Shutdown(ctx)
}
