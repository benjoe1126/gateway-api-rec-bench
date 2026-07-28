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
	"sigs.k8s.io/gateway-api/apis/v1beta1"
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
	baseHttpRoute = gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "base-http-route-%d",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "HTTPRoute",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{},
				},
			},
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{},
			},
		},
	}
	baseGRPCRoute = gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "GRPCRoute",
		},
		Spec: gatewayv1.GRPCRouteSpec{
			Hostnames: []gatewayv1.Hostname{},
			Rules: []gatewayv1.GRPCRouteRule{
				{
					BackendRefs: []gatewayv1.GRPCBackendRef{},
				},
			},
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name: "base-gateway-0",
					},
				},
			},
		},
	}
	baseService = v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "http-backend",
			Namespace: metav1.NamespaceDefault,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
		},
	}
)

func (s *Scenario) initGatewayClasses(ctx context.Context, logger *slog.Logger) (map[string]*gatewayv1.GatewayClass, error) {
	if err := s.Resources.GatewayClasses.Validate(); err != nil {
		return nil, fmt.Errorf("invalid gateway class resource definition: %w", err)
	}
	logger.Log(ctx, slog.LevelInfo, "creating gateway classes")
	gwclassNamingScheme := s.Resources.GatewayClasses.NamingScheme
	if gwclassNamingScheme == "" {
		gwclassNamingScheme = "default-gatewayclass"
	}
	gwclasses := make(map[string]*gatewayv1.GatewayClass)
	for i := range s.Resources.GatewayClasses.Count {
		gwclass := baseGatewayClass.DeepCopy()
		gwclass.Name = fmt.Sprintf("%s-%d", s.Resources.GatewayClasses.NamingScheme, i)
		gwclasses[gwclass.ObjectMeta.Name] = gwclass
	}
	return gwclasses, nil
}

