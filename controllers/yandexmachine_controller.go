/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	compute "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/services/compute"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
)

const (
	// RequeueDuration is pause before reconcile will be repeated
	RequeueDuration time.Duration = 10 * time.Second
)

// YandexMachineReconciler reconciles a YandexMachine object.
type YandexMachineReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	YandexClient yandex.Client
}

//+kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=yandexmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=yandexmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=yandexmachines/finalizers,verbs=update

// Reconcile brings YandexMachine into desired state.
func (r *YandexMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	yandexMachine := &infrav1.YandexMachine{}
	err := r.Get(ctx, req.NamespacedName, yandexMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("error occurred while fetching YandexMachine resource: %w", err)
	}

	logger.V(1).Info("machine found")
	machine, err := util.GetOwnerMachine(ctx, r.Client, yandexMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("machine controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", machine.Name)
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("machine is missing cluster label or cluster does not exist")

		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", cluster.Name)
	if annotations.IsPaused(cluster, yandexMachine) {
		logger.Info("YandexMachine or linked cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	yandexCluster := &infrav1.YandexCluster{}
	yandexClusterKey := client.ObjectKey{
		Namespace: yandexMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err = r.Client.Get(ctx, yandexClusterKey, yandexCluster); err != nil {
		logger.Info("YandexCluster is not available yet")
		return ctrl.Result{}, nil
	}

	clusterScope, err := scope.NewClusterScope(ctx, scope.ClusterScopeParams{
		Client:        r.Client,
		Cluster:       cluster,
		YandexCluster: yandexCluster,
		YandexClient:  r.YandexClient,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:        r.Client,
		Machine:       machine,
		YandexMachine: yandexMachine,
		ClusterGetter: clusterScope,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// always close the scope when exiting this function so we can persist any YandexMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()

	if !yandexMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	return r.reconcile(ctx, machineScope)
}

// reconcile it is a part of reconciliation loop in case of yandexmachine update/create.
func (r *YandexMachineReconciler) reconcile(ctx context.Context, machineScope *scope.MachineScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconciling YandexMachine")

	controllerutil.AddFinalizer(machineScope.YandexMachine, infrav1.MachineFinalizer)

	if err := machineScope.PatchObject(); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("bootstrap data is not ready yet: linked Machine's bootstrap.dataSecretName is nil. Skipping reconciliation")
		return ctrl.Result{}, nil
	}

	if err := compute.New(machineScope).Reconcile(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("error reconciling instance resources: %w", err)
	}

	instanceState := *machineScope.GetInstanceStatus()
	switch instanceState {
	case infrav1.InstanceStatusStarting, infrav1.InstanceStatusProvisioning:
		logger.Info("YandexMachine instance is provisioning", "instance-id", machineScope.GetInstanceID())
		return ctrl.Result{RequeueAfter: RequeueDuration}, nil
	case infrav1.InstanceStatusRunning:
		logger.Info("YandexMachine instance is running", "instance-id", machineScope.GetInstanceID())
		machineScope.SetReady()
		return ctrl.Result{}, nil
	default:
		machineScope.SetFailureReason(capierrors.UpdateMachineError)
		machineScope.SetFailureMessage(errors.Errorf("YandexMachine instance state %s is unexpected", instanceState))
		return ctrl.Result{RequeueAfter: RequeueDuration}, nil
	}
}

// reconcileDelete it is a part of reconciliation loop in case of yandexmachine delete.
func (r *YandexMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("reconciling YandexMachine delete")

	deleted, err := compute.New(machineScope).Delete(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error deleting instance resources: %w", err)
	}

	if deleted {
		logger.Info("YandexMachine instance is deleted", "instance-id", machineScope.GetInstanceID())
		controllerutil.RemoveFinalizer(machineScope.YandexMachine, infrav1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	logger.Info("YandexMachine instance is deleting", "instance-id", machineScope.GetInstanceID())
	return ctrl.Result{RequeueAfter: RequeueDuration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *YandexMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.YandexMachine{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("YandexMachine"))),
		).
		Watches(
			&infrav1.YandexCluster{},
			handler.EnqueueRequestsFromMapFunc(r.YandexClusterToYandexMachines(ctx)),
		).
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	clusterToObjectFunc, err := util.ClusterToTypedObjectsMapper(r.Client, &infrav1.YandexMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to YandexMachines")
	}

	// add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &clusterv1.Cluster{}),
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

// YandexClusterToYandexMachines is a handler.ToRequestsFunc to be used to enqeue requests for reconciliation
// of YandexMachines.
func (r *YandexMachineReconciler) YandexClusterToYandexMachines(ctx context.Context) handler.MapFunc {
	logger := ctrl.LoggerFrom(ctx)
	return func(mapCtx context.Context, o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*infrav1.YandexCluster)
		if !ok {
			logger.Error(errors.Errorf("expected a YandexCluster but got a %T", o), "failed to get YandexMachine for YandexCluster")
			return nil
		}

		cluster, err := util.GetOwnerCluster(mapCtx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			logger.Error(err, "failed to get owning cluster")
			return result
		}

		labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(mapCtx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			logger.Error(err, "failed to list Machines")
			return nil
		}
		for _, m := range machineList.Items {
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}
