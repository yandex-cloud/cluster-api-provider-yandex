package client

import (
	"context"

	nlb "github.com/yandex-cloud/go-genproto/yandex/cloud/loadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
)

// NLBAddTarget sends AddTargetsRequest to Yandex NLB TargetGroup.
func (c *YandexClient) NLBAddTarget(ctx context.Context, req *nlb.AddTargetsRequest) (*operation.Operation, error) {
	result, err := c.sdk.LoadBalancer().TargetGroup().AddTargets(ctx, req)
	return result, err
}

// NLBRemoveTarget sends RemoveTargetsRequest to Yandex NLB TargetGroup.
func (c *YandexClient) NLBRemoveTarget(ctx context.Context, req *nlb.RemoveTargetsRequest) (*operation.Operation, error) {
	result, err := c.sdk.LoadBalancer().TargetGroup().RemoveTargets(ctx, req)
	return result, err
}

// NLBGetTargetGroup returns TargetGroup from Yandex NLB.
func (c *YandexClient) NLBGetTargetGroup(ctx context.Context, targetGroupID string) (*nlb.TargetGroup, error) {
	req := &nlb.GetTargetGroupRequest{
		TargetGroupId: targetGroupID,
	}
	result, err := c.sdk.LoadBalancer().TargetGroup().Get(ctx, req)
	return result, err
}
