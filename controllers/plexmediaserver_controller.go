/*
Copyright Adam B Kaplan

SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlexMediaServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&plexv1alpha1.PlexMediaServer{}).
		Complete(r)
}
