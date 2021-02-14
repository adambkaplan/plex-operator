package reconcilers

import (
	"context"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

// ServiceReconciler reconciles the Service deployment for Plex Media Server
type ServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// NewServiceReconciler returns a new Reconciler that reconciles the Service for Plex Media Server
func NewServiceReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *ServiceReconciler {
	return &ServiceReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, plex *v1alpha1.PlexMediaServer) (bool, error) {
	origService := &corev1.Service{}
	namespacedName := types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}
	log := r.Log.WithValues("service", namespacedName)
	err := r.Client.Get(ctx, namespacedName, origService)
	if err != nil && errors.IsNotFound(err) {
		log.Info("creating")
		origService = r.createService(plex)
		err = r.Client.Create(ctx, origService, &client.CreateOptions{})
		if err != nil {
			log.Error(err, "failed to create object")
			return true, err
		}
		log.Info("created object")
		return true, nil
	}
	if err != nil {
		return true, err
	}
	desiredService := origService.DeepCopy()
	desiredService.Spec = r.renderServiceSpec(plex, desiredService.Spec)
	if !equality.Semantic.DeepEqual(origService.Spec, desiredService.Spec) {
		log.Info("updating")
		err = r.Update(ctx, desiredService, &client.UpdateOptions{})
		if errors.IsConflict(err) {
			log.Info("conflict on update, requeueing")
			return true, nil
		}
		if err != nil {
			log.Error(err, "failed to update object")
			return true, err
		}
		log.Info("updated object")
		return true, nil
	}

	return false, nil
}

func (r *ServiceReconciler) createService(plex *v1alpha1.PlexMediaServer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: plex.Namespace,
			Name:      plex.Name,
		},
	}
	service.Spec = r.renderServiceSpec(plex, service.Spec)
	ctrl.SetControllerReference(plex, service, r.Scheme)
	return service
}

func (r *ServiceReconciler) renderServiceSpec(plex *v1alpha1.PlexMediaServer, existingService corev1.ServiceSpec) corev1.ServiceSpec {
	existingService.Selector = map[string]string{
		"plex.adambkaplan.com/instance": plex.Name,
	}
	existingService.ClusterIP = corev1.ClusterIPNone
	existingService.Ports = r.renderServicePorts(existingService.Ports)
	return existingService
}

func (r *ServiceReconciler) renderServicePorts(existing []corev1.ServicePort) []corev1.ServicePort {
	servicePorts := []corev1.ServicePort{}
	plexPort := corev1.ServicePort{}
	for _, port := range existing {
		if port.Port == int32(32400) {
			plexPort = port
		}
	}
	plexPort.Port = int32(32400)
	plexPort.Protocol = corev1.ProtocolTCP
	plexPort.Name = "plex"
	servicePorts = append(servicePorts, plexPort)
	return servicePorts
}
