package client

import (
	"context"
	"fmt"

	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/metrics"
	alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	"github.com/yandex-cloud/go-sdk/sdkresolvers"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ALBAddTarget sends AddTargetsRequest to Yandex ALB TargetGroup.
func (c *YandexClient) ALBAddTarget(ctx context.Context, req *alb.AddTargetsRequest) (*operation.Operation, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	result, err := c.sdk.ApplicationLoadBalancer().TargetGroup().AddTargets(ctx, req)
	mc.ObserveRequest(err)
	return result, err
}

// ALBRemoveTarget sends RemoveTargetsRequest to Yandex ALB TargetGroup.
func (c *YandexClient) ALBRemoveTarget(ctx context.Context, req *alb.RemoveTargetsRequest) (*operation.Operation, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	result, err := c.sdk.ApplicationLoadBalancer().TargetGroup().RemoveTargets(ctx, req)
	mc.ObserveRequest(err)
	return result, err
}

// ALBGetTargetGroup returns TargetGroup from Yandex ALB.
func (c *YandexClient) ALBGetTargetGroup(ctx context.Context, targetGroupID string) (*alb.TargetGroup, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	req := &alb.GetTargetGroupRequest{
		TargetGroupId: targetGroupID,
	}
	result, err := c.sdk.ApplicationLoadBalancer().TargetGroup().Get(ctx, req)
	mc.ObserveRequest(err)
	return result, err
}

// ALBTargetGroupCreate sends ALB TargetGroup creation request to Yandex Cloud and returns TargetGroup Instance ID.
func (c *YandexClient) ALBTargetGroupCreate(ctx context.Context, req *alb.CreateTargetGroupRequest) (string, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	op, err := c.sdk.ApplicationLoadBalancer().TargetGroup().Create(ctx, req)
	mc.ObserveRequest(err)
	if err != nil {
		return "", err
	}

	meta, err := c.getMeta(op)
	if err != nil {
		return "", err
	}

	tgmeta, ok := meta.(*alb.CreateTargetGroupMetadata)
	if !ok {
		return "", fmt.Errorf("could not get application loadbalancer TargetGroup metatdata from operation response")
	}

	return tgmeta.GetTargetGroupId(), nil
}

// ALBTargetGroupDelete sends ALB TargetGroup deletion request to Yandex Cloud and returns operation result.
func (c *YandexClient) ALBTargetGroupDelete(ctx context.Context, id string) error {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	request := &alb.DeleteTargetGroupRequest{
		TargetGroupId: id,
	}

	_, err := c.sdk.ApplicationLoadBalancer().TargetGroup().Delete(ctx, request)
	mc.ObserveRequest(err)
	return err
}

// ALBTargetGroupGet returns ALB TargetGroup instance by instance ID.
func (c *YandexClient) ALBTargetGroupGet(ctx context.Context, id string) (*alb.TargetGroup, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	result, err := c.sdk.ApplicationLoadBalancer().TargetGroup().Get(ctx, &alb.GetTargetGroupRequest{
		TargetGroupId: id,
	})
	mc.ObserveRequest(err)
	return result, err
}

// ALBTargetGroupGetByName returns ALB TargetGroup instance by name for the  specified Folder ID.
func (c *YandexClient) ALBTargetGroupGetByName(ctx context.Context, id, name string) (*alb.TargetGroup, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbTargetGroup)
	resp, err := c.sdk.ApplicationLoadBalancer().TargetGroup().List(ctx, &alb.ListTargetGroupsRequest{
		FolderId: id,
		Filter:   sdkresolvers.CreateResolverFilter("name", name),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	mc.ObserveRequest(err)
	if err != nil {
		return nil, err
	}
	if len(resp.TargetGroups) == 0 {
		return nil, nil
	}
	return resp.TargetGroups[0], nil
}

// ALBBackendGroupCreate sends ALB BackendGroup creation request to Yandex Cloud and returns BackendGroup Instance ID.
func (c *YandexClient) ALBBackendGroupCreate(ctx context.Context, req *alb.CreateBackendGroupRequest) (string, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbBackendGroup)
	op, err := c.sdk.ApplicationLoadBalancer().BackendGroup().Create(ctx, req)
	mc.ObserveRequest(err)
	if err != nil {
		return "", err
	}

	meta, err := c.getMeta(op)
	if err != nil {
		return "", err
	}

	bgmeta, ok := meta.(*alb.CreateBackendGroupMetadata)
	if !ok {
		return "", fmt.Errorf("could not get application loadbalancer BackendGroup metatdata from operation response")
	}

	return bgmeta.GetBackendGroupId(), nil
}

// ALBBackendGroupDelete sends ALB BackendGroup deletion request to Yandex Cloud and returns operation result.
func (c *YandexClient) ALBBackendGroupDelete(ctx context.Context, id string) error {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbBackendGroup)
	request := &alb.DeleteBackendGroupRequest{
		BackendGroupId: id,
	}

	_, err := c.sdk.ApplicationLoadBalancer().BackendGroup().Delete(ctx, request)
	mc.ObserveRequest(err)
	return err
}

