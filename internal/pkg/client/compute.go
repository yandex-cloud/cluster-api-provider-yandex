package client

import (
	"context"

	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/metrics"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
)

// ComputeGet returns Yandex Compute Instance by instance ID.
func (c *YandexClient) ComputeGet(ctx context.Context, id string) (*compute.Instance, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelCompute)
	ComputeInstance, err := c.sdk.Compute().Instance().Get(ctx, &compute.GetInstanceRequest{
		InstanceId: id,
	})
	mc.ObserveRequest(err)
	return ComputeInstance, err
}

// ComputeCreate sends compute creation request to Yandex Cloud and returns Compute Instance ID.
func (c *YandexClient) ComputeCreate(ctx context.Context, req *compute.CreateInstanceRequest) (string, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelCompute)
	op, err := c.sdk.Compute().Instance().Create(ctx, req)
	mc.ObserveRequest(err)
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

// ComputeDelete sends compute delete request to Yandex Cloud and returns operation status.
func (c *YandexClient) ComputeDelete(ctx context.Context, id string) error {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelCompute)
	_, err := c.sdk.Compute().Instance().Delete(ctx, &compute.DeleteInstanceRequest{
		InstanceId: id,
	})
	mc.ObserveRequest(err)

	return err
}
