package builders

import (
	"time"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	alb "github.com/yandex-cloud/go-genproto/yandex/cloud/apploadbalancer/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	describePrefix = "k8s api "
)

// ALBTargetGroupBuilder defines a builder for an application load balancer target group request.
type ALBTargetGroupBuilder struct {
	lbs              infrav1.LoadBalancerSpec
	folderID         string
	clusterName      string
	name             string
	subnetID         string
	targetGroupID    string
	ipAddress        string
	additionalLabels infrav1.Labels
}

// ALBBackendGroupBuilder defines a builder for an application load balancer backend group request.
type ALBBackendGroupBuilder struct {
	lbs              infrav1.LoadBalancerSpec
	folderID         string
	clusterName      string
	name             string
	targetGroupID    string
	additionalLabels infrav1.Labels
}

// ALBBuilder defines a builder for an application load balancer request.
type ALBBuilder struct {
	lbs              infrav1.LoadBalancerSpec
	folderID         string
	networkID        string
	clusterName      string
	name             string
	backendGroupID   string
	additionalLabels infrav1.Labels
}

// NewALBTargetGroupBuilder returns the new ALBTargetGroupBuilder.
func NewALBTargetGroupBuilder(lbs infrav1.LoadBalancerSpec) *ALBTargetGroupBuilder {
	return &ALBTargetGroupBuilder{lbs: lbs}
}

// WithCluster sets the CAPI cluster name.
func (a *ALBTargetGroupBuilder) WithCluster(clusterName string) *ALBTargetGroupBuilder {
	a.clusterName = clusterName
	return a
}

// WithFolder sets the YandexCloud FolderID.
func (a *ALBTargetGroupBuilder) WithFolder(folderID string) *ALBTargetGroupBuilder {
	a.folderID = folderID
	return a
}

// WithLBName sets the TargetGroup name.
func (a *ALBTargetGroupBuilder) WithLBName(name string) *ALBTargetGroupBuilder {
	a.name = name
	return a
}

// WithLabels sets an additional set of tags on TargetGroup.
func (a *ALBTargetGroupBuilder) WithLabels(labels infrav1.Labels) *ALBTargetGroupBuilder {
	a.additionalLabels = labels
	return a
}

// WithSubnetID sets the YandexCloud SubnetID.
func (a *ALBTargetGroupBuilder) WithSubnetID(id string) *ALBTargetGroupBuilder {
	a.subnetID = id
	return a
}

// WithTargetGroupID sets the YandexCloud TargetGroupID.
func (a *ALBTargetGroupBuilder) WithTargetGroupID(id string) *ALBTargetGroupBuilder {
	a.targetGroupID = id
	return a
}

// WithIP sets the BackendGroup IP address.
func (a *ALBTargetGroupBuilder) WithIP(ipAddress string) *ALBTargetGroupBuilder {
	a.ipAddress = ipAddress
	return a
}

// Build  prepares and returns the ALB target group creation request.
func (a *ALBTargetGroupBuilder) Build() (*alb.CreateTargetGroupRequest, error) {
	request := &alb.CreateTargetGroupRequest{
		FolderId:    a.folderID,
		Name:        a.name,
		Description: describePrefix + a.clusterName + " target",
	}

	if a.additionalLabels != nil {
		request.SetLabels(a.additionalLabels)
	}

	return request, nil
}

// BuildAddTargetRequest returns the ALB AddTargetsRequest.
// address: IPv4 address.
func (a *ALBTargetGroupBuilder) BuildAddTargetRequest(address string) *alb.AddTargetsRequest {
	return &alb.AddTargetsRequest{
		TargetGroupId: a.targetGroupID,
		Targets: []*alb.Target{{
			SubnetId: a.subnetID,
			AddressType: &alb.Target_IpAddress{
				IpAddress: address,
			}},
		},
	}
}

