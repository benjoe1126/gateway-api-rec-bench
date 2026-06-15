package scenario

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"maps"
	"math"
	"onlab-bm/pkg/api"
	"slices"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type ScenarioGenerator struct {
	ScenarioTemplate string                 `json:"scenarioTemplate" yaml:"scenarioTemplate"`
	Values           map[string]interface{} `json:"values" yaml:"values"`
}

func (s *ScenarioGenerator) Generate() ([]Scenario, error) {
	tpl := template.New("scenarioGenerator")
	tpl, err := tpl.Parse(string(s.ScenarioTemplate))
	if err != nil {
		return nil, fmt.Errorf("failed to parse scenario template: %w", err)
	}
	var scenarios []Scenario
	var rendered bytes.Buffer
	if err := tpl.Execute(&rendered, s.Values); err != nil {
		return nil, fmt.Errorf("failed to render scenario template: %w", err)
	}
	log.Println(rendered.String())
	if err := yaml.Unmarshal(rendered.Bytes(), &scenarios); err != nil {
		var scenario Scenario
		if err := yaml.Unmarshal(rendered.Bytes(), &scenario); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scenario template: %w", err)
		}
		scenarios = append(scenarios, scenario)
	}
	fmt.Println(scenarios[0].Description)
	return scenarios, nil
}

type ScenarioTemplate struct {
	Name             string `json:"name" yaml:"name"`
	Description      string `json:"description" yaml:"description"`
	ResourceTemplate string `json:"resourceTemplate" yaml:"resourceTemplate"`
	DeltasTemplate   string `json:"deltasTemplate" yaml:"deltasTemplate"`
	InitialDelay     string `json:"initialDelay" yaml:"initialDelay"`
}

type Scenario struct {
	Name         string          `yaml:"name" json:"name"`
	Description  string          `yaml:"description,omitempty" json:"description,omitempty"`
	Resources    Resources       `yaml:"resources" json:"resources"`
	Deltas       []ResourceDelta `yaml:"deltas" json:"deltas"`
	InitialDelay time.Duration   `yaml:"initialDelay,omitempty" json:"initialDelay,omitempty"`
}

var (
	gatewayClassDescription = "Base gateway class for measurements"
	httPort                 = gatewayv1.PortNumber(80)
	baseGatewayClass        = gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "base-gatewayclass",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "GatewayClass",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.envoyproxy.io/gatewayclass-controller",
			Description:    &gatewayClassDescription,
		},
	}
	baseGateway = gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "base-gateway",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "Gateway",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(baseGatewayClass.Name),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}
)

