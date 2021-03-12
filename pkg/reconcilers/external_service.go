package reconcilers

import (
	"context"
	"fmt"

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

// ExternalServiceReconciler reconciles the external Service deployment for Plex Media Server
type ExternalServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// NewServiceReconciler returns a new Reconciler that reconciles the Service for Plex Media Server
func NewExternalServiceReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *ExternalServiceReconciler {
	return &ExternalServiceReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *ExternalServiceReconciler) Reconcile(ctx context.Context, plex *v1alpha1.PlexMediaServer) (bool, error) {
	origService := &corev1.Service{}
	serviceName := fmt.Sprintf("%s-ext", plex.Name)
	namespacedName := types.NamespacedName{Namespace: plex.Namespace, Name: serviceName}
	log := r.Log.WithValues("service", namespacedName)
	err := r.Client.Get(ctx, namespacedName, origService)

	if errors.IsNotFound(err) {
		// Only create if service is not found and an external service type was specified
		if plex.Spec.Networking.ExternalServiceType == "" {
			return false, nil
		}
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

	// If the external service type is set to "", this means we no longer need an external service
	if plex.Spec.Networking.ExternalServiceType == "" {
		log.Info("deleting")
		background := metav1.DeletePropagationBackground
		err = r.Client.Delete(ctx, origService, &client.DeleteOptions{
			PropagationPolicy: &background,
		})
		if err != nil {
			return true, err
		}
		return true, nil
	}

	// Reconcile the existing service based on the specification
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

func (r *ExternalServiceReconciler) createService(plex *v1alpha1.PlexMediaServer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: plex.Namespace,
			Name:      fmt.Sprintf("%s-ext", plex.Name),
		},
	}
	service.Spec = r.renderServiceSpec(plex, service.Spec)
	ctrl.SetControllerReference(plex, service, r.Scheme)
	return service
}

func (r *ExternalServiceReconciler) renderServiceSpec(plex *v1alpha1.PlexMediaServer, existingService corev1.ServiceSpec) corev1.ServiceSpec {
	existingService.Selector = map[string]string{
		"plex.adambkaplan.com/instance": plex.Name,
	}
	// TODO: Update invalid fields if we transition from NodePort -> LoadBalancer, and vice versa
	existingService.Type = plex.Spec.Networking.ExternalServiceType
	existingService.Ports = r.renderServicePorts(plex, existingService.Ports)
	return existingService
}

func (r *ExternalServiceReconciler) renderServicePorts(plex *v1alpha1.PlexMediaServer, existing []corev1.ServicePort) []corev1.ServicePort {
	servicePorts := []corev1.ServicePort{}
	rokuPort := corev1.ServicePort{
		Port: 8324,
	}
	plexPort := corev1.ServicePort{
		Port: 32400,
	}
	for _, port := range existing {
		if port.Port == 8324 {
			rokuPort = port
		}
		if port.Port == 32400 {
			plexPort = port
		}
	}

	if plex.Spec.Networking.EnableRoku {
		rokuPort.Name = "roku"
		rokuPort.Protocol = corev1.ProtocolTCP
		servicePorts = append(servicePorts, rokuPort)
	}

	plexPort.Protocol = corev1.ProtocolTCP
	plexPort.Name = "plex"
	servicePorts = append(servicePorts, plexPort)
	return servicePorts
}
