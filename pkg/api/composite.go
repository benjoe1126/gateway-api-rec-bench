package api

import "k8s.io/client-go/dynamic"

type CompositeApi struct {
	gw    *GatewayApi
	gwc   *GatewayClassApi
	route *HttpRouteApi
	svc   *ServiceApi
}

func NewCompositeApi(client *dynamic.DynamicClient) *CompositeApi {
	return &CompositeApi{
		gw:    NewGatewayApi(client),
		gwc:   NewGatewayClassApi(client),
		route: NewHttpRouteApi(client),
		svc:   NewServiceApi(client),
	}
}

func (api *CompositeApi) Gateway() *GatewayApi {
	return api.gw
}
func (api *CompositeApi) GatewayClass() *GatewayClassApi {
	return api.gwc
}
func (api *CompositeApi) HTTPRoute() *HttpRouteApi {
	return api.route
}
func (api *CompositeApi) Service() *ServiceApi {
	return api.svc
}
