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
	"sigs.k8s.io/controller-runtime/pkg/client"

	plexv1alpha1 "github.com/adambkaplan/plex-operator/api/v1alpha1"
)

var _ = Describe("Default Deployment", func() {

	var (
		plexMediaServer *plexv1alpha1.PlexMediaServer
		testNamespace   *corev1.Namespace
		ctx             context.Context
		retryInterval   = 100 * time.Millisecond
		retryTimeout    = 1 * time.Second
	)

	JustBeforeEach(func() {
		ctx = context.Background()
		testNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: plexMediaServer.Namespace,
			},
		}
		err := k8sClient.Create(ctx, testNamespace, &client.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Create(ctx, plexMediaServer, &client.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	JustAfterEach(func() {
		err := k8sClient.Delete(ctx, plexMediaServer, &client.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.Delete(ctx, testNamespace, &client.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	When("a PlexMediaServer object is created", func() {

		BeforeEach(func() {
			plexMediaServer = &plexv1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
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
	})
})
