package reconcilers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

type statefulSetTestCase struct {
	name                string
	plex                *v1alpha1.PlexMediaServer
	existingStatefulSet *appsv1.StatefulSet
	expectedStatefulSet *appsv1.StatefulSet
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
			expectRequeue:       true,
			expectedStatefulSet: mockStatefulSet("test", "test", 1, "latest"),
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
			expectRequeue:       true,
			expectedStatefulSet: mockStatefulSet("test", "test-version", 1, "v1.21"),
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
			existingStatefulSet: mockStatefulSet("update", "update-version", 1, "latest"),
			expectedStatefulSet: mockStatefulSet("update", "update-version", 1, "v1.25"),
			expectRequeue:       true,
		},
		{
			name: "no change",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "no-change",
					Name:      "no-change",
				},
			},
			existingStatefulSet: mockStatefulSet("no-change", "no-change", 1, "latest"),
			expectedStatefulSet: mockStatefulSet("no-change", "no-change", 1, "latest"),
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
				Client: client,
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
				"expected statefulset\n\n%s\n\ndoes not match\n\n%s",
				tc.expectedStatefulSet.Spec,
				updatedStatefulSet.Spec)
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

func mockStatefulSet(namespace, name string, replicas int32, version string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
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
			Replicas: &replicas,
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
							Image: fmt.Sprintf("docker.io/plexinc/pms-docker:%s", version),
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(32400),
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestStatefulSetSuite(t *testing.T) {
	suite.Run(t, new(statefulSetReconcileSuite))
}