// init sets up the base resources associated with the scenario, it does not start the delta application process
// ideally it should be left up to Apply to be called
func (s *Scenario) init(capi *api.CompositeApi, ctx context.Context, logger slog.Logger) error {
	var err error
	logger.Log(ctx, slog.LevelInfo, "started initializing resources of scenario", slog.String("scenario", s.Name))
	if err := s.Resources.GatewayClasses.Validate(); err != nil {
		return fmt.Errorf("invalid gateway class resource definition: %w", err)
	}
	logger.Log(ctx, slog.LevelInfo, "creating gateway classes")
	gwclassNamingScheme := s.Resources.GatewayClasses.NamingScheme
	if gwclassNamingScheme == "" {
		gwclassNamingScheme = "default-gatewayclass"
	}
	gwclasses := make(map[string]*gatewayv1.GatewayClass)
	gateways := make(map[string]*gatewayv1.Gateway)
	httpRoutes := make(map[string]*gatewayv1.HTTPRoute)
	grpcRoutes := make(map[string]*gatewayv1.GRPCRoute)
	services := make(map[string]*v1.Service)
	defer func() {
		if err != nil {
			logger.Log(ctx, slog.LevelInfo, "failed to initialize all resources, deleting dangling gateway classes...")
			for k, _ := range gwclasses {
				if err2 := capi.GatewayClass().Delete(ctx, k, ""); err2 != nil {
					logger.Log(ctx, slog.LevelInfo, "failed to delete gateway class %s", k)
				}
			}
			logger.Log(ctx, slog.LevelInfo, "deleting dangling gateways")
			for k, _ := range gateways {
				if err2 := capi.Gateway().Delete(ctx, k, ""); err2 != nil {
					logger.Log(ctx, slog.LevelInfo, "failed to delete gateway class %s", k)
				}
			}
			logger.Log(ctx, slog.LevelInfo, "deleting dangling http routes")
			for k, _ := range httpRoutes {
				if err2 := capi.HTTPRoute().Delete(ctx, k, ""); err2 != nil {
				}
			}
			logger.Log(ctx, slog.LevelInfo, "deleting dangling grpc routes")
			for k, _ := range grpcRoutes {
				if err2 := capi.GRPCRoute().Delete(ctx, k, ""); err2 != nil {
					logger.Log(ctx, slog.LevelInfo, "failed to delete gateway class %s", k)
				}
			}
			logger.Log(ctx, slog.LevelInfo, "deleting dangling services")
			for k, _ := range services {
				if err2 := capi.Service().Delete(ctx, k, ""); err2 != nil {
					logger.Log(ctx, slog.LevelInfo, "failed to delete gateway class %s", k)
				}
			}
		}
	}()
	for i := range s.Resources.GatewayClasses.Count {
		gwclass := baseGatewayClass.DeepCopy()
		gwclass.Name = fmt.Sprintf("%s-%d", s.Resources.GatewayClasses.NamingScheme, i)
		gwclasses[gwclass.ObjectMeta.Name] = gwclass
		if err := capi.GatewayClass().Create(ctx, gwclass); err != nil {
			return fmt.Errorf("failed to create gateway class %s: %w", gwclass.Name, err)
		}
	}
	gwNamingScheme := s.Resources.Gateways.NamingScheme
	if gwNamingScheme == "" {
		gwNamingScheme = "default-gateway"
	}
	distribtion := s.Resources.Gateways.Distribution
	countPerGwclass := make(map[string]int)
	for k, _ := range gwclasses {
		countPerGwclass[k] = 0
	}

	if s.Resources.Gateways.Count > 0 {
		logger.Log(ctx, slog.LevelInfo, "creating gateways")
		if err := s.Resources.Gateways.Validate(); err != nil {
			return fmt.Errorf("invalid gateway resource definition: %w", err)
		}
		gwClassName := ""
		for i := range s.Resources.Gateways.Count {
			gw := baseGateway.DeepCopy()
			gw.Name = fmt.Sprintf("%s-%d", s.Resources.Gateways.NamingScheme, i)
			switch distribtion.Type {
			case Uniform:
				gwClassName = gwclassWithLeastGateways(countPerGwclass)
			case Random:
				gwClassName = randomGWClass(gwclasses)
			case Weighted:
				gwClassName = weightedGWClass(s.Resources.Gateways.Distribution.Targets)
			}
			gw.Spec.GatewayClassName = gatewayv1.ObjectName(gwClassName)
		}
	}
	return nil
}
func gwclassWithLeastGateways(gatewayclassCounts map[string]int) string {
	minValue := math.MaxInt32
	minName := ""
	for k, v := range gatewayclassCounts {
		if v < minValue {
			minValue = v
			minName = k
		}
	}
	return minName
}
func randomGWClass(gwclasses map[string]*gatewayv1.GatewayClass) string {
	asSclie := slices.Sorted(maps.Keys(gwclasses))
	randomIndex := rand.Intn(len(asSclie))
	return asSclie[randomIndex]

}

func weightedGWClass(weights []Target) string {
	if len(weights) == 0 {
		return ""
	}

	totalWeight := 0
	for _, w := range weights {
		totalWeight += int(w.Weight)
	}

	if totalWeight == 0 {
		return ""
	}

	r := rand.Intn(totalWeight)

	cumulative := 0
	for _, w := range weights {
		cumulative += int(w.Weight)
		if r < cumulative {
			return w.ParentName
		}
	}

	return ""
}
