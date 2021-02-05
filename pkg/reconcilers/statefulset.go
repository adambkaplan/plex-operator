package reconcilers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
	"github.com/go-logr/logr"
)

// StatefulSetReconciler is a reconciler for the PlexMediaServer's StatefulSet
type StatefulSetReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reconciles an object with the desired state of the PlexMediaServer
func (r *StatefulSetReconciler) Reconcile(ctx context.Context, plex *plexv1alpha1.PlexMediaServer) (bool, error) {
	origStatefulSet := &appsv1.StatefulSet{}
	namespacedName := types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}
	log := r.Log.WithValues("statefulset", namespacedName)
	err := r.Client.Get(ctx, namespacedName, origStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		origStatefulSet = createStatefulSet(plex, r.Scheme)
		err = r.Client.Create(ctx, origStatefulSet, &client.CreateOptions{})
		if err != nil {
			log.Error(err, "failed to create object")
			return true, err
		}
		log.Info("created object")
		return true, nil
	}

	desiredStatefulSet := origStatefulSet.DeepCopy()
	desiredStatefulSet.Spec = renderStatefulSetSpec(plex, origStatefulSet.Spec)

	if !equality.Semantic.DeepEqual(origStatefulSet.Spec, desiredStatefulSet.Spec) {
		err = r.Update(ctx, desiredStatefulSet, &client.UpdateOptions{})
		if err != nil {
			log.Error(err, "failed to update object")
			return true, err
		}
		return true, nil
	}

	return false, nil
}

// createStatefulSet creates a StatefulSet for the Plex media server
func createStatefulSet(plex *plexv1alpha1.PlexMediaServer, scheme *runtime.Scheme) *appsv1.StatefulSet {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: plex.Namespace,
			Name:      plex.Name,
		},
	}
	statefulSet.Spec = renderStatefulSetSpec(plex, statefulSet.Spec)
	ctrl.SetControllerReference(plex, statefulSet, scheme)
	return statefulSet
}

// renderStatefulSetSpec renders a StatefulSet spec for the Plex Media Server on top of the
// existing StatefulSetSpec. This ensures that the output StatefulSetSpec aligns with the settings
// in the PlexMediaServer configuration.
func renderStatefulSetSpec(plex *plexv1alpha1.PlexMediaServer, existingStatefulSet appsv1.StatefulSetSpec) appsv1.StatefulSetSpec {
	replicas := int32(1)
	existingStatefulSet.Replicas = &replicas
	existingStatefulSet.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"plex.adambkaplan.com/instance": plex.Name,
		},
	}
	existingStatefulSet.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{
			"plex.adambkaplan.com/instance": plex.Name,
		},
	}
	containers := []corev1.Container{}
	version := plex.Spec.Version
	if version == "" {
		version = "latest"
	}
	plexContainer := corev1.Container{
		Name:  "plex",
		Image: fmt.Sprintf("docker.io/plexinc/pms-docker:%s", version),
	}
	containers = append(containers, plexContainer)
	existingStatefulSet.Template.Spec.Containers = containers
	return existingStatefulSet
}
