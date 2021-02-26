package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Storage options", func() {

	var (
		plex          *v1alpha1.PlexMediaServer
		testNamespace *v1.Namespace
		ctx           context.Context
		retryInterval = 100 * time.Millisecond
		retryTimeout  = 1 * time.Second
	)

	JustBeforeEach(func() {
		ctx, testNamespace = InitTestEnvironment(k8sClient, plex)
	})

	JustAfterEach(func() {
		TearDownTestEnvironment(ctx, k8sClient, plex, testNamespace)
	})

	When("the Config PVC attributes are set", func() {

		BeforeEach(func() {
			block := "block"
			plex = &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("storage-config"),
					Name:      "test-config",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Storage: v1alpha1.PlexStorageSpec{
						Config: &v1alpha1.PlexStorageOptions{
							AccessMode:       v1.ReadWriteOnce,
							StorageClassName: &block,
							Capacity:         resource.MustParse("10Gi"),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"media": "plex",
								},
							},
						},
					},
				},
			}
		})

		It("adds the Config PVC attributes to the StatefulSet volume claim template", func() {
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
			By("checking the volumes on the StatefulSet")
			Expect(len(statefulSet.Spec.VolumeClaimTemplates)).To(Equal(1))
			configTemplate := statefulSet.Spec.VolumeClaimTemplates[0]
			Expect(configTemplate.Name).To(Equal("config"))
			Expect(configTemplate.Spec.AccessModes).To(Equal([]v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}))
			Expect(configTemplate.Spec.Resources.Requests).To(Equal(v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("10Gi"),
			}))
			Expect(*configTemplate.Spec.StorageClassName).To(BeEquivalentTo("block"))
			Expect(configTemplate.Spec.Selector).To(BeEquivalentTo(&metav1.LabelSelector{
				MatchLabels: map[string]string{
					"media": "plex",
				},
			}))
			podVolumes := statefulSet.Spec.Template.Spec.Volumes
			foundTranscode := false
			foundData := false
			for _, volume := range podVolumes {
				if volume.Name == "transcode" {
					foundTranscode = true
					Expect(volume.EmptyDir).NotTo(BeNil())
				}
				if volume.Name == "data" {
					foundData = true
					Expect(volume.EmptyDir).NotTo(BeNil())
				}
			}
			Expect(foundTranscode).To(BeTrue())
			Expect(foundData).To(BeTrue())
		})
	})

	When("the Transcode PVC attributes are set", func() {

		BeforeEach(func() {
			block := "block"
			plex = &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("storage-config"),
					Name:      "test-transcode",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Storage: v1alpha1.PlexStorageSpec{
						Transcode: &v1alpha1.PlexStorageOptions{
							AccessMode:       v1.ReadWriteOnce,
							StorageClassName: &block,
							Capacity:         resource.MustParse("10Gi"),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"media": "plex",
								},
							},
						},
					},
				},
			}
		})

		It("adds the Transcode PVC attributes to the StatefulSet volume claim template", func() {
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
			By("checking the volumes on the StatefulSet")
			Expect(len(statefulSet.Spec.VolumeClaimTemplates)).To(Equal(1))
			transcodeTemplate := statefulSet.Spec.VolumeClaimTemplates[0]
			Expect(transcodeTemplate.Name).To(Equal("transcode"))
			Expect(transcodeTemplate.Spec.AccessModes).To(Equal([]v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}))
			Expect(transcodeTemplate.Spec.Resources.Requests).To(Equal(v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("10Gi"),
			}))
			Expect(*transcodeTemplate.Spec.StorageClassName).To(BeEquivalentTo("block"))
			Expect(transcodeTemplate.Spec.Selector).To(BeEquivalentTo(&metav1.LabelSelector{
				MatchLabels: map[string]string{
					"media": "plex",
				},
			}))
			podVolumes := statefulSet.Spec.Template.Spec.Volumes
			foundConfig := false
			foundData := false
			for _, volume := range podVolumes {
				if volume.Name == "config" {
					foundConfig = true
					Expect(volume.EmptyDir).NotTo(BeNil())
				}
				if volume.Name == "data" {
					foundData = true
					Expect(volume.EmptyDir).NotTo(BeNil())
				}
			}
			Expect(foundConfig).To(BeTrue())
			Expect(foundData).To(BeTrue())
		})
	})

	When("the Data PVC attributes are set", func() {

		BeforeEach(func() {
			nfs := "nfs"
			plex = &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: RandomName("storage-data"),
					Name:      "test-data",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Storage: v1alpha1.PlexStorageSpec{
						Data: &v1alpha1.PlexStorageOptions{
							AccessMode:       v1.ReadWriteMany,
							StorageClassName: &nfs,
							Capacity:         resource.MustParse("100Gi"),
							Selector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"media": "plex",
								},
							},
						},
					},
				},
			}
		})

		It("adds the Data PVC attributes to the StatefulSet volume claim template", func() {
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
			By("checking the volumes on the StatefulSet")
			Expect(len(statefulSet.Spec.VolumeClaimTemplates)).To(Equal(1))
			dataTemplate := statefulSet.Spec.VolumeClaimTemplates[0]
			Expect(dataTemplate.Name).To(Equal("data"))
			Expect(dataTemplate.Spec.AccessModes).To(Equal([]v1.PersistentVolumeAccessMode{v1.ReadWriteMany}))
			Expect(dataTemplate.Spec.Resources.Requests).To(Equal(v1.ResourceList{
				v1.ResourceStorage: resource.MustParse("100Gi"),
			}))
			Expect(*dataTemplate.Spec.StorageClassName).To(Equal("nfs"))
			Expect(dataTemplate.Spec.Selector).To(BeEquivalentTo(&metav1.LabelSelector{
				MatchLabels: map[string]string{
					"media": "plex",
				},
			}))
			podVolumes := statefulSet.Spec.Template.Spec.Volumes
			foundTranscode := false
			foundConfig := false
			for _, volume := range podVolumes {
				if volume.Name == "transcode" {
					foundTranscode = true
					Expect(volume.EmptyDir).NotTo(BeNil())
				}
				if volume.Name == "config" {
					foundConfig = true
					Expect(volume.EmptyDir).NotTo(BeNil())
				}
			}
			Expect(foundTranscode).To(BeTrue())
			Expect(foundConfig).To(BeTrue())
		})
	})
})
