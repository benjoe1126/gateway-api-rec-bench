package api

import (
	"context"
	"onlab-bm/pkg/patch"
)

type GWV1Resource interface {
	GetNamespace() string
	GetName() string
}

type GWV1Api interface {
	Get(ctx context.Context, name, namespace string) (GWV1Resource, error)
	List(ctx context.Context, namespace string) ([]GWV1Resource, error)
	Create(ctx context.Context, res GWV1Resource) error
	Patch(ctx context.Context, name, namespace string, patches []patch.JsonPatch) error
	Delete(ctx context.Context, name, namespace string) error
}