func (s *Scenario) initGateways(ctx context.Context, logger *slog.Logger, gwclasses map[string]*gatewayv1.GatewayClass) (map[string]*gatewayv1.Gateway, error) {
	gateways := make(map[string]*gatewayv1.Gateway)
	gwNamingScheme := s.Resources.Gateways.NamingScheme
	if gwNamingScheme == "" {
		gwNamingScheme = "default-gateway"
	}
	distribution := s.Resources.Gateways.Distribution
	countPerGwclass := make(map[string]int)
	for k, _ := range gwclasses {
		countPerGwclass[k] = 0
	}

	if s.Resources.Gateways.Count > 0 {
		logger.Log(ctx, slog.LevelInfo, "creating gateways")
		if err := s.Resources.Gateways.Validate(); err != nil {
			return nil, fmt.Errorf("invalid gateway resource definition: %w", err)
		}
		gwClassName := ""
		for i := range s.Resources.Gateways.Count {
			gw := baseGateway.DeepCopy()
			gw.Name = fmt.Sprintf("%s-%d", s.Resources.Gateways.NamingScheme, i)
			switch distribution.Type {
			case Uniform:
				gwClassName = distributionWithLeastAttached(countPerGwclass)
			case Random:
				gwClassName = distributionRandom(gwclasses)
			case Weighted:
				gwClassName = distributionWithWeight(s.Resources.Gateways.Distribution.Targets)
			}
			gw.Spec.GatewayClassName = gatewayv1.ObjectName(gwClassName)
			gateways[gw.Name] = gw
		}
	}
	return gateways, nil
}
func (s *Scenario) initHTTPRoutes(ctx context.Context, logger *slog.Logger, gateways map[string]*gatewayv1.Gateway) (map[string]*gatewayv1.HTTPRoute, error) {
	httpRoutes := make(map[string]*gatewayv1.HTTPRoute)
	routeDistribution := s.Resources.HTTPRoutes.Distribution
	countPerGateway := make(map[string]int)
	for k, _ := range gateways {
		countPerGateway[k] = 0
	}
	if s.Resources.HTTPRoutes.Count > 0 {
		logger.Log(ctx, slog.LevelInfo, "creating http routes")
		if err := s.Resources.HTTPRoutes.Validate(); err != nil {
			return nil, fmt.Errorf("invalid http route resource definition: %w", err)
		}
		for i := range s.Resources.HTTPRoutes.Count {
			httproute := baseHttpRoute.DeepCopy()
			gwName := ""
			httproute.Name = fmt.Sprintf("%s-%d", s.Resources.HTTPRoutes.NamingScheme, i)
			switch routeDistribution.Type {
			case Uniform:
				gwName = distributionWithLeastAttached(countPerGateway)
			case Random:
				gwName = distributionRandom(gateways)
			case Weighted:
				gwName = distributionWithWeight(s.Resources.HTTPRoutes.Distribution.Targets)
			}
			kind := gatewayv1.Kind(baseGateway.Kind)
			ns := gatewayv1.Namespace(baseGateway.Namespace)
			httproute.Spec.ParentRefs = append(httproute.Spec.ParentRefs, gatewayv1.ParentReference{
				Name:      gatewayv1.ObjectName(gwName),
				Kind:      &kind,
				Namespace: &ns,
			})
			httpRoutes[httproute.Name] = httproute
		}
	}
	return httpRoutes, nil
}
func (s *Scenario) initGRPCRoutes(ctx context.Context, logger *slog.Logger, gateways map[string]*gatewayv1.Gateway) (map[string]*gatewayv1.GRPCRoute, error) {
	grpcRoutes := make(map[string]*gatewayv1.GRPCRoute)
	routeDistribution := s.Resources.GRPCRoutes.Distribution
	countPerGateway := make(map[string]int)
	for k, _ := range gateways {
		countPerGateway[k] = 0
	}
	if s.Resources.GRPCRoutes.Count > 0 {
		logger.Log(ctx, slog.LevelInfo, "creating grpc routes")
		if err := s.Resources.GRPCRoutes.Validate(); err != nil {
			return nil, fmt.Errorf("invalid grpc route resource definition: %w", err)
		}
		for i := range s.Resources.GRPCRoutes.Count {
			grpcRoute := baseGRPCRoute.DeepCopy()
			gwName := ""
			grpcRoute.Name = fmt.Sprintf("%s-%d", s.Resources.GRPCRoutes.NamingScheme, i)
			switch routeDistribution.Type {
			case Uniform:
				gwName = distributionWithLeastAttached(countPerGateway)
			case Random:
				gwName = distributionRandom(gateways)
			case Weighted:
				gwName = distributionWithWeight(s.Resources.GRPCRoutes.Distribution.Targets)
			}
			kind := gatewayv1.Kind(baseGateway.Kind)
			ns := gatewayv1.Namespace(baseGateway.Namespace)
			grpcRoute.Spec.ParentRefs = append(grpcRoute.Spec.ParentRefs, gatewayv1.ParentReference{
				Name:      gatewayv1.ObjectName(gwName),
				Kind:      &kind,
				Namespace: &ns,
			})
			grpcRoutes[grpcRoute.Name] = grpcRoute
		}
	}
	return grpcRoutes, nil
}

