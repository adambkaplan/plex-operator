package reconcilers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

type serviceTestCase struct {
	name            string
	plex            *v1alpha1.PlexMediaServer
	existingService *corev1.Service
	expectedService *corev1.Service
	expectError     bool
	expectRequeue   bool
}

type serviceReconcileSuite struct {
	suite.Suite
	cases []serviceTestCase
}

func (test *serviceReconcileSuite) SetupTest() {
	test.cases = []serviceTestCase{
		{
			name: "create with defaults",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "test",
				},
			},
			expectedService: mockService("test", "test"),
			expectRequeue:   true,
		},
		{
			name: "no change",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "no-change",
					Name:      "no-change",
				},
			},
			existingService: mockService("no-change", "no-change"),
			expectedService: mockService("no-change", "no-change"),
		},
	}
}

func (test *serviceReconcileSuite) TestServiceReconcile() {
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
			client := builder.Build()
			reconciler := &ServiceReconciler{
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
			updatedService := &corev1.Service{}
			err = client.Get(ctx, types.NamespacedName{Namespace: tc.plex.Namespace, Name: tc.plex.Name}, updatedService)
			test.Require().NoError(err, "failed to get Service")
			test.True(equality.Semantic.DeepEqual(tc.expectedService.Spec, updatedService.Spec),
				"expected service\n\n%s\n\ndoes not match\n\n%s",
				tc.expectedService.Spec,
				updatedService.Spec)

		})
	}
}

func mockService(namespace, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"plex.adambkaplan.com/instance": name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "plex",
					Port:     32400,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

}
