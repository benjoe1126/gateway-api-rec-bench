package suite

import (
	"context"
	"fmt"
	"onlab-bm/pkg/api"

	v1 "k8s.io/api/core/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type DeltaOperation int

const (
	DeltaOpAdd DeltaOperation = iota
	DeltaOpDelete
	DeltaOpModify
)

func (dop DeltaOperation) String() string {
	switch dop {
	case DeltaOpAdd:
		return "add: "
	case DeltaOpDelete:
		return "delete: "
	case DeltaOpModify:
		return "modify: "
	default:
		return " "
	}
}

type Delta struct {
	op          DeltaOperation
	resourceApi api.GWV1Api
	resource    api.GWV1Resource
}

func (d *Delta) String() string {
	ret := d.op.String()
	switch d.resource.(type) {
	case *gatewayv1.Gateway:
		ret += "Gateway"
	case *gatewayv1.GatewayClass:
		ret += "GatewayClass"
	case *gatewayv1.HTTPRoute:
		ret += "HTTPRoute"
	case *v1.Service:
		ret += "Service"
	default:
		return ""
	}
	return ret + " Name: " + d.resource.GetName()
}

type DeltaUnderlyingType int

const (
	DeltaUnderlyingTypeGateway DeltaUnderlyingType = iota
	DeltaUnderlyingTypeGatewayClass
	DeltaUnderlyingTypeHTTPRoute
	DeltaUnderlyingTypeService
	DeltaUnderlyingTypeUnknown
)

func (d *Delta) UnderlyingType() DeltaUnderlyingType {
	switch d.resource.(type) {
	case *gatewayv1.Gateway:
		return DeltaUnderlyingTypeGateway
	case *gatewayv1.GatewayClass:
		return DeltaUnderlyingTypeGatewayClass
	case *gatewayv1.HTTPRoute:
		return DeltaUnderlyingTypeHTTPRoute
	case *v1.Service:
		return DeltaUnderlyingTypeService
	}
	return DeltaUnderlyingTypeUnknown
}

func (d *Delta) Apply(ctx context.Context) error {
	name := d.resource.GetName()
	namespace := d.resource.GetNamespace()
	switch d.op {
	case DeltaOpAdd:
		return d.resourceApi.Create(ctx, d.resource)
	case DeltaOpDelete:
		return d.resourceApi.Delete(ctx, name, namespace)
	case DeltaOpModify:
		return d.resourceApi.Update(ctx, d.resource)
	default:
		return fmt.Errorf("unknown delta operation: %s", d.op)
	}
}

func NewDelta(op DeltaOperation, rapi api.GWV1Api, resource api.GWV1Resource) *Delta {
	return &Delta{
		op:          op,
		resourceApi: rapi,
		resource:    resource,
	}
}
