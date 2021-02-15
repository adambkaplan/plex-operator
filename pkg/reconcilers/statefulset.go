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

// NewStatefulSetReconciler returns a Reconciler for Plex's StatefulSet
func NewStatefulSetReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *StatefulSetReconciler {
	return &StatefulSetReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

// Reconcile reconciles an object with the desired state of the PlexMediaServer
func (r *StatefulSetReconciler) Reconcile(ctx context.Context, plex *plexv1alpha1.PlexMediaServer) (bool, error) {
	origStatefulSet := &appsv1.StatefulSet{}
	namespacedName := types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}
	log := r.Log.WithValues("statefulset", namespacedName)
	err := r.Client.Get(ctx, namespacedName, origStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		log.Info("creating")
		origStatefulSet = r.createStatefulSet(plex)
		err = r.Client.Create(ctx, origStatefulSet, &client.CreateOptions{})
		if err != nil {
			log.Error(err, "failed to create object")
			return true, err
		}
		log.Info("created object")
		return true, nil
	}
	if err != nil {
		// Other errors, return true and force a requeue
		return true, err
	}

	desiredStatefulSet := origStatefulSet.DeepCopy()
	desiredStatefulSet.Spec = r.renderStatefulSetSpec(plex, desiredStatefulSet.Spec)

	if !equality.Semantic.DeepEqual(origStatefulSet.Spec, desiredStatefulSet.Spec) {
		log.Info("updating")
		err = r.Update(ctx, desiredStatefulSet, &client.UpdateOptions{})
		if errors.IsConflict(err) {
			log.Info("conflict on update, requeueing")
			return true, nil
		}
		if err != nil {
			log.Error(err, "failed to update object")
			return true, err
		}
		return true, nil
	}

	return false, nil
}

// createStatefulSet creates a StatefulSet for the Plex media server
func (r *StatefulSetReconciler) createStatefulSet(plex *plexv1alpha1.PlexMediaServer) *appsv1.StatefulSet {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: plex.Namespace,
			Name:      plex.Name,
		},
	}
	statefulSet.Spec = r.renderStatefulSetSpec(plex, statefulSet.Spec)
	ctrl.SetControllerReference(plex, statefulSet, r.Scheme)
	return statefulSet
}

// renderStatefulSetSpec renders a StatefulSet spec for the Plex Media Server on top of the
// existing StatefulSetSpec. This ensures that the output StatefulSetSpec aligns with the settings
// in the PlexMediaServer configuration.
func (r *StatefulSetReconciler) renderStatefulSetSpec(plex *plexv1alpha1.PlexMediaServer, existingStatefulSet appsv1.StatefulSetSpec) appsv1.StatefulSetSpec {
	replicas := int32(1)
	existingStatefulSet.Replicas = &replicas
	existingStatefulSet.ServiceName = plex.Name
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
	plexContainer := r.findPlexContainer(existingStatefulSet.Template.Spec.Containers)
	plexContainer.Image = fmt.Sprintf("docker.io/plexinc/pms-docker:%s", version)
	plexContainer.Ports = r.renderPlexContainerPorts(plexContainer.Ports)
	plexContainer.VolumeMounts = r.renderPlexContainerVolumeMounts(plexContainer.VolumeMounts)
	containers = append(containers, plexContainer)
	existingStatefulSet.Template.Spec.Containers = containers
	existingStatefulSet.Template.Spec.Volumes = r.renderPlexVolumes(existingStatefulSet.Template.Spec.Volumes)
	return existingStatefulSet
}

func (r *StatefulSetReconciler) findPlexContainer(existing []corev1.Container) corev1.Container {
	plexContainer := corev1.Container{
		Name: "plex",
	}
	for _, container := range existing {
		if container.Name == "plex" {
			plexContainer = container
			break
		}
	}
	return plexContainer
}

func (r *StatefulSetReconciler) renderPlexContainerPorts(existing []corev1.ContainerPort) []corev1.ContainerPort {
	containerPorts := []corev1.ContainerPort{}
	plexPort := corev1.ContainerPort{}
	for _, port := range existing {
		if port.ContainerPort == int32(32400) {
			plexPort = port
			continue
		}
		// Append any other ContainerPorts to the returned slice
		containerPorts = append(containerPorts, port)
	}
	plexPort.ContainerPort = int32(32400)
	plexPort.Name = "plex"
	plexPort.Protocol = corev1.ProtocolTCP
	containerPorts = append(containerPorts, plexPort)
	return containerPorts
}

func (r *StatefulSetReconciler) renderPlexContainerVolumeMounts(existing []corev1.VolumeMount) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}
	configMount := corev1.VolumeMount{Name: "config"}
	transcodeMount := corev1.VolumeMount{Name: "transcode"}
	dataMount := corev1.VolumeMount{Name: "data"}
	for _, mount := range existing {
		if mount.Name == "config" {
			configMount = mount
			continue
		}
		if mount.Name == "transcode" {
			transcodeMount = mount
			continue
		}
		if mount.Name == "data" {
			dataMount = mount
			continue
		}
		// Append any other volume mounts to the returned slice
		volumeMounts = append(volumeMounts, mount)
	}
	configMount.MountPath = "/config"
	transcodeMount.MountPath = "/transcode"
	dataMount.MountPath = "/data"
	volumeMounts = append(volumeMounts, configMount, transcodeMount, dataMount)
	return volumeMounts
}

func (r *StatefulSetReconciler) renderPlexVolumes(existing []corev1.Volume) []corev1.Volume {
	volumes := []corev1.Volume{}
	configVolume := corev1.Volume{Name: "config"}
	transcodeVolume := corev1.Volume{Name: "transcode"}
	dataVolume := corev1.Volume{Name: "data"}
	for _, volume := range existing {
		if volume.Name == "config" {
			configVolume = volume
			continue
		}
		if volume.Name == "transcode" {
			transcodeVolume = volume
			continue
		}
		if volume.Name == "data" {
			dataVolume = volume
			continue
		}
		volumes = append(volumes, volume)
	}
	configVolume.EmptyDir = &corev1.EmptyDirVolumeSource{}
	transcodeVolume.EmptyDir = &corev1.EmptyDirVolumeSource{}
	dataVolume.EmptyDir = &corev1.EmptyDirVolumeSource{}
	volumes = append(volumes, configVolume, transcodeVolume, dataVolume)
	return volumes
}
