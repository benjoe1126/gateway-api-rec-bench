package api

import (
	"context"
	"encoding/json"
	"onlab-bm/pkg/patch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var HTTPRouteGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "httproutes",
}

type HttpRouteApi struct {
	client *dynamic.DynamicClient
}

func NewHttpRouteApi(client *dynamic.DynamicClient) *HttpRouteApi {
	return &HttpRouteApi{client: client}
}

func (h *HttpRouteApi) Get(ctx context.Context, name, namespace string) (GWV1Resource, error) {
	unstruct, err := h.client.Resource(HTTPRouteGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	route := &gatewayv1.HTTPRoute{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), route)
	if err != nil {
		return nil, err
	}
	return route, nil
}

func (h *HttpRouteApi) List(ctx context.Context, namespace string) ([]GWV1Resource, error) {
	list, err := h.client.Resource(HTTPRouteGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	routes := make([]GWV1Resource, 0, len(list.Items))
	for _, item := range list.Items {
		route := &gatewayv1.HTTPRoute{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, route)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, nil
}

func (h *HttpRouteApi) Create(ctx context.Context, route GWV1Resource) error {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(route)
	if err != nil {
		return err
	}

	u := &unstructured.Unstructured{Object: unstruct}
	if _, err = h.client.Resource(HTTPRouteGVR).Namespace(route.GetNamespace()).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (h *HttpRouteApi) Patch(ctx context.Context, name, namespace string, patches []patch.JsonPatch) error {
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return err
	}
	_, err = h.client.Resource(HTTPRouteGVR).Namespace(namespace).Patch(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (h *HttpRouteApi) Delete(ctx context.Context, name, namespace string) error {
	return h.client.Resource(HTTPRouteGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
