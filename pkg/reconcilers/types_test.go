package reconcilers

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type errorClient struct {
	client.Client
	errCreate error
	errUpdate error
}

func (e *errorClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	if e.errCreate != nil {
		return e.errCreate
	}
	return e.Client.Get(ctx, key, obj)
}

func (e *errorClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if e.errUpdate != nil {
		return e.errUpdate
	}
	return e.Client.Update(ctx, obj, opts...)
}
