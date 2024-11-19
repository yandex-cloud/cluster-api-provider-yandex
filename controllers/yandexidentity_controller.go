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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pkg/errors"
	infrav1 "github.com/yandex-cloud/cluster-api-provider-yandex/api/v1alpha1"
	yandex "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/cloud/scope"
	"github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/options"
)

// YandexIdentityReconciler reconciles a YandexIdentity object.
type YandexIdentityReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	YandexClientBuilder yandex.Builder
	Config              options.Config
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=YandexIdentity,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=YandexIdentity/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=YandexIdentity/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get
//+kubebuilder:rbac:groups="",resources=secrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *YandexIdentityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	// logger := log.FromContext(ctx)

	ctx, cancel := context.WithTimeout(ctx, r.Config.ReconcileTimeout)
	defer cancel()

	identity := &infrav1.YandexIdentity{}
	if err := r.Client.Get(ctx, req.NamespacedName, identity); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, "failed to get YandexIdentity")
	}

	identityScope, err := scope.NewIdentityScope(scope.IdentityScopeParams{
		Client:         r.Client,
		Builder:        r.YandexClientBuilder,
		YandexIdentity: identity,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to create scope")
	}

	// Always close the scope when exiting this function so we can persist any YandexIdentity changes.
	defer func() {
		if err := identityScope.PersistIndentityChanges(ctx); err != nil && rerr == nil {
			rerr = err
		}
	}()

	// Handle deleted identities.
	if !identity.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, identityScope)
	}

	return r.reconcile(ctx, identityScope)
}

// reconcile it is a part of reconciliation loop in case of YandexIdentity update/create.
func (r *YandexIdentityReconciler) reconcile(ctx context.Context, identityScope *scope.IdentityScope) (_ ctrl.Result, rerr error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(identityScope.Identity, infrav1.IdentityFinalizer) {
		controllerutil.AddFinalizer(identityScope.Identity, infrav1.IdentityFinalizer)
		logger.Info("Finalizer added to YandexIdentity, requeueing")
		return ctrl.Result{Requeue: true}, nil
	}

	// Set status to ready if we return without error
	defer func() {
		if rerr == nil {
			identityScope.Identity.Status.Ready = true
			conditions.MarkTrue(identityScope.Identity, clusterv1.ReadyCondition)
		} else {
			identityScope.Identity.Status.Ready = false
			conditions.MarkFalse(identityScope.Identity, clusterv1.ReadyCondition, "ReconciliationError", clusterv1.ConditionSeverityError, rerr.Error())
		}
	}()

	if err := identityScope.CheckConnectWithIdentity(ctx); err != nil {
		conditions.MarkFalse(identityScope.Identity,
			infrav1.IdentityValidCondition, "identity check error", clusterv1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrap(err, "failed to check connection with identity")
	}

	// set IdentityValidCondition to true
	conditions.MarkTrue(identityScope.Identity, infrav1.IdentityValidCondition)

	if err := identityScope.SetSecretFinalizer(ctx); err != nil {
		conditions.MarkFalse(identityScope.Identity,
			infrav1.IdentitySecretUpdatedCondition, "secret update error", clusterv1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrap(err, "failed to set secret finalizer")
	}

	// set IdentitySecretUpdatedCondition to true
	conditions.MarkTrue(identityScope.Identity, infrav1.IdentitySecretUpdatedCondition)

	if err := identityScope.UpdateLinkedClusters(ctx); err != nil {
		conditions.MarkFalse(identityScope.Identity,
			infrav1.IdentityLinkedClustersUpdatedCondition, "linked clusters update error", clusterv1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrap(err, "failed to update linked clusters")
	}

	// set IdentityLinkedClustersUpdatedCondition to true
	conditions.MarkTrue(identityScope.Identity, infrav1.IdentityLinkedClustersUpdatedCondition)

	return ctrl.Result{}, nil
}

// reconcileDelete it is a part of reconciliation loop in case of YandexIdentity delete.
func (r *YandexIdentityReconciler) reconcileDelete(ctx context.Context, identityScope *scope.IdentityScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(identityScope.Identity, infrav1.ClusterFinalizer) {
		logger.Info("no finalizer found on YandexIdentity, skipping deletion reconciliation")
		return ctrl.Result{}, nil
	}

	linkedToClusters, err := identityScope.IsLinkedToCluster(ctx)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to check if identity is linked to cluster")
	}

	if linkedToClusters {
		logger.Info("identity is still used by clusters, skipping deletion", "clusters", identityScope.Identity.Status.LinkedClusters)

		return ctrl.Result{RequeueAfter: RequeueDuration}, nil
	}

	if err := identityScope.RemoveSecretFinalizer(ctx); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to remove secret finalizer")
	}

	controllerutil.RemoveFinalizer(identityScope.Identity, infrav1.IdentityFinalizer)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *YandexIdentityReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	watchForCluster := func(ctx context.Context, a client.Object) []reconcile.Request {
		cluster, ok := a.(*infrav1.YandexCluster)
		if !ok {
			return nil
		}

		if cluster.Spec.IdentityRef == nil {
			return nil
		}

		return []reconcile.Request{{NamespacedName: cluster.Spec.IdentityRef.NamespacedName()}}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.YandexIdentity{}).
		Watches(&infrav1.YandexCluster{}, handler.EnqueueRequestsFromMapFunc(watchForCluster)).
		Complete(r)
}
