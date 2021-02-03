package statefulset

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

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
			outSpec := RenderStatefulSetSpec(tc.plex, tc.existing)
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
