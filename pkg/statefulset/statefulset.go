package statefulset

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
)

// CreateStatefulSet creates a StatefulSet for the Plex media server
func CreateStatefulSet(plex *plexv1alpha1.PlexMediaServer, scheme *runtime.Scheme) *appsv1.StatefulSet {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: plex.Namespace,
			Name:      plex.Name,
		},
	}
	statefulSet.Spec = RenderStatefulSetSpec(plex, statefulSet.Spec)
	ctrl.SetControllerReference(plex, statefulSet, scheme)
	return statefulSet
}

// RenderStatefulSetSpec renders a StatefulSet spec for the Plex Media Server on top of the
// existing StatefulSetSpec. This ensures that the output StatefulSetSpec aligns with the settings
// in the PlexMediaServer configuration.
func RenderStatefulSetSpec(plex *plexv1alpha1.PlexMediaServer, existingStatefulSet appsv1.StatefulSetSpec) appsv1.StatefulSetSpec {
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
