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

var GatewayClassGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "gatewayclasses",
}

type GatewayClassApi struct {
	client *dynamic.DynamicClient
}

func NewGatewayClassApi(dc *dynamic.DynamicClient) *GatewayClassApi {
	return &GatewayClassApi{client: dc}
}

func (g *GatewayClassApi) Get(ctx context.Context, name, _ string) (GWV1Resource, error) {
	unstruct, err := g.client.Resource(GatewayClassGVR).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	gateway := &gatewayv1.GatewayClass{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), gateway)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

func (g *GatewayClassApi) List(ctx context.Context, _ string) ([]GWV1Resource, error) {
	list, err := g.client.Resource(GatewayClassGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	gateways := make([]GWV1Resource, 0, len(list.Items))
	for _, item := range list.Items {
		gateway := &gatewayv1.GatewayClass{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, gateway)
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}
	return gateways, nil
}

func (g *GatewayClassApi) Create(ctx context.Context, gateway GWV1Resource) error {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(gateway)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: unstruct}
	if _, err = g.client.Resource(GatewayClassGVR).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (g *GatewayClassApi) Patch(ctx context.Context, name, _ string, patches []patch.JsonPatch) error {
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return err
	}
	_, err = g.client.Resource(GatewayClassGVR).Patch(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (g *GatewayClassApi) Delete(ctx context.Context, name, _ string) error {
	return g.client.Resource(GatewayClassGVR).Delete(ctx, name, metav1.DeleteOptions{})
}
