package suite

import (
	"context"
	"fmt"
	"onlab-bm/pkg/api"
	"onlab-bm/pkg/patch"
	"strings"

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

type DeltaHooks func(...any)

type DeltaOpts func(d *Delta)

func WithPatches(patches ...patch.JsonPatch) DeltaOpts {
	return func(d *Delta) {
		d.patches = patches
	}
}

func WithHooks(hooks ...DeltaHooks) DeltaOpts {
	return func(d *Delta) {
		d.hooks = append(d.hooks, hooks...)
	}
}

type DeltaInterface interface {
	Apply(ctx context.Context) error
	UnderlyingType() DeltaUnderlyingType
	String() string
	Operation() DeltaOperation
}

type CompositeDelta struct {
	deltas []DeltaInterface
}

func NewCompositeDelta(deltas ...DeltaInterface) *CompositeDelta {
	return &CompositeDelta{deltas}
}
func (d *CompositeDelta) Operation() DeltaOperation {
	return d.deltas[0].Operation()
}

func (d *CompositeDelta) String() string {
	res := make([]string, 0, len(d.deltas))
	for _, ds := range d.deltas {
		res = append(res, ds.String())
	}
	return strings.Join(res, "\n")
}
func (d *CompositeDelta) Apply(ctx context.Context) error {
	for _, ds := range d.deltas {
		if err := ds.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (d *CompositeDelta) UnderlyingType() DeltaUnderlyingType {
	return d.deltas[0].UnderlyingType()
}

type Delta struct {
	op          DeltaOperation
	resourceApi api.GWV1Api
	resource    api.GWV1Resource
	patches     []patch.JsonPatch
	hooks       []DeltaHooks
}

func (d *Delta) Operation() DeltaOperation {
	return d.op
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
	defer func() {
		for _, h := range d.hooks {
			h()
		}
	}()
	switch d.op {
	case DeltaOpAdd:
		return d.resourceApi.Create(ctx, d.resource)
	case DeltaOpDelete:
		return d.resourceApi.Delete(ctx, name, namespace)
	case DeltaOpModify:
		return d.resourceApi.Patch(ctx, name, namespace, d.patches)
	default:
		return fmt.Errorf("unknown delta operation: %s", d.op)
	}
}

func NewDelta(op DeltaOperation, rapi api.GWV1Api, resource api.GWV1Resource, opts ...DeltaOpts) *Delta {
	ret := &Delta{
		op:          op,
		resourceApi: rapi,
		resource:    resource,
		patches:     nil,
		hooks:       nil,
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}
