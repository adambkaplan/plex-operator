package reconcilers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func TestStatefulSetReconcile(t *testing.T) {
	assert := assert.New(t)
	log := logr.Discard()
	cases := []struct {
		name                string
		plex                *v1alpha1.PlexMediaServer
		existingStatefulSet *appsv1.StatefulSet
		expectedStatefulSet *appsv1.StatefulSet
		expectError         bool
		expectRequeue       bool
	}{
		{
			name: "create - default",
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
			name: "create - with version",
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
			name: "update - with version",
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			scheme := scheme.Scheme
			err := v1alpha1.AddToScheme(scheme)
			assert.Nil(err, "test %s: failed to add scheme", tc.name)
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if tc.plex != nil {
				builder.WithObjects(tc.plex)
			}
			if tc.existingStatefulSet != nil {
				builder.WithObjects(tc.existingStatefulSet)
			}
			client := builder.Build()
			reconciler := &StatefulSetReconciler{
				Client: client,
				Scheme: client.Scheme(),
				Log:    log,
			}
			requeue, err := reconciler.Reconcile(ctx, tc.plex)
			assert.Equal(tc.expectRequeue, requeue, "requeue result should be equal")
			if tc.expectError {
				assert.Error(err, "expected error was not returned")
			}
			updatedStatefulSet := &appsv1.StatefulSet{}
			_ = client.Get(ctx, types.NamespacedName{Namespace: tc.plex.Namespace, Name: tc.plex.Name}, updatedStatefulSet)
			assert.True(equality.Semantic.DeepEqual(tc.expectedStatefulSet.Spec, updatedStatefulSet.Spec),
				"test %s: statefulset %s does not match %s",
				tc.name,
				tc.expectedStatefulSet.Spec,
				updatedStatefulSet.Spec)
			// TODO: Verify owner references
		})
	}
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
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"plex.adambkaplan.com/instance": name,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "plex",
							Image: fmt.Sprintf("docker.io/plexinc/pms-docker:%s", version),
						},
					},
				},
			},
		},
	}
}

func TestRenderStatefulSetSpec(t *testing.T) {
	cases := []struct {
		name     string
		plex     *v1alpha1.PlexMediaServer
		existing appsv1.StatefulSetSpec
	}{
		{
			name: "empty",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test",
				},
			},
		},
		{
			name: "versioned",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Version: "v1.21",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outSpec := renderStatefulSetSpec(tc.plex, tc.existing)
			if *outSpec.Replicas != int32(1) {
				t.Errorf("expected replicas %d, got %d", 1, *outSpec.Replicas)
			}

			firstContainer := outSpec.Template.Spec.Containers[0]
			if firstContainer.Name != "plex" {
				t.Errorf("expected first container name %s, got %s", "plex", firstContainer.Name)
			}
			expectedVersion := "latest"
			if tc.plex.Spec.Version != "" {
				expectedVersion = tc.plex.Spec.Version
			}
			expectedImage := fmt.Sprintf("docker.io/plexinc/pms-docker:%s", expectedVersion)
			if firstContainer.Image != expectedImage {
				t.Errorf("expected image to be %s, got %s", expectedImage, firstContainer.Image)
			}
		})
	}
}
