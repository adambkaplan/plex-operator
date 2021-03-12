/*
Copyright Adam B Kaplan

SPDX-License-Identifier: Apache-2.0
*/
package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

var _ = Describe("Network Roku access", func() {

	var (
		plex          *v1alpha1.PlexMediaServer
		testNamespace *corev1.Namespace
		ctx           context.Context
	)

	JustBeforeEach(func() {
		ctx, testNamespace = InitTestEnvironment(k8sClient, plex)
	})

	JustAfterEach(func() {
		TearDownTestEnvironment(ctx, k8sClient, plex, testNamespace)
	})

	When("Roku access is enabled", func() {

		BeforeEach(func() {
			plex = &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("roku"),
					Name:      "plex",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						EnableRoku: true,
					},
				},
			}
		})

		It("exposes Roku port on the StatefulSet", func() {
			statefulSet := &appsv1.StatefulSet{}
			By("checking the StatefulSet exists")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, statefulSet)
				if err != nil {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(statefulSet).NotTo(BeNil())
			By("checking the StatefulSet's first container ports")
			Expect(len(statefulSet.Spec.Template.Spec.Containers)).To(Equal(1))
			firstContainer := statefulSet.Spec.Template.Spec.Containers[0]
			Expect(firstContainer.Name).To(Equal("plex"))
			expectedPorts := []int32{8324}
			foundPorts := []int32{}
			for _, port := range firstContainer.Ports {
				foundPorts = append(foundPorts, port.ContainerPort)
			}
			Expect(foundPorts).To(ContainElements(expectedPorts))
		})

		It("exposes Roku ports on the headless service", func() {
			By("checking the Service exposes the Plex Media Server port")
			service := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Namespace: plex.Namespace, Name: plex.Name}, service)
				if err != nil {
					return false
				}
				return true
			}, retryTimeout, retryInterval).Should(BeTrue())
			Expect(service).NotTo(BeNil())
			expectedPorts := []int32{8324}
			foundPorts := []int32{}
			for _, port := range service.Spec.Ports {
				foundPorts = append(foundPorts, port.Port)
			}
			Expect(foundPorts).To(ContainElements(expectedPorts))
		})

	})

	When("Roku is enabled with an external service", func() {

		BeforeEach(func() {
			plex = &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("roku"),
					Name:      "plex",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						EnableRoku:          true,
						ExternalServiceType: corev1.ServiceTypeLoadBalancer,
					},
				},
			}
		})

		It("exposes Roke ports on the external service", func() {
			testExternalService(ctx, plex)
		})
	})
})