// BuildRemoveTargetRequest returns the ALB RemoveTargetsRequest.
// address: IPv4 address.
func (a *ALBTargetGroupBuilder) BuildRemoveTargetRequest(address string) *alb.RemoveTargetsRequest {
	return &alb.RemoveTargetsRequest{
		TargetGroupId: a.targetGroupID,
		Targets: []*alb.Target{{
			SubnetId: a.subnetID,
			AddressType: &alb.Target_IpAddress{
				IpAddress: address,
			}},
		},
	}
}

// GetName gets the ALB target group name from ALBBackendGroupBuilder.
func (a *ALBTargetGroupBuilder) GetName() string {
	return a.name
}

// NewALBBackendGroupBuilder returns the new NewALBBackendGroupBuilder.
func NewALBBackendGroupBuilder(lbs infrav1.LoadBalancerSpec) *ALBBackendGroupBuilder {
	return &ALBBackendGroupBuilder{lbs: lbs}
}

// WithCluster sets the CAPI cluster name.
func (a *ALBBackendGroupBuilder) WithCluster(clusterName string) *ALBBackendGroupBuilder {
	a.clusterName = clusterName
	return a
}

// WithFolder sets YandexCloud FolderID.
func (a *ALBBackendGroupBuilder) WithFolder(folderID string) *ALBBackendGroupBuilder {
	a.folderID = folderID
	return a
}

// WithLBName sets the BackendGroup name.
func (a *ALBBackendGroupBuilder) WithLBName(name string) *ALBBackendGroupBuilder {
	a.name = name
	return a
}

// WithTargetGroupID sets the BackendGroup name.
func (a *ALBBackendGroupBuilder) WithTargetGroupID(id string) *ALBBackendGroupBuilder {
	a.targetGroupID = id
	return a
}

// WithLabels sets an additional set of tags on BackendGroup.
func (a *ALBBackendGroupBuilder) WithLabels(labels infrav1.Labels) *ALBBackendGroupBuilder {
	a.additionalLabels = labels
	return a
}

// Build prepares and returns the ALB backend group creation request.
func (a *ALBBackendGroupBuilder) Build() (*alb.CreateBackendGroupRequest, error) {
	sb := &alb.StreamBackend{
		Name: a.name,
	}

	sb.SetPort(int64(a.lbs.BackendPort))
	sb.SetTargetGroups(&alb.TargetGroupsBackend{TargetGroupIds: []string{a.targetGroupID}})
	sb.SetLoadBalancingConfig(a.createLoadBalancingConfig())
	sb.SetHealthchecks(a.createHealthChecks())

	sbg := &alb.StreamBackendGroup{
		Backends: []*alb.StreamBackend{sb},
	}

	// create backend request.
	request := &alb.CreateBackendGroupRequest{
		FolderId:    a.folderID,
		Name:        a.name,
		Description: describePrefix + a.clusterName + " backend",
	}

	if a.additionalLabels != nil {
		request.SetLabels(a.additionalLabels)
	}

	request.SetStream(sbg)
	return request, nil
}

// GetName gets the ALB backend group name from ALBBackendGroupBuilder.
func (a *ALBBackendGroupBuilder) GetName() string {
	return a.name
}

// createLoadBalancingConfig creates backend configuration for ALB.
func (a *ALBBackendGroupBuilder) createLoadBalancingConfig() *alb.LoadBalancingConfig {
	cfg := &alb.LoadBalancingConfig{}
	cfg.SetMode(alb.LoadBalancingMode_ROUND_ROBIN)
	cfg.SetStrictLocality(false)

	return cfg
}

// createHealthChecks creates list of Stream healtchecks for ALB.
func (a *ALBBackendGroupBuilder) createHealthChecks() []*alb.HealthCheck {
	var healthChecks []*alb.HealthCheck
	timeout := time.Second * time.Duration(a.lbs.Healthcheck.HealthcheckTimeoutSec)
	interval := time.Second * time.Duration(a.lbs.Healthcheck.HealthcheckIntervalSec)

	healthCheck := &alb.HealthCheck{}
	healthCheck.SetTimeout(durationpb.New(timeout))
	healthCheck.SetInterval(durationpb.New(interval))
	healthCheck.SetUnhealthyThreshold(int64(a.lbs.Healthcheck.HealthcheckThreshold))
	healthCheck.SetHealthyThreshold(int64(a.lbs.Healthcheck.HealthcheckThreshold))
	healthCheck.SetStream(&alb.HealthCheck_StreamHealthCheck{})

	return append(healthChecks, healthCheck)
}

