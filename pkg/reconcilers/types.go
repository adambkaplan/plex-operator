package reconcilers

import (
	"context"

	"github.com/adambkaplan/plex-operator/api/v1alpha1"
)

// Reconciler reconciles the desired state of a managed object with the object's state in the cluster
type Reconciler interface {
	Reconcile(ctx context.Context, plex *v1alpha1.PlexMediaServer) (bool, error)
}
