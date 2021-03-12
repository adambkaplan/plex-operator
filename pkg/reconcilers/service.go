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
	existingService.Ports = r.renderServicePorts(plex, existingService.Ports)
	return existingService
}

func (r *ServiceReconciler) renderServicePorts(plex *v1alpha1.PlexMediaServer, existing []corev1.ServicePort) []corev1.ServicePort {
	servicePorts := []corev1.ServicePort{}
	rokuPort := corev1.ServicePort{
		Port: 8324,
	}
	plexPort := corev1.ServicePort{
		Port: 32400,
	}
	discovery0 := corev1.ServicePort{
		Port: 32410,
	}
	discovery1 := corev1.ServicePort{
		Port: 32412,
	}
	discovery2 := corev1.ServicePort{
		Port: 32413,
	}
	discovery3 := corev1.ServicePort{
		Port: 32414,
	}
	for _, port := range existing {
		if port.Port == 8324 {
			rokuPort = port
			continue
		}
		if port.Port == 32400 {
			plexPort = port
			continue
		}
		if port.Port == 32410 {
			discovery0 = port
			continue
		}
		if port.Port == 32412 {
			discovery1 = port
			continue
		}
		if port.Port == 32413 {
			discovery2 = port
			continue
		}
		if port.Port == 32414 {
			discovery3 = port
			continue
		}
		servicePorts = append(servicePorts, port)
	}

	if plex.Spec.Networking.EnableRoku {
		rokuPort.Name = "roku"
		rokuPort.Protocol = corev1.ProtocolTCP
		servicePorts = append(servicePorts, rokuPort)
	}

	plexPort.Name = "plex"
	plexPort.Protocol = corev1.ProtocolTCP

	servicePorts = append(servicePorts, plexPort)
	if plex.Spec.Networking.EnableDiscovery {
		discovery0.Name = "discovery-0"
		discovery0.Protocol = corev1.ProtocolUDP

		discovery1.Name = "discovery-1"
		discovery1.Protocol = corev1.ProtocolUDP

		discovery2.Name = "discovery-2"
		discovery2.Protocol = corev1.ProtocolUDP

		discovery3.Name = "discovery-3"
		discovery3.Protocol = corev1.ProtocolUDP

		servicePorts = append(servicePorts, discovery0, discovery1, discovery2, discovery3)
	}
	return servicePorts
}
