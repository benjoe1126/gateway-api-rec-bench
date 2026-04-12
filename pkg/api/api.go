package api

import (
	"context"
)

type GWV1Resource interface {
	GetNamespace() string
	GetName() string
}

type GWV1Api interface {
	Get(ctx context.Context, name, namespace string) (GWV1Resource, error)
	List(ctx context.Context, namespace string) ([]GWV1Resource, error)
	Create(ctx context.Context, res GWV1Resource) error
	Update(ctx context.Context, res GWV1Resource) error
	Delete(ctx context.Context, name, namespace string) error
}
