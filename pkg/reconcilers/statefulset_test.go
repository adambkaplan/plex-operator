package reconcilers

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

type statefulSetDoubleOptions struct {
	Replicas        int32
	Version         string
	IncludeDefaults bool
	Ready           bool
	ConfigVolume    *corev1.PersistentVolumeClaimSpec
	TranscodeVolume *corev1.PersistentVolumeClaimSpec
	DataVolume      *corev1.PersistentVolumeClaimSpec
}

func doubleStatefulSet(namespace, name string, options statefulSetDoubleOptions) *appsv1.StatefulSet {
	if options.Version == "" {
		options.Version = "latest"
	}
	if options.Ready && options.Replicas < 1 {
		options.Replicas = 1
	}
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"plex.adambkaplan.com/instance": name,
				},
			},
			Replicas:    &options.Replicas,
			ServiceName: name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"plex.adambkaplan.com/instance": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "plex",
							Image: fmt.Sprintf("docker.io/plexinc/pms-docker:%s", options.Version),
							Ports: []corev1.ContainerPort{
								{
									Name:          "plex",
									ContainerPort: int32(32400),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
								},
								{
									Name:      "transcode",
									MountPath: "/transcode",
								},
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
					},
				},
			},
		},
	}
	podVolumes := []corev1.Volume{}
	volumeClaimTemplates := []corev1.PersistentVolumeClaim{}
	if options.ConfigVolume == nil {
		podVolumes = append(podVolumes, corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	} else {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "config",
			},
			Spec: *options.ConfigVolume,
		})
	}
	if options.TranscodeVolume == nil {
		podVolumes = append(podVolumes, corev1.Volume{
			Name: "transcode",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	} else {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "transcode",
			},
			Spec: *options.TranscodeVolume,
		})
	}
	if options.DataVolume == nil {
		podVolumes = append(podVolumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	} else {
		volumeClaimTemplates = append(volumeClaimTemplates, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: *options.DataVolume,
		})
	}
	statefulSet.Spec.Template.Spec.Volumes = podVolumes
	statefulSet.Spec.VolumeClaimTemplates = volumeClaimTemplates
	if options.IncludeDefaults {
		plexContainer := statefulSet.Spec.Template.Spec.Containers[0]
		plexContainer.ImagePullPolicy = corev1.PullAlways
		plexContainer.TerminationMessagePolicy = corev1.TerminationMessageReadFile
		plexContainer.TerminationMessagePath = "/dev/termination-log"
		statefulSet.Spec.Template.Spec.Containers[0] = plexContainer
	}
	if options.Ready {
		statefulSet.Status.Replicas = options.Replicas
		statefulSet.Status.ReadyReplicas = options.Replicas
	}
	return statefulSet
}

type statefulSetTestCase struct {
	name                string
	plex                *v1alpha1.PlexMediaServer
	existingStatefulSet *appsv1.StatefulSet
	expectedStatefulSet *appsv1.StatefulSet
	errCreate           error
	errUpdate           error
	expectError         bool
	expectRequeue       bool
}

type statefulSetReconcileSuite struct {
	suite.Suite
	cases []statefulSetTestCase
}

