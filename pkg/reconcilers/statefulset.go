package reconcilers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
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
	if errors.IsNotFound(err) {
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

	if !equality.Semantic.DeepEqual(origStatefulSet.Spec.VolumeClaimTemplates, desiredStatefulSet.Spec.VolumeClaimTemplates) {
		log.Info("deleting because volume claim templates changed")
		log.Info(fmt.Sprintf("diff: %s", cmp.Diff(origStatefulSet.Spec.VolumeClaimTemplates, desiredStatefulSet.Spec.VolumeClaimTemplates)))
		background := metav1.DeletePropagationBackground
		err = r.Delete(ctx, desiredStatefulSet, &client.DeleteOptions{
			PropagationPolicy: &background,
		})
		if errors.IsConflict(err) {
			log.Info("conflict on delete, requeueing")
			return true, nil
		}
		if err != nil {
			log.Error(err, "failed to delete object")
			return true, err
		}
		return true, nil
	}

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
	existingStatefulSet.Template.Spec.Containers = r.renderContainers(plex, existingStatefulSet.Template.Spec.Containers)
	existingStatefulSet.Template.Spec.Volumes = r.renderPlexPodVolumes(plex, existingStatefulSet.Template.Spec.Volumes)
	existingStatefulSet.VolumeClaimTemplates = r.renderPlexVolumeClaims(plex, existingStatefulSet.VolumeClaimTemplates)
	return existingStatefulSet
}

func (r *StatefulSetReconciler) renderContainers(plex *plexv1alpha1.PlexMediaServer, existing []corev1.Container) []corev1.Container {
	containers := []corev1.Container{}
	plexContainer := corev1.Container{
		Name: "plex",
	}
	for _, c := range existing {
		if c.Name == "plex" {
			plexContainer = c
			continue
		}
		containers = append(containers, c)
	}
	version := plex.Spec.Version
	if version == "" {
		version = "latest"
	}
	plexContainer.Image = fmt.Sprintf("docker.io/plexinc/pms-docker:%s", version)
	plexContainer.Ports = r.renderPlexContainerPorts(plex, plexContainer.Ports)
	plexContainer.VolumeMounts = r.renderPlexContainerVolumeMounts(plexContainer.VolumeMounts)
	containers = append(containers, plexContainer)
	return containers
}

func (r *StatefulSetReconciler) renderPlexContainerPorts(plex *v1alpha1.PlexMediaServer, existing []corev1.ContainerPort) []corev1.ContainerPort {
	containerPorts := []corev1.ContainerPort{}
	plexPort := corev1.ContainerPort{
		ContainerPort: 32400,
	}
	discovery0 := corev1.ContainerPort{
		ContainerPort: 32410,
	}
	discovery1 := corev1.ContainerPort{
		ContainerPort: 32412,
	}
	discovery2 := corev1.ContainerPort{
		ContainerPort: 32413,
	}
	discovery3 := corev1.ContainerPort{
		ContainerPort: 32414,
	}

	for _, port := range existing {
		if port.ContainerPort == int32(32400) {
			plexPort = port
			continue
		}
		if port.ContainerPort == 32410 {
			discovery0 = port
			continue
		}
		if port.ContainerPort == 32412 {
			discovery1 = port
			continue
		}
		if port.ContainerPort == 32413 {
			discovery2 = port
			continue
		}
		if port.ContainerPort == 32414 {
			discovery3 = port
			continue
		}
		// Append any other ContainerPorts to the returned slice
		containerPorts = append(containerPorts, port)
	}

	plexPort.Name = "plex"
	plexPort.Protocol = corev1.ProtocolTCP
	containerPorts = append(containerPorts, plexPort)

	if plex.Spec.Networking.EnableDiscovery {
		discovery0.Name = "discovery-0"
		discovery0.Protocol = corev1.ProtocolUDP

		discovery1.Name = "discovery-1"
		discovery1.Protocol = corev1.ProtocolUDP

		discovery2.Name = "discovery-2"
		discovery2.Protocol = corev1.ProtocolUDP

		discovery3.Name = "discovery-3"
		discovery3.Protocol = corev1.ProtocolUDP

		containerPorts = append(containerPorts, discovery0, discovery1, discovery2, discovery3)
	}

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

func (r *StatefulSetReconciler) renderPlexPodVolumes(plex *plexv1alpha1.PlexMediaServer, existing []corev1.Volume) []corev1.Volume {
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
	if plex.Spec.Storage.Config == nil {
		volumes = append(volumes, configVolume)
	}

	transcodeVolume.EmptyDir = &corev1.EmptyDirVolumeSource{}
	if plex.Spec.Storage.Transcode == nil {
		volumes = append(volumes, transcodeVolume)
	}

	dataVolume.EmptyDir = &corev1.EmptyDirVolumeSource{}
	if plex.Spec.Storage.Data == nil {
		volumes = append(volumes, dataVolume)
	}
	return volumes
}

func (r *StatefulSetReconciler) renderPlexVolumeClaims(plex *plexv1alpha1.PlexMediaServer, existing []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	claims := []corev1.PersistentVolumeClaim{}
	config := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config",
		},
	}
	transcode := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "transcode",
		},
	}
	data := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "data",
		},
	}
	for _, v := range existing {
		if v.Name == "config" {
			config = v
			continue
		}
		if v.Name == "transcode" {
			transcode = v
			continue
		}
		if v.Name == "data" {
			data = v
			continue
		}
		claims = append(claims, v)
	}

	if newConfig, add := r.renderPersistentVolumeClaim(config, plex.Spec.Storage.Config); add {
		claims = append(claims, newConfig)
	}
	if newTranscode, add := r.renderPersistentVolumeClaim(transcode, plex.Spec.Storage.Transcode); add {
		claims = append(claims, newTranscode)
	}
	if newData, add := r.renderPersistentVolumeClaim(data, plex.Spec.Storage.Data); add {
		claims = append(claims, newData)
	}
	return claims
}

// renderPersistentVolumeClaim renders a PVC on top of the provided PVC, based on the configuration
// provided in the PlexStorageOptions. It returns the updated PVC, and true if the PVC should be
// appended to the StatefulSet volume claim template.
func (r *StatefulSetReconciler) renderPersistentVolumeClaim(existing corev1.PersistentVolumeClaim, spec *plexv1alpha1.PlexStorageOptions) (corev1.PersistentVolumeClaim, bool) {
	if spec == nil {
		return existing, false
	}
	pvc := existing.DeepCopy()
	if len(spec.AccessMode) > 0 {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{
			spec.AccessMode,
		}
	}
	if !spec.Capacity.IsZero() {
		if pvc.Spec.Resources.Requests == nil {
			pvc.Spec.Resources.Requests = make(corev1.ResourceList)
		}
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = spec.Capacity
	}
	if spec.StorageClassName != nil {
		pvc.Spec.StorageClassName = spec.StorageClassName
	}
	if spec.Selector != nil {
		pvc.Spec.Selector = spec.Selector
	}
	return *pvc, true
}