// NewALBBuilder returns the new ALBBuilder.
func NewALBBuilder(lbs infrav1.LoadBalancerSpec) *ALBBuilder {
	return &ALBBuilder{lbs: lbs}
}

// WithCluster sets the CAPI cluster name.
func (a *ALBBuilder) WithCluster(clusterName string) *ALBBuilder {
	a.clusterName = clusterName
	return a
}

// WithFolder sets YandexCloud FolderID.
func (a *ALBBuilder) WithFolder(folderID string) *ALBBuilder {
	a.folderID = folderID
	return a
}

// WithName sets the ALB name.
func (a *ALBBuilder) WithName(name string) *ALBBuilder {
	a.name = name
	return a
}

// WithBackendGroupID sets the BackendGroup ID.
func (a *ALBBuilder) WithBackendGroupID(id string) *ALBBuilder {
	a.backendGroupID = id
	return a
}

// WithNetworkID sets the Network ID.
func (a *ALBBuilder) WithNetworkID(id string) *ALBBuilder {
	a.networkID = id
	return a
}

// WithLabels sets an additional set of tags on ALB.
func (a *ALBBuilder) WithLabels(labels infrav1.Labels) *ALBBuilder {
	a.additionalLabels = labels
	return a
}

// Build  prepares and returns the ALB creation request.
func (a *ALBBuilder) Build() (*alb.CreateLoadBalancerRequest, error) {
	request := &alb.CreateLoadBalancerRequest{
		FolderId:    a.folderID,
		NetworkId:   a.networkID,
		Name:        a.name,
		Description: describePrefix + a.clusterName + " loadbalancer",
	}

	// Single zone allocation only in alpha version.
	request.SetAllocationPolicy(&alb.AllocationPolicy{
		Locations: []*alb.Location{{
			ZoneId:   a.lbs.Listener.Subnet.ZoneID,
			SubnetId: a.lbs.Listener.Subnet.ID,
		}},
	})

	if a.additionalLabels != nil {
		request.SetLabels(a.additionalLabels)
	}

	request.SetListenerSpecs(a.createListenerSpec(
		a.lbs.Listener.Address,
		a.lbs.Listener.Port,
		a.lbs.Listener.Subnet.ID,
		a.backendGroupID))
	request.SetLogOptions(&alb.LogOptions{Disable: true})

	return request, nil
}

// GetName gets the ALB backend group name from ALBBackendGroupBuilder.
func (a *ALBBuilder) GetName() string {
	return a.name
}

// createListenerSpec prepares and returns the ALB Listener specification.
func (a *ALBBuilder) createListenerSpec(address string, port int32, subnetID, backendID string) []*alb.ListenerSpec {
	intAddressSpec := &alb.InternalIpv4AddressSpec{}
	if address != "" {
		intAddressSpec.SetAddress(address)
	}
	intAddressSpec.SetSubnetId(subnetID)
	addressSpec := &alb.AddressSpec{}
	addressSpec.SetInternalIpv4AddressSpec(intAddressSpec)

	endpointSpec := &alb.EndpointSpec{}
	endpointSpec.SetPorts([]int64{int64(port)})
	endpointSpec.SetAddressSpecs([]*alb.AddressSpec{addressSpec})

	listenerSpec := &alb.ListenerSpec{}
	listenerSpec.SetName(a.name)
	listenerSpec.SetEndpointSpecs([]*alb.EndpointSpec{endpointSpec})

	// Stream listener only.
	streamHandler := &alb.StreamHandler{}
	streamHandler.SetBackendGroupId(backendID)

	streamListener := &alb.StreamListener{}
	streamListener.SetHandler(streamHandler)
	listenerSpec.SetStream(streamListener)

	return []*alb.ListenerSpec{listenerSpec}
}
