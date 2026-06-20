package test

import (
	"context"
	_ "embed"
	"log/slog"
	"onlab-bm/pkg/api"
	"onlab-bm/pkg/patch"
	"onlab-bm/pkg/scenario"
	"reflect"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

//go:embed resources/simple_scenario.yaml
var simpleScenario []byte

func TestSimpleScenarioParsing(t *testing.T) {
	const (
		scenarioName             = "simple-scenario"
		description              = "some description"
		gatewayClassCount        = 1
		gatewayClassNamingScheme = "gwclass"
		gatewayCount             = 3
		gatewayNamingScheme      = "simple-gateway"
		gatewayDistribution      = scenario.Uniform
		selectorKind             = "Gateway"
		selectorName             = "simple-gateway-1"
	)
	var (
		jpatch = patch.JsonPatch{
			Op:    patch.PatchOpReplace,
			Path:  "/metadata/name",
			Value: "gateway",
		}
	)
	var sc scenario.Scenario
	err := yaml.Unmarshal(simpleScenario, &sc)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Name != scenarioName {
		t.Fatalf("scenario name should be %s, got %s", scenarioName, sc.Name)
	}
	if sc.Description != description {
		t.Fatalf("scenario description should be %s, got %s", description, sc.Description)
	}
	if sc.Resources.GatewayClasses.NamingScheme != gatewayClassNamingScheme {
		t.Fatalf("gatewayclass naming scheme should be %s, got %s", gatewayClassNamingScheme, sc.Resources.GatewayClasses.NamingScheme)
	}
	if sc.Resources.GatewayClasses.Count != gatewayClassCount {
		t.Fatalf("gatewayclass count should be %d, got %d", gatewayClassCount, sc.Resources.GatewayClasses.Count)
	}
	if sc.Resources.Gateways.NamingScheme != gatewayNamingScheme {
		t.Fatalf("gateway naming scheme should be %s, got %s", gatewayNamingScheme, sc.Resources.Gateways.NamingScheme)
	}
	if sc.Resources.Gateways.Count != gatewayCount {
		t.Fatalf("gateway count should be %d, got %d", gatewayCount, sc.Resources.Gateways.Count)
	}
	if sc.Resources.Gateways.Distribution.Type != gatewayDistribution {
		t.Fatalf("gateway distribution should be %s, got %s", gatewayDistribution, sc.Resources.Gateways.Distribution.Type)
	}
	if len(sc.Deltas) != 1 {
		t.Fatalf("delta count should be %d, got %d", 1, len(sc.Deltas))
	}
	if len(sc.Deltas[0].Patches) != 1 {
		t.Fatalf("delta patch count should be %d, got %d", 1, len(sc.Deltas[0].Patches))
	}
	if sc.Deltas[0].Patches[0] != jpatch {
		t.Fatalf("json patch should be %v, got %v", jpatch, sc.Deltas[0].Patches[0])
	}
	if sc.Deltas[0].ResourceSelector.Kind != selectorKind {
		t.Fatalf("patch selector kind is %s, got %s", selectorKind, sc.Deltas[0].ResourceSelector.Kind)
	}
	if sc.Deltas[0].ResourceSelector.Name != selectorName {
		t.Fatalf("patch selector name is %s, got %s", selectorName, sc.Deltas[0].ResourceSelector.Name)
	}
}

//go:embed resources/scenario_generator.yaml
var scenarioGenerator []byte

func TestScenarioGeneration(t *testing.T) {
	var generator scenario.ScenarioGenerator
	err := yaml.Unmarshal(scenarioGenerator, &generator)
	if err != nil {
		t.Fatal(err)
	}
	//they share the same delta, so instaniate only once
	deltas := []scenario.ResourceDelta{
		{
			ResourceSelector: scenario.ResourceSelector{
				Kind:      scenario.ResourceKindGateway,
				Name:      "simple-gateway-1",
				Namespace: "gateway-system",
			},
			Patches: []patch.JsonPatch{
				{
					Op:    patch.PatchOpReplace,
					Path:  "/metadata/name",
					Value: "gateway",
				},
			},
		},
	}
	expectedScenarios := []scenario.Scenario{
		{
			Name:        "simple-scenario-1",
			Description: "some description 1",
			Resources: scenario.Resources{
				GatewayClasses: scenario.GatewayClass{
					GenericResource: scenario.GenericResource{
						Count:        1,
						NamingScheme: "gwclass",
					},
				},
				Gateways: scenario.Gateway{
					GenericResourceWithDistribution: scenario.GenericResourceWithDistribution{
						Distribution: scenario.Distribution{
							Type: scenario.Uniform,
						},
						GenericResource: scenario.GenericResource{
							Count:        3,
							NamingScheme: "gateway",
						},
					},
				},
			},
			Deltas: deltas,
		},
		{
			Name:        "simple-scenario-2",
			Description: "some description 2",
			Resources: scenario.Resources{
				GatewayClasses: scenario.GatewayClass{
					GenericResource: scenario.GenericResource{
						Count:        1,
						NamingScheme: "gwclass-second",
					},
				},
				Gateways: scenario.Gateway{
					GenericResourceWithDistribution: scenario.GenericResourceWithDistribution{
						Distribution: scenario.Distribution{
							Type: scenario.Random,
						},
						GenericResource: scenario.GenericResource{
							Count:        6,
							NamingScheme: "gateway-second",
						},
					},
				},
			},
			Deltas: deltas,
		},
	}
	actualScenarios, err := generator.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(actualScenarios) != len(expectedScenarios) {
		t.Fatalf("scenario count should be %d, got %d", len(expectedScenarios), len(actualScenarios))
	}
	sfunc := func(a scenario.Scenario, b scenario.Scenario) int {
		if a.Name > b.Name {
			return 1
		}
		return -1
	}
	slices.SortFunc(expectedScenarios, sfunc)
	slices.SortFunc(actualScenarios, sfunc)
	for i, _ := range expectedScenarios {
		if !reflect.DeepEqual(actualScenarios[i], expectedScenarios[i]) {
			t.Errorf("scenario %d: expected %v, got %v", i, expectedScenarios[i], actualScenarios[i])
			t.Log(diff.Diff(expectedScenarios[i], actualScenarios[i]))
		}
	}
}

// builds its resource tree internally, so testing is plausable
type mockApi[T api.GatewayV1Resource] struct {
	resourceTree map[string]T
}

func (m *mockApi[T]) Get(ctx context.Context, name, namespace string) (T, error) {
	return nil, nil
}

func (m *mockApi[T]) List(ctx context.Context, namespace string) ([]T, error) {
	return nil, nil
}

func (m *mockApi[T]) Create(ctx context.Context, res T) error {
	m.resourceTree[res.GetName()] = res
	return nil
}

func (m *mockApi[T]) Patch(ctx context.Context, name, namespace string, patches []patch.JsonPatch) error {
	return nil
}

func (m *mockApi[T]) Delete(ctx context.Context, name, namespace string) error {
	return nil
}

type mockCompositeApi struct {
	gw        mockApi[*gatewayv1.Gateway]
	gwc       mockApi[*gatewayv1.GatewayClass]
	httproute mockApi[*gatewayv1.HTTPRoute]
	svc       mockApi[*v1.Service]
	grpc      mockApi[*gatewayv1.GRPCRoute]
}

func (m mockCompositeApi) Gateway() api.K8STypedAPI[*gatewayv1.Gateway] {
	return &m.gw
}

func (m mockCompositeApi) GatewayClass() api.K8STypedAPI[*gatewayv1.GatewayClass] {
	return &m.gwc
}

func (m mockCompositeApi) HTTPRoute() api.K8STypedAPI[*gatewayv1.HTTPRoute] {
	return &m.httproute
}

func (m mockCompositeApi) Service() api.K8STypedAPI[*v1.Service] {
	return &m.svc
}

func (m mockCompositeApi) GRPCRoute() api.K8STypedAPI[*gatewayv1.GRPCRoute] {
	return &m.grpc
}

func TestScenarioInit(t *testing.T) {
	var sc scenario.Scenario
	capi := mockCompositeApi{
		gw: mockApi[*gatewayv1.Gateway]{
			resourceTree: make(map[string]*gatewayv1.Gateway),
		},
		gwc: mockApi[*gatewayv1.GatewayClass]{
			resourceTree: make(map[string]*gatewayv1.GatewayClass),
		},
		httproute: mockApi[*gatewayv1.HTTPRoute]{
			resourceTree: make(map[string]*gatewayv1.HTTPRoute),
		},
		svc: mockApi[*v1.Service]{
			resourceTree: make(map[string]*v1.Service),
		},
		grpc: mockApi[*gatewayv1.GRPCRoute]{
			resourceTree: make(map[string]*gatewayv1.GRPCRoute),
		},
	}
	err := yaml.Unmarshal(simpleScenario, &sc)
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.Init(context.Background(), slog.New(slog.DiscardHandler), &capi); err != nil {
		t.Fatalf("scenario initialization failed: %v", err)
	}
	for k, v := range capi.gw.resourceTree {
		t.Log(k, v.Spec.GatewayClassName)
	}
}
