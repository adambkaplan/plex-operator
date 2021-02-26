package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
)

var _ = Describe("External Service", func() {

	var (
		plex          *plexv1alpha1.PlexMediaServer
		testNamespace *corev1.Namespace
		ctx           context.Context
		retryInterval = 100 * time.Microsecond
		retryTimeout  = 1 * time.Second
	)

	JustBeforeEach(func() {
		ctx, testNamespace = InitTestEnvironment(k8sClient, plex)
	})

	JustAfterEach(func() {
		TearDownTestEnvironment(ctx, k8sClient, plex, testNamespace)
	})

	When("a LoadBalancer external service is enabled", func() {

		BeforeEach(func() {
			plex = &plexv1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("external-service"),
					Name:      "plex",
				},
				Spec: plexv1alpha1.PlexMediaServerSpec{
					Networking: plexv1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeLoadBalancer,
					},
				},
			}
		})

		It("creates a LoadBalancer service to send traffic to the Plex Media Server", func() {
			testExternalService(ctx, plex, retryTimeout, retryInterval)
		})

		It("updates the service to NodePort if the external service type is later changed to NodePort", func() {
			testExternalService(ctx, plex, retryTimeout, retryInterval)
			var err error
			Eventually(func() bool {
				currentPlex := &plexv1alpha1.PlexMediaServer{}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, currentPlex)
				if err != nil {
					return true
				}
				currentPlex.Spec.Networking.ExternalServiceType = corev1.ServiceTypeNodePort
				err = k8sClient.Update(ctx, currentPlex, &client.UpdateOptions{})
				if errors.IsConflict(err) {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() corev1.ServiceType {
				svc := &corev1.Service{}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: fmt.Sprintf("%s-ext", plex.Name)}, svc)
				if err != nil {
					return ""
				}
				return svc.Spec.Type
			}, retryTimeout, retryInterval).Should(Equal(corev1.ServiceTypeLoadBalancer))
		})

		It("deletes the service if the external service type is later removed", func() {
			testExternalService(ctx, plex, retryTimeout, retryInterval)
			var err error
			Eventually(func() bool {
				currentPlex := &plexv1alpha1.PlexMediaServer{}
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, currentPlex)
				if err != nil {
					return true
				}
				currentPlex.Spec.Networking.ExternalServiceType = ""
				err = k8sClient.Update(ctx, currentPlex, &client.UpdateOptions{})
				if errors.IsConflict(err) {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				svc := &corev1.Service{}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: fmt.Sprintf("%s-ext", plex.Name)}, svc)
				return errors.IsNotFound(err)
			}, retryTimeout, retryInterval).Should(BeTrue())
		})
	})

	When("a NodePort external service is enabled", func() {

		BeforeEach(func() {
			plex = &plexv1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("external-service"),
					Name:      "plex",
				},
				Spec: plexv1alpha1.PlexMediaServerSpec{
					Networking: plexv1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeNodePort,
					},
				},
			}
		})

		It("creates a NodePort service to send traffic to the Plex Media Server", func() {
			testExternalService(ctx, plex, retryTimeout, retryInterval)
		})

		It("updates the service to LoadBalancer if the external service type is later changed to LoadBalancer", func() {
			testExternalService(ctx, plex, retryTimeout, retryInterval)
			var err error
			Eventually(func() bool {
				currentPlex := &plexv1alpha1.PlexMediaServer{}
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, currentPlex)
				if err != nil {
					return true
				}
				currentPlex.Spec.Networking.ExternalServiceType = corev1.ServiceTypeLoadBalancer
				err = k8sClient.Update(ctx, currentPlex, &client.UpdateOptions{})
				if errors.IsConflict(err) {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())

			Expect(err).NotTo(HaveOccurred())
			Eventually(func() corev1.ServiceType {
				svc := &corev1.Service{}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: fmt.Sprintf("%s-ext", plex.Name)}, svc)
				if err != nil {
					return ""
				}
				return svc.Spec.Type
			}, retryTimeout, retryInterval).Should(Equal(corev1.ServiceTypeLoadBalancer))
		})

		It("deletes the service if the external service type is later removed", func() {
			testExternalService(ctx, plex, retryTimeout, retryInterval)
			var err error
			Eventually(func() bool {
				currentPlex := &plexv1alpha1.PlexMediaServer{}
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, currentPlex)
				if err != nil {
					return true
				}
				currentPlex.Spec.Networking.ExternalServiceType = ""
				err = k8sClient.Update(ctx, currentPlex, &client.UpdateOptions{})
				if errors.IsConflict(err) {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				svc := &corev1.Service{}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: fmt.Sprintf("%s-ext", plex.Name)}, svc)
				return errors.IsNotFound(err)
			}, retryTimeout, retryInterval).Should(BeTrue())
		})
	})
})

func testExternalService(ctx context.Context, plex *v1alpha1.PlexMediaServer, retryTimeout, retryInterval time.Duration) {
	service := &corev1.Service{}
	By("finding the external service")
	Eventually(func() bool {
		err := k8sClient.Get(ctx,
			types.NamespacedName{Namespace: plex.Namespace, Name: fmt.Sprintf("%s-ext", plex.Name)},
			service)
		if err != nil {
			return false
		}
		return true
	}, retryTimeout, retryInterval).Should(BeTrue())
	By("checking the external service spec")
	Expect(service.Spec.Type).To(Equal(plex.Spec.Networking.ExternalServiceType))
	Expect(service.Spec.Selector).To(BeEquivalentTo(map[string]string{
		"plex.adambkaplan.com/instance": plex.Name,
	}))
	foundPlex := false
	for _, port := range service.Spec.Ports {
		if port.Name == "plex" {
			foundPlex = true
			Expect(port.Port).To(BeEquivalentTo(32400))
			Expect(port.Protocol).To(Equal(corev1.ProtocolTCP))
			break
		}
	}
	Expect(foundPlex).To(BeTrue())
}
