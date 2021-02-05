/*
Copyright Adam B Kaplan

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
	"github.com/adambkaplan/plex-operator/pkg/reconcilers"
)

// PlexMediaServerReconciler reconciles a PlexMediaServer object
type PlexMediaServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=plex.adambkaplan.com,resources=plexmediaservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=plex.adambkaplan.com,resources=plexmediaservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=plex.adambkaplan.com,resources=plexmediaservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PlexMediaServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *PlexMediaServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("plexmediaserver", req.NamespacedName)

	// your logic here
	r.Log.V(5).Info("reconciling PlexMediaServer")
	plex := &plexv1alpha1.PlexMediaServer{}
	err := r.Client.Get(ctx, req.NamespacedName, plex)
	if err != nil {
		if errors.IsNotFound(err) {
			// Parent PlexMediaServer has been deleted
			// PlexMediaServer adds owner refs to managed objects, which should be garbage collected by Kubernetes
			return ctrl.Result{}, nil
		}
		// Requeue on error
		return ctrl.Result{Requeue: true}, err
	}

	reconciler := &reconcilers.StatefulSetReconciler{
		Client: r.Client,
		Log:    r.Log,
		Scheme: r.Scheme,
	}
	requeue, err := reconciler.Reconcile(ctx, plex)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{Requeue: requeue}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlexMediaServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&plexv1alpha1.PlexMediaServer{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
