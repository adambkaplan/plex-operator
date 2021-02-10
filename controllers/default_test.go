package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
)

var _ = Describe("Default deployment", func() {

	var (
		plexMediaServer *plexv1alpha1.PlexMediaServer
		testNamespace   *corev1.Namespace
		ctx             context.Context
		retryInterval   = 100 * time.Millisecond
		retryTimeout    = 1 * time.Second
	)

	JustBeforeEach(func() {
		ctx, testNamespace = InitTestEnvironment(k8sClient, plexMediaServer)
	})

	JustAfterEach(func() {
		TearDownTestEnvironment(ctx, k8sClient, plexMediaServer, testNamespace)
	})

	When("a PlexMediaServer object is created", func() {

		BeforeEach(func() {
			plexMediaServer = &plexv1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("default"),
					Name:      "plex-server",
				},
				Spec: plexv1alpha1.PlexMediaServerSpec{
					ClaimToken: "CHANGEME",
				},
			}
		})

		It("creates a StatefulSet to deploy the Plex media server", func() {
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plexMediaServer.Namespace, Name: plexMediaServer.Name}, statefulSet)
				if err != nil {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(statefulSet).NotTo(BeNil())
			Expect(statefulSet.Spec.Selector).To(Equal(&metav1.LabelSelector{
				MatchLabels: map[string]string{
					"plex.adambkaplan.com/instance": plexMediaServer.Name,
				},
			}))
			Expect(len(statefulSet.Spec.Template.Spec.Containers)).To(Equal(1))
			firstContainer := statefulSet.Spec.Template.Spec.Containers[0]
			Expect(firstContainer.Name).To(Equal("plex"))
			expectedVersion := plexMediaServer.Spec.Version
			if expectedVersion == "" {
				expectedVersion = "latest"
			}
			Expect(firstContainer.Image).To(Equal(fmt.Sprintf("docker.io/plexinc/pms-docker:%s", expectedVersion)))
		})

		It("creates a Service to route requests to the Plex Media Server", func() {
			By("checking the Service exposes the Plex Media Server port")
			service := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plexMediaServer.Namespace, Name: plexMediaServer.Name}, service)
				if err != nil {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(service).NotTo(BeNil())
			Expect(service.Spec.Selector).To(BeEquivalentTo(map[string]string{
				"plex.adambkaplan.com/instance": plexMediaServer.Name,
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
			By("checking the StatefulSet accepts traffic to the Plex Media Server Port")
			statefulSet := &appsv1.StatefulSet{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plexMediaServer.Namespace, Name: plexMediaServer.Name}, statefulSet)
				if err != nil {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(statefulSet).NotTo(BeNil())
			foundPlex = false
			for _, container := range statefulSet.Spec.Template.Spec.Containers {
				if container.Name != "plex" {
					continue
				}
				for _, port := range container.Ports {
					if port.ContainerPort == int32(32400) {
						foundPlex = true
						Expect(port.Protocol).To(Equal(corev1.ProtocolTCP))
						break
					}
				}
			}
			Expect(foundPlex).To(BeTrue())
		})
	})
})
