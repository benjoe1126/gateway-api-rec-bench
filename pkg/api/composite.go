package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type CompositeApi interface {
	Gateway() K8STypedAPI[*gatewayv1.Gateway]
	GatewayClass() K8STypedAPI[*gatewayv1.GatewayClass]
	HTTPRoute() K8STypedAPI[*gatewayv1.HTTPRoute]
	Service() K8STypedAPI[*v1.Service]
	GRPCRoute() K8STypedAPI[*gatewayv1.GRPCRoute]
}

type compositeApi struct {
	gw        K8STypedAPI[*gatewayv1.Gateway]
	gwc       K8STypedAPI[*gatewayv1.GatewayClass]
	httproute K8STypedAPI[*gatewayv1.HTTPRoute]
	svc       K8STypedAPI[*v1.Service]
	grpc      K8STypedAPI[*gatewayv1.GRPCRoute]
}

func (c compositeApi) Gateway() K8STypedAPI[*gatewayv1.Gateway] {
	return c.gw
}

func (c compositeApi) GatewayClass() K8STypedAPI[*gatewayv1.GatewayClass] {
	return c.gwc
}

func (c compositeApi) HTTPRoute() K8STypedAPI[*gatewayv1.HTTPRoute] {
	return c.httproute
}

func (c compositeApi) Service() K8STypedAPI[*v1.Service] {
	return c.svc
}

func (c compositeApi) GRPCRoute() K8STypedAPI[*gatewayv1.GRPCRoute] {
	return c.grpc
}

func NewCompositeApi(client *dynamic.DynamicClient) CompositeApi {
	return &compositeApi{
		gw:        NewTypedAPI[*gatewayv1.Gateway](client, GatewayGVR),
		gwc:       NewTypedAPI[*gatewayv1.GatewayClass](client, GatewayClassGVR),
		httproute: NewTypedAPI[*gatewayv1.HTTPRoute](client, HTTPRouteGVR),
		svc:       NewTypedAPI[*v1.Service](client, ServiceGVR),
		grpc:      NewTypedAPI[*gatewayv1.GRPCRoute](client, GRPCRouteGVR),
	}
}
