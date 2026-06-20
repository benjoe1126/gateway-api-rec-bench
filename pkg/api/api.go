package api

import (
	"context"
	"encoding/json"
	"onlab-bm/pkg/patch"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayV1Resource interface {
	*gatewayv1.Gateway | *gatewayv1.GatewayClass | *gatewayv1.HTTPRoute | *gatewayv1.GRPCRoute | *v1.Service
	GetNamespace() string
	GetName() string
}

type K8SGWAPIKind string

type K8STypedAPI[T GatewayV1Resource] interface {
	Get(ctx context.Context, name, namespace string) (T, error)
	List(ctx context.Context, namespace string) ([]T, error)
	Create(ctx context.Context, res T) error
	Patch(ctx context.Context, name, namespace string, patches []patch.JsonPatch) error
	Delete(ctx context.Context, name, namespace string) error
}

type k8sTypedApi[T GatewayV1Resource] struct {
	client *dynamic.DynamicClient
	gvk    schema.GroupVersionResource
}

func (k k8sTypedApi[T]) Get(ctx context.Context, name, namespace string) (T, error) {
	unstruct, err := k.client.Resource(GatewayGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var resource T
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), resource)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

func (k k8sTypedApi[T]) List(ctx context.Context, namespace string) ([]T, error) {
	list, err := k.client.Resource(GatewayGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	objs := make([]T, 0, len(list.Items))
	for _, item := range list.Items {
		var res T
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, res)
		if err != nil {
			return nil, err
		}
		objs = append(objs, res)
	}
	return objs, nil
}

func (k k8sTypedApi[T]) Create(ctx context.Context, res T) error {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(res)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: unstruct}
	if _, err = k.client.Resource(GatewayGVR).Namespace(res.GetNamespace()).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (k k8sTypedApi[T]) Patch(ctx context.Context, name, namespace string, patches []patch.JsonPatch) error {
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return err
	}
	_, err = k.client.Resource(GatewayGVR).Namespace(namespace).Patch(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (k k8sTypedApi[T]) Delete(ctx context.Context, name, namespace string) error {
	return k.client.Resource(GatewayGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func NewTypedAPI[T GatewayV1Resource](client *dynamic.DynamicClient, gvk schema.GroupVersionResource) K8STypedAPI[T] {
	return k8sTypedApi[T]{
		client: client,
		gvk:    gvk,
	}
}
