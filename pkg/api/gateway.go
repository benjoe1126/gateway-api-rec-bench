package api

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var GatewayGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "gateways",
}

type GatewayApi struct {
	client *dynamic.DynamicClient
}

func NewGatewayApi(client *dynamic.DynamicClient) *GatewayApi {
	return &GatewayApi{client: client}
}

func (g *GatewayApi) Get(ctx context.Context, name, namespace string) (GWV1Resource, error) {
	unstruct, err := g.client.Resource(GatewayGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	gateway := &gatewayv1.Gateway{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.UnstructuredContent(), gateway)
	if err != nil {
		return nil, err
	}
	return gateway, nil
}

func (g *GatewayApi) List(ctx context.Context, namespace string) ([]GWV1Resource, error) {
	list, err := g.client.Resource(GatewayGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	gateways := make([]GWV1Resource, 0, len(list.Items))
	for _, item := range list.Items {
		gateway := &gatewayv1.Gateway{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, gateway)
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}
	return gateways, nil
}
func (g *GatewayApi) Create(ctx context.Context, gateway GWV1Resource) error {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(gateway)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: unstruct}
	if _, err = g.client.Resource(GatewayGVR).Namespace(gateway.GetNamespace()).Create(ctx, u, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (g *GatewayApi) Update(ctx context.Context, gateway GWV1Resource) error {
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(gateway)
	if err != nil {
		return err
	}
	u := &unstructured.Unstructured{Object: unstruct}
	if _, err = g.client.Resource(GatewayGVR).Namespace(gateway.GetNamespace()).Update(ctx, u, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (g *GatewayApi) Delete(ctx context.Context, name, namespace string) error {
	return g.client.Resource(GatewayGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