func (test *statefulSetReconcileSuite) SetupTest() {
	test.cases = []statefulSetTestCase{
		{
			name: "create with defaults",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
			},
			expectRequeue: true,
			expectedStatefulSet: doubleStatefulSet("test", "test", statefulSetDoubleOptions{
				Replicas: 1,
			}),
		},
		{
			name: "create with version",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-version",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Version: "v1.21",
				},
			},
			expectRequeue: true,
			expectedStatefulSet: doubleStatefulSet("test", "test-version", statefulSetDoubleOptions{
				Replicas: 1,
				Version:  "v1.21",
			}),
		},
		{
			name: "create with one persistent volume",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-volume",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Storage: v1alpha1.PlexMediaServerStorageSpec{
						Config: &v1alpha1.PlexStorageOptions{
							AccessMode: corev1.ReadWriteOnce,
							Capacity:   resource.MustParse("10Gi"),
						},
					},
				},
			},
			expectRequeue: true,
			expectedStatefulSet: doubleStatefulSet("test", "test-volume", statefulSetDoubleOptions{
				Replicas: 1,
				ConfigVolume: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			}),
		},
		{
			name: "create with all persistent volumes",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test-volume",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Storage: v1alpha1.PlexMediaServerStorageSpec{
						Config: &v1alpha1.PlexStorageOptions{
							AccessMode: corev1.ReadWriteOnce,
							Capacity:   resource.MustParse("10Gi"),
						},
						Transcode: &v1alpha1.PlexStorageOptions{
							AccessMode: corev1.ReadWriteOnce,
							Capacity:   resource.MustParse("10Gi"),
						},
						Data: &v1alpha1.PlexStorageOptions{
							AccessMode: corev1.ReadWriteMany,
							Capacity:   resource.MustParse("100Gi"),
						},
					},
				},
			},
			expectRequeue: true,
			expectedStatefulSet: doubleStatefulSet("test", "test-volume", statefulSetDoubleOptions{
				Replicas: 1,
				ConfigVolume: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
				TranscodeVolume: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
				DataVolume: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteMany,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			}),
		},
		{
			name: "update with version",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "update",
					Name:      "update-version",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Version: "v1.25",
				},
			},
			existingStatefulSet: doubleStatefulSet("update", "update-version", statefulSetDoubleOptions{
				Replicas:        1,
				IncludeDefaults: true,
			}),
			expectedStatefulSet: doubleStatefulSet("update", "update-version", statefulSetDoubleOptions{
				Replicas:        1,
				Version:         "v1.25",
				IncludeDefaults: true,
			}),
			expectRequeue: true,
		},
		{
			name: "update with conflict",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "no-change",
					Name:      "no-change",
				},
			},
			errUpdate: errors.NewConflict(schema.ParseGroupResource("statefulset.apps"), "no-change", fmt.Errorf("test")),
			existingStatefulSet: doubleStatefulSet("no-change", "no-change", statefulSetDoubleOptions{
				Replicas:        1,
				Version:         "v1.23",
				IncludeDefaults: true,
			}),
			expectedStatefulSet: doubleStatefulSet("no-change", "no-change", statefulSetDoubleOptions{
				Replicas:        1,
				Version:         "v1.23",
				IncludeDefaults: true,
			}),
			expectError:   false,
			expectRequeue: true,
		},
		{
			name: "no change",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "no-change",
					Name:      "no-change",
				},
			},
			existingStatefulSet: doubleStatefulSet("no-change", "no-change", statefulSetDoubleOptions{
				Replicas:        1,
				IncludeDefaults: true,
			}),
			expectedStatefulSet: doubleStatefulSet("no-change", "no-change", statefulSetDoubleOptions{
				Replicas:        1,
				IncludeDefaults: true,
			}),
		},
	}
}

func (test *statefulSetReconcileSuite) TestStatefulSetReconcile() {
	log := logr.Discard()

	for _, tc := range test.cases {
		test.Run(tc.name, func() {
			ctx := context.TODO()
			scheme := scheme.Scheme
			err := v1alpha1.AddToScheme(scheme)
			test.Require().Nil(err, "failed to add scheme")
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if tc.plex != nil {
				builder.WithObjects(tc.plex)
			}
			if tc.existingStatefulSet != nil {
				err = ctrl.SetControllerReference(tc.plex, tc.existingStatefulSet, scheme)
				test.Require().NoError(err, "failed to set controller reference")
				builder.WithObjects(tc.existingStatefulSet)
			}
			client := builder.Build()
			reconciler := &StatefulSetReconciler{
				Client: &errorClient{
					Client:    client,
					errCreate: tc.errCreate,
					errUpdate: tc.errUpdate,
				},
				Scheme: client.Scheme(),
				Log:    log,
			}
			requeue, err := reconciler.Reconcile(ctx, tc.plex)
			test.Equal(tc.expectRequeue, requeue, "requeue result should be equal")
			if tc.expectError {
				test.Error(err, "expected error was not returned")
				return
			}
			updatedStatefulSet := &appsv1.StatefulSet{}
			err = client.Get(ctx, types.NamespacedName{Namespace: tc.plex.Namespace, Name: tc.plex.Name}, updatedStatefulSet)
			test.Require().NoError(err, "failed to get StatefulSet")
			test.True(equality.Semantic.DeepEqual(tc.expectedStatefulSet.Spec, updatedStatefulSet.Spec),
				"expected statefulSet does not match - diff: %s",
				cmp.Diff(tc.expectedStatefulSet.Spec, updatedStatefulSet.Spec))
			test.True(plexOwnsStatefulSet(tc.plex, updatedStatefulSet),
				"statefulSet not owned by plex. Owner references: %s",
				updatedStatefulSet.OwnerReferences)
		})
	}
}

func plexOwnsStatefulSet(plex *v1alpha1.PlexMediaServer, statefulSet *appsv1.StatefulSet) bool {
	for _, ref := range statefulSet.OwnerReferences {
		if ref.Kind == "PlexMediaServer" && ref.Name == plex.Name && *ref.Controller {
			return true
		}
	}
	return false
}

func TestStatefulSetSuite(t *testing.T) {
	suite.Run(t, new(statefulSetReconcileSuite))
}
