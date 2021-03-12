package reconcilers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/suite"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

type externalServiceReconcileSuite struct {
	suite.Suite
	cases []serviceTestCase
}

func (test *externalServiceReconcileSuite) SetupTest() {
	test.cases = []serviceTestCase{
		{
			name: "none with no existing service",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "none",
					Name:      "none",
				},
			},
			expectRequeue: false,
		},
		{
			name: "none with existing LoadBalancer service",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "none",
					Name:      "none-lb",
				},
			},
			existingService: serviceDouble("none", "none-lb", serviceDoubleOptions{
				ServiceName: "none-lb-ext",
				ServiceType: corev1.ServiceTypeLoadBalancer,
			}),
			expectRequeue: true,
		},
		{
			name: "none with existing NodePort service",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "none",
					Name:      "none-np",
				},
			},
			existingService: serviceDouble("none", "none-np", serviceDoubleOptions{
				ServiceName: "none-np-ext",
				ServiceType: corev1.ServiceTypeNodePort,
			}),
			expectRequeue: true,
		},
		{
			name: "create LoadBalancer service",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "create",
					Name:      "create-lb",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeLoadBalancer,
					},
				},
			},
			expectedService: serviceDouble("create", "create-lb", serviceDoubleOptions{
				ServiceName: "create-lb-ext",
				ServiceType: corev1.ServiceTypeLoadBalancer,
			}),
			expectRequeue: true,
		},
		{
			name: "create NodePort service",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "create",
					Name:      "create-np",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeNodePort,
					},
				},
			},
			expectedService: serviceDouble("create", "create-np", serviceDoubleOptions{
				ServiceName: "create-np-ext",
				ServiceType: corev1.ServiceTypeNodePort,
			}),
			expectRequeue: true,
		},
		{
			name: "transition LoadBalancer to NodePort",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "update",
					Name:      "update-np",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeNodePort,
					},
				},
			},
			existingService: serviceDouble("update", "update-np", serviceDoubleOptions{
				ServiceName: "update-np-ext",
				ServiceType: corev1.ServiceTypeLoadBalancer,
			}),
			expectedService: serviceDouble("update", "update-np", serviceDoubleOptions{
				ServiceName: "update-np-ext",
				ServiceType: corev1.ServiceTypeNodePort,
			}),
			expectRequeue: true,
		},
		{
			name: "transition NodePort to LoadBalancer",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "create",
					Name:      "update-lb",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeLoadBalancer,
					},
				},
			},
			existingService: serviceDouble("update", "update-lb", serviceDoubleOptions{
				ServiceName: "update-lb-ext",
				ServiceType: corev1.ServiceTypeNodePort,
			}),
			expectedService: serviceDouble("create", "update-lb", serviceDoubleOptions{
				ServiceName: "update-lb-ext",
				ServiceType: corev1.ServiceTypeLoadBalancer,
			}),
			expectRequeue: true,
		},
		{
			name: "no change with LoadBalancer",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "update",
					Name:      "update-lb",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeLoadBalancer,
					},
				},
			},
			existingService: serviceDouble("update", "update-lb", serviceDoubleOptions{
				ServiceName: "update-lb-ext",
				ServiceType: corev1.ServiceTypeLoadBalancer,
			}),
			expectedService: serviceDouble("update", "update-lb", serviceDoubleOptions{
				ServiceName: "update-lb-ext",
				ServiceType: corev1.ServiceTypeLoadBalancer,
			}),
		},
		{
			name: "no change with NodePort",
			plex: &v1alpha1.PlexMediaServer{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "update",
					Name:      "update-np",
				},
				Spec: v1alpha1.PlexMediaServerSpec{
					Networking: v1alpha1.PlexNetworkSpec{
						ExternalServiceType: corev1.ServiceTypeNodePort,
					},
				},
			},
			existingService: serviceDouble("update", "update-np", serviceDoubleOptions{
				ServiceName: "update-np-ext",
				ServiceType: corev1.ServiceTypeNodePort,
			}),
			expectedService: serviceDouble("update", "update-np", serviceDoubleOptions{
				ServiceName: "update-np-ext",
				ServiceType: corev1.ServiceTypeNodePort,
			}),
		},
	}
}

func (test *externalServiceReconcileSuite) TestExternalService() {
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
			if tc.existingService != nil {
				builder.WithObjects(tc.existingService)
			}
			client := builder.Build()
			reconciler := &ExternalServiceReconciler{
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
			test.Require().NoError(err, "unexpected error from reconcile")
			updatedService := &corev1.Service{}
			serviceName := fmt.Sprintf("%s-ext", tc.plex.Name)
			err = client.Get(ctx, types.NamespacedName{Namespace: tc.plex.Namespace, Name: serviceName}, updatedService)
			if tc.expectedService == nil {
				test.Assert().True(errors.IsNotFound(err))
			} else {
				test.Require().NoError(err, "failed to get Service")
				test.True(equality.Semantic.DeepEqual(tc.expectedService.Spec, updatedService.Spec),
					"expected service does not match - diff: %s",
					cmp.Diff(tc.expectedService.Spec, updatedService.Spec))
			}
		})
	}
}

func TestExternalServiceSuite(t *testing.T) {
	suite.Run(t, new(externalServiceReconcileSuite))
}