// ALBBackendGroupGet returns ALB BackendGroup instance by instance ID.
func (c *YandexClient) ALBBackendGroupGet(ctx context.Context, id string) (*alb.BackendGroup, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbBackendGroup)
	result, err := c.sdk.ApplicationLoadBalancer().BackendGroup().Get(ctx, &alb.GetBackendGroupRequest{
		BackendGroupId: id,
	})
	mc.ObserveRequest(err)
	return result, err
}

// ALBBackendGroupGetByName returns ALB BackendGroup instance by name for the specified Folder ID.
func (c *YandexClient) ALBBackendGroupGetByName(ctx context.Context, id, name string) (*alb.BackendGroup, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlbBackendGroup)
	resp, err := c.sdk.ApplicationLoadBalancer().BackendGroup().List(ctx, &alb.ListBackendGroupsRequest{
		FolderId: id,
		Filter:   sdkresolvers.CreateResolverFilter("name", name),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	mc.ObserveRequest(err)
	if err != nil {
		return nil, err
	}
	if len(resp.BackendGroups) == 0 {
		return nil, nil
	}
	return resp.BackendGroups[0], nil
}

// ALBCreate sends ALB creation request to Yandex Cloud and returns ALB ID.
func (c *YandexClient) ALBCreate(ctx context.Context, req *alb.CreateLoadBalancerRequest) (string, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlb)
	op, err := c.sdk.ApplicationLoadBalancer().LoadBalancer().Create(ctx, req)
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

	md, ok := meta.(*alb.CreateLoadBalancerMetadata)
	if !ok {
		return "", fmt.Errorf("could not get application load balancer ID from create operation metadata")
	}

	// We have to wait until ALB will be created and operational.
	id := md.GetLoadBalancerId()
	if err := ci.Wait(ctx); err != nil {
		return "", err
	}

	if _, err := ci.Response(); err != nil {
		return "", err
	}

	return id, nil
}

// ALBDelete sends ALB deletion request to Yandex Cloud and returns operation result.
func (c *YandexClient) ALBDelete(ctx context.Context, id string) error {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlb)
	request := &alb.DeleteLoadBalancerRequest{
		LoadBalancerId: id,
	}

	_, err := c.sdk.ApplicationLoadBalancer().LoadBalancer().Delete(ctx, request)
	mc.ObserveRequest(err)
	return err
}

// ALBGet returns ALB instance by instance ID.
func (c *YandexClient) ALBGet(ctx context.Context, id string) (*alb.LoadBalancer, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlb)
	result, err := c.sdk.ApplicationLoadBalancer().LoadBalancer().Get(ctx, &alb.GetLoadBalancerRequest{
		LoadBalancerId: id,
	})
	mc.ObserveRequest(err)
	return result, err
}

// ALBGetByName returns ALB instance by name for the specified Folder ID.
func (c *YandexClient) ALBGetByName(ctx context.Context, id, name string) (*alb.LoadBalancer, error) {
	mc := metrics.NewMetricContext(metrics.ControllerLabelMachine, metrics.ServiceLabelAlb)
	resp, err := c.sdk.ApplicationLoadBalancer().LoadBalancer().List(ctx, &alb.ListLoadBalancersRequest{
		FolderId: id,
		Filter:   sdkresolvers.CreateResolverFilter("name", name),
		PageSize: sdkresolvers.DefaultResolverPageSize,
	})
	mc.ObserveRequest(err)
	if err != nil {
		return nil, err
	}
	if len(resp.LoadBalancers) == 0 {
		return nil, nil
	}
	return resp.LoadBalancers[0], nil
}

// getMeta returns metadata message from operation.
func (c *YandexClient) getMeta(op *operation.Operation) (protoreflect.ProtoMessage, error) {
	wo, err := c.sdk.WrapOperation(op, nil)
	if err != nil {
		return nil, err
	}

	meta, err := wo.Metadata()
	if err != nil {
		return nil, err
	}

	return meta, nil
}
