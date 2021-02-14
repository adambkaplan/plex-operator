package reconcilers

import (
	"context"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
)

type statusTestCase struct {
	name                string
	plex                *v1alpha1.PlexMediaServer
	expectedStatus      v1alpha1.PlexMediaServerStatus
	existingStatefulSet *appsv1.StatefulSet
	expectError         bool
	expectRequeue       bool
}

type statusReconcileSuite struct {
	suite.Suite
	cases []statusTestCase
}

func (test *statusReconcileSuite) SetupTest() {
	test.cases = []statusTestCase{
		{
			name: "not created",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test",
					Name:       "not-created",
					Generation: int64(1),
				},
			},
			expectedStatus: v1alpha1.PlexMediaServerStatus{
				ObservedGeneration: int64(1),
				Conditions: []metav1.Condition{
					{
						Type:    "Ready",
						Status:  metav1.ConditionFalse,
						Reason:  "NotFound",
						Message: "Plex media server deployment not found",
					},
				},
			},
		},
		{
			name: "not ready",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test",
					Name:       "not-ready",
					Generation: int64(1),
				},
			},
			existingStatefulSet: mockStatefulSet("test", "not-ready", int32(1), "latest", true, false),
			expectedStatus: v1alpha1.PlexMediaServerStatus{
				ObservedGeneration: int64(1),
				Conditions: []metav1.Condition{
					{
						Type:    "Ready",
						Status:  metav1.ConditionFalse,
						Reason:  "NotReady",
						Message: "Plex media server has no ready replicas",
					},
				},
			},
		},
		{
			name: "ready",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  "test",
					Name:       "ready",
					Generation: int64(1),
				},
			},
			existingStatefulSet: mockStatefulSet("test", "ready", int32(1), "latest", true, true),
			expectedStatus: v1alpha1.PlexMediaServerStatus{
				ObservedGeneration: int64(1),
				Conditions: []metav1.Condition{
					{
						Type:    "Ready",
						Status:  metav1.ConditionTrue,
						Reason:  "AsExpected",
						Message: "Plex media server has at least 1 ready replica",
					},
				},
			},
		},
	}
}

func (test *statusReconcileSuite) TestStatusReconcile() {
	log := logr.Discard()

	for _, tc := range test.cases {
		test.Run(tc.name, func() {
			ctx := context.TODO()
			scheme := scheme.Scheme
			err := v1alpha1.AddToScheme(scheme)
			test.Require().NoError(err, "failed to add scheme")
			builder := fake.NewClientBuilder()
			if tc.plex != nil {
				builder.WithObjects(tc.plex)
			}
			if tc.existingStatefulSet != nil {
				err := ctrl.SetControllerReference(tc.plex, tc.existingStatefulSet, scheme)
				test.Require().NoError(err, "failed to set controller reference")
				builder.WithObjects(tc.existingStatefulSet)
			}
			client := builder.Build()
			reconciler := &StatusReconciler{
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
			updatedPlex := &v1alpha1.PlexMediaServer{}
			err = client.Get(ctx, types.NamespacedName{Namespace: tc.plex.Namespace, Name: tc.plex.Name}, updatedPlex)
			test.Require().NoError(err, "failed to get PlexMediaServer")
			test.Equal(updatedPlex.Status.ObservedGeneration, tc.expectedStatus.ObservedGeneration, "observedGeneration should be equal")
			for _, c := range tc.expectedStatus.Conditions {
				updated := meta.FindStatusCondition(updatedPlex.Status.Conditions, c.Type)
				test.NotNil(updated, "condition %s not found", c.Type)
				if updated == nil {
					continue
				}
				test.Equal(c.Status, updated.Status, "condition statuses for %s are not equal", c.Type)
				test.Equal(c.Reason, updated.Reason, "condition reasons for %s are not equal", c.Type)
				test.Equal(c.Message, updated.Message, "condition messages for %s are not equal", c.Type)
			}

		})
	}
}

func TestStatusSuite(t *testing.T) {
	suite.Run(t, new(statusReconcileSuite))
}