func (s *Scenario) initServices(ctx context.Context, logger *slog.Logger, httpRoutes map[string]*gatewayv1.HTTPRoute, grpcRoutes map[string]*gatewayv1.GRPCRoute) (map[string]*v1.Service, error) {
	services := make(map[string]*v1.Service)
	serviceDistribution := s.Resources.Services.Distribution
	countPerRoute := make(map[string]int)
	routeCollection := make(map[string]struct{})
	for k, _ := range httpRoutes {
		routeCollection[k] = struct{}{}
	}
	for k, _ := range grpcRoutes {
		routeCollection[k] = struct{}{}
	}
	for k, _ := range routeCollection {
		countPerRoute[k] = 0
	}
	if s.Resources.Services.Count > 0 {
		logger.Log(ctx, slog.LevelInfo, "creating services")
		if err := s.Resources.Services.Validate(); err != nil {
			return nil, fmt.Errorf("invalid service resource definition: %w", err)
		}
		for i := range s.Resources.Services.Count {
			service := baseService.DeepCopy()
			service.Name = fmt.Sprintf("%s-%d", s.Resources.Services.NamingScheme, i)
			routeName := ""
			switch serviceDistribution.Type {
			case Uniform:
				routeName = distributionWithLeastAttached(countPerRoute)
			case Random:
				routeName = distributionRandom(routeCollection)
			case Weighted:
				routeName = distributionWithWeight(s.Resources.Services.Distribution.Targets)
			}
			w := int32(rand.Int())
			var (
				group     = gatewayv1.Group(service.GroupVersionKind().Group)
				kind      = gatewayv1.Kind(service.Kind)
				name      = v1beta1.ObjectName(service.Name)
				namespace = gatewayv1.Namespace(service.Namespace)
			)
			if v, ok := httpRoutes[routeName]; ok {
				route := v.DeepCopy()
				route.Spec.Rules[0].BackendRefs = append(v.Spec.Rules[0].BackendRefs, gatewayv1.HTTPBackendRef{
					BackendRef: gatewayv1.BackendRef{
						Weight: &w,
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Name:      name,
							Group:     &group,
							Kind:      &kind,
							Namespace: &namespace,
						},
					},
				})
				httpRoutes[routeName] = route
			} else if v, ok := grpcRoutes[routeName]; ok {
				route := v.DeepCopy()
				route.Spec.Rules[0].BackendRefs = append(v.Spec.Rules[0].BackendRefs, gatewayv1.GRPCBackendRef{
					BackendRef: gatewayv1.BackendRef{
						Weight: &w,
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Name:      name,
							Group:     &group,
							Kind:      &kind,
							Namespace: &namespace,
						},
					},
				})
				grpcRoutes[routeName] = route
			}
			services[service.Name] = service
		}
	}
	return services, nil
}

// Init sets up the base resources associated with the scenario, it does not start the delta application process
// ideally it should be left up to Apply to be called
func (s *Scenario) Init(ctx context.Context, logger *slog.Logger, capi api.CompositeApi) error {
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
	gwclasses, err := s.initGatewayClasses(ctx, logger)
	if err != nil {
		return fmt.Errorf("invalid gateway class resource definition: %w", err)
	}
	gateways, err := s.initGateways(ctx, logger, gwclasses)
	if err != nil {
		return fmt.Errorf("invalid gateway resource definition: %w", err)
	}
	httpRoutes, err := s.initHTTPRoutes(ctx, logger, gateways)
	if err != nil {
		return fmt.Errorf("invalid http route resource definition: %w", err)
	}
	grpcRoutes, err := s.initGRPCRoutes(ctx, logger, gateways)
	if err != nil {
		return fmt.Errorf("invalid grpc route resource definition: %w", err)
	}
	services, err := s.initServices(ctx, logger, httpRoutes, grpcRoutes)
	if err != nil {
		return fmt.Errorf("invalid service resource definition: %w", err)
	}
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
	for _, gwclass := range gwclasses {
		if err := capi.GatewayClass().Create(ctx, gwclass); err != nil {
			return fmt.Errorf("failed to create gateway class %s: %w", gwclass, err)
		}
	}
	for _, gw := range gateways {
		if err := capi.Gateway().Create(ctx, gw); err != nil {
			return fmt.Errorf("failed to create gateway: %w", err)
		}
	}
	for _, http := range httpRoutes {
		if err := capi.HTTPRoute().Create(ctx, http); err != nil {
			return fmt.Errorf("failed to create httproute: %w", err)
		}
	}
	for _, grpc := range grpcRoutes {
		if err := capi.GRPCRoute().Create(ctx, grpc); err != nil {
			return fmt.Errorf("failed to create grpc route: %w", err)
		}
	}
	for _, service := range services {
		if err := capi.Service().Create(ctx, service); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}
	return nil
}
func distributionWithLeastAttached(parentToChildCount map[string]int) string {
	minValue := math.MaxInt32
	minName := ""
	for k, v := range parentToChildCount {
		if v < minValue {
			minName, minValue = k, v
		}
	}
	parentToChildCount[minName] += parentToChildCount[minName] + 1
	return minName
}
func distributionRandom[T any](currentDistribtuion map[string]T) string {
	asSclie := slices.Sorted(maps.Keys(currentDistribtuion))
	randomIndex := rand.Intn(len(asSclie))
	return asSclie[randomIndex]

}

func distributionWithWeight(weights []Target) string {
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
