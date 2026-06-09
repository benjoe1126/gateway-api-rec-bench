package test

import (
	_ "embed"
	"onlab-bm/pkg/scenario"
	"testing"

	"gopkg.in/yaml.v3"
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
		gatweayDistribution      = scenario.Uniform
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
	if sc.Resources.Gateways.Distribution != gatweayDistribution {
		t.Fatalf("gateway distribution should be %s, got %s", gatweayDistribution, sc.Resources.Gateways.Distribution)
	}
	t.Log(sc.Deltas)
}
