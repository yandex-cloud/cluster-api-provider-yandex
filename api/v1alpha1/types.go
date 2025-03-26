package v1alpha1

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// InstanceStatus describes the status of a Yandex Cloud Compute instance.
type InstanceStatus string

var (
	// InstanceStatusProvisioning is the string representing an instance in a provisioning state.
	InstanceStatusProvisioning = InstanceStatus("PROVISIONING")
	// InstanceStatusRunning is the string representing an instance in a running state.
	InstanceStatusRunning = InstanceStatus("RUNNING")
	// InstanceStatusError is the string representing an instance in a error state.
	InstanceStatusError = InstanceStatus("ERROR")
	// InstanceStatusStopped is the string representing an instance in a stopped state.
	InstanceStatusStopped = InstanceStatus("STOPPED")
	// InstanceStatusStarting is the string representing an instance in a starting state.
	InstanceStatusStarting = InstanceStatus("STARTING")
	// InstanceStatusStopping is the string representing an instance in a stopping state.
	InstanceStatusStopping = InstanceStatus("STOPPING")
	// InstanceStatusRestarting is the string representing an instance in a restarting state.
	InstanceStatusRestarting = InstanceStatus("RESTARTING")
	// InstanceStatusUpdating is the string representing an instance in a updating state.
	InstanceStatusUpdating = InstanceStatus("UPDATING")
	// InstanceStatusCrashed is the string representing an instance in a crashed state.
	InstanceStatusCrashed = InstanceStatus("CRASHED")
	// InstanceStatusDeleting is the string representing an instance in a deleting state.
	InstanceStatusDeleting = InstanceStatus("DELETING")
	// InstanceStatusDeleted is the string representing an instance in a deleted state.
	InstanceStatusDeleted = InstanceStatus("DELETED")
	// InstanceStatusUnspecified is the string representing an instance in a unknown state.
	InstanceStatusUnspecified = InstanceStatus("UNSPECIFIED")
)

const (
	// ConditionStatusProvisioning is the string representing an instance in a provisioning state.
	ConditionStatusProvisioning clusterv1.ConditionType = "PROVISIONING"
	// ConditionStatusRunning is the string representing an instance in a running state.
	ConditionStatusRunning = "RUNNING"
	// ConditionStatusReady is the string representing an instance in a ready state.
	ConditionStatusReady = "READY"
	// ConditionStatusError is the string representing an instance in a error state.
	ConditionStatusError = "ERROR"
	// ConditionStatusNotfound used when the instance couldn't be retrieved.
	ConditionStatusNotfound = "NOTFOUND"
)

const (
	// LoadBalancerReadyCondition reports on whether a control plane load balancer was successfully reconciled.
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
	// LoadBalancerFailedReason used when an error occurs during load balancer reconciliation.
	LoadBalancerFailedReason = "LoadBalancerFailed"
)

const (
	// ProviderIDSetCondition reports on whether the providerID was successfully set on the node.
	ProviderIDSetCondition clusterv1.ConditionType = "ProviderIDSet"
)
