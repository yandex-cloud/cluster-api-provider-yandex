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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"

	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/options"

	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
)

// YandexClusterReconciler reconciles a YandexCluster object.
type YandexClusterReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	YandexClient yandex.Client
	Config       options.Config
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=yandexclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=yandexclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=yandexclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the YandexCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *YandexClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	logger := log.FromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, r.Config.ReconcileTimeout)
	defer cancel()

	yandexCluster := &infrav1.YandexCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, yandexCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get the Cluster
	cluster, err := util.GetOwnerCluster(ctx, r.Client, yandexCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		logger.Info("Waiting for Cluster Controller to set OwnerRef on YandexCluster")
		return ctrl.Result{}, nil
	}

	if annotations.IsPaused(cluster, yandexCluster) {
		logger.Info("YandexCluster or owning Cluster is marked as paused, not reconciling")
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

	// Always close the scope when exiting this function so we can persist any YandexMachine changes.
	defer func() {
		if err := clusterScope.Close(ctx); err != nil && rerr == nil {
			rerr = err
		}
	}()

	// Handle deleted clusters
	if !yandexCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	return r.reconcile(ctx, clusterScope)
}

func (r *YandexClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer) {
		controllerutil.AddFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer)
		logger.Info("Finalizer added to YandexCluster, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	clusterScope.SetReady()

	return ctrl.Result{}, nil
}

func (r *YandexClusterReconciler) reconcileDelete(_ context.Context, clusterScope *scope.ClusterScope) (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(clusterScope.YandexCluster, infrav1.ClusterFinalizer)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *YandexClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.YandexCluster{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Build(r)
	if err != nil {
		return err
	}

	return c.Watch(
		source.Kind(mgr.GetCache(), &clusterv1.Cluster{}),
		handler.EnqueueRequestsFromMapFunc(
			util.ClusterToInfrastructureMapFunc(
				ctx,
				infrav1.GroupVersion.WithKind("YandexCluster"),
				mgr.GetClient(),
				&infrav1.YandexCluster{},
			),
		),
		predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx)),
	)
}
