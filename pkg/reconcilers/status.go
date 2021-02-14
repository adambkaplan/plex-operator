package reconcilers

import (
	"context"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

type StatusReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewStatusReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *StatusReconciler {
	return &StatusReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *StatusReconciler) Reconcile(ctx context.Context, plex *v1alpha1.PlexMediaServer) (bool, error) {
	origPlex := plex.DeepCopy()
	plex.Status.ObservedGeneration = plex.Generation
	log := r.Log.WithValues("status.observedGeneration", plex.Generation)

	readyCondition := v1.Condition{
		Type:               "Ready",
		ObservedGeneration: plex.Generation,
		Status:             v1.ConditionUnknown,
	}

	statefulSet := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, statefulSet)
	if err != nil && !errors.IsNotFound(err) {
		log.WithValues("statefulset", types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}).
			Error(err, "failed to get object")
		return true, err
	}
	if errors.IsNotFound(err) {
		meta.SetStatusCondition(&plex.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			"Plex media server deployment not found",
			readyCondition))
		err = r.Client.Status().Update(ctx, plex, &client.UpdateOptions{})
		if errors.IsConflict(err) {
			log.Info("conflict updating object, requeueing")
			return true, nil
		}
		if err != nil {
			log.Error(err, "failed to update object")
			return true, err
		}
		log.Info("updated status")
		return false, nil
	}

	ready := statefulSet.Status.ReadyReplicas > 0

	if ready {
		meta.SetStatusCondition(&plex.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(true),
			"AsExpected",
			"Plex media server has at least 1 ready replica",
			readyCondition,
		))
	} else {
		meta.SetStatusCondition(&plex.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotReady",
			"Plex media server has no ready replicas",
			readyCondition,
		))
	}

	if equality.Semantic.DeepEqual(plex.Status, origPlex.Status) {
		return false, nil
	}

	err = r.Client.Status().Update(ctx, plex, &client.UpdateOptions{})
	if errors.IsConflict(err) {
		log.Info("conflict updating status, requeuing")
		return true, nil
	}
	if err != nil {
		log.Error(err, "error updating status")
		return true, err
	}
	log.Info("updated status")
	return false, nil
}

func (r *StatusReconciler) setStatusInfo(status v1.ConditionStatus, reason string, message string, condition v1.Condition) v1.Condition {
	condition.Status = status
	condition.Reason = reason
	condition.Message = message
	return condition
}

func (r *StatusReconciler) conditionStatus(b bool) v1.ConditionStatus {
	if b {
		return v1.ConditionTrue
	}
	return v1.ConditionFalse
}
