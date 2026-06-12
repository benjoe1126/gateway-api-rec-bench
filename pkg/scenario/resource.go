package scenario

import (
	"errors"
	"fmt"
	"onlab-bm/pkg/patch"
	"regexp"
)

type Resource interface {
	Validate() error
}

type GenericResource struct {
	Count        uint64 `yaml:"count" json:"count"`
	NamingScheme string `yaml:"namingScheme,omitempty" json:"namingScheme,omitempty"`
}

func (g GenericResource) String() string {
	return fmt.Sprintf("count: %d, namingScheme: %s", g.Count, g.NamingScheme)
}

type GenericResourceWithDistribution struct {
	GenericResource `yaml:",inline" json:",inline"`
	Distribution    DistributionType `yaml:"distribution" json:"distribution"`
}

func (g GenericResourceWithDistribution) String() string {
	return fmt.Sprintf("count: %d, namingScheme: %s, distribution: %s", g.Count, g.NamingScheme, g.Distribution)
}

type Resources struct {
	GatewayClasses GatewayClass `yaml:"gatewayClasses" json:"gatewayClasses"`
	Gateways       Gateway      `yaml:"gateways" json:"gateways"`
	HTTPRoutes     HTTPRoute    `yaml:"httpRoutes,omitempty" json:"httpRoutes,omitempty"`
	GRPCRoutes     GRPCRoute    `yaml:"grpcRoutes,omitempty" json:"grpcRoutes,omitempty"`
	Services       Service      `yaml:"services,omitempty" json:"services,omitempty"`
}

type GatewayClass struct {
	GenericResource `yaml:",inline"`
}
type Gateway struct {
	GenericResourceWithDistribution `yaml:",inline"`
}

var k8sNameRegexp = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func genericValidation(namingScheme string, count uint64) error {
	var errs error
	if count == 0 {
		return nil
	}
	if !k8sNameRegexp.MatchString(namingScheme) {
		errs = errors.Join(errs, fmt.Errorf("invalid naming scheme: %s %w", namingScheme, InvalidNameError))
	}
	if count < 0 {
		errs = errors.Join(errs, InvalidCountError)
	}
	return errs
}

func genericValidationWithDistribution(namingScheme string, distribution DistributionType, count uint64) error {
	var errs error
	errs = errors.Join(errs, genericValidation(namingScheme, count))
	if distribution != Weighted && distribution != Uniform && distribution != Random {
		errs = errors.Join(errs, fmt.Errorf("invalid distribution: %s %w", distribution, InvalidDistributionError))
	}
	return errs
}

func (g Gateway) Validate() error {
	return genericValidationWithDistribution(g.NamingScheme, g.Distribution, g.Count)
}

type HTTPRoute struct {
	GenericResourceWithDistribution `yaml:",inline"`
}

func (h HTTPRoute) Validate() error {
	return genericValidationWithDistribution(h.NamingScheme, h.Distribution, h.Count)
}

type GRPCRoute struct {
	GenericResourceWithDistribution `yaml:",inline"`
}

func (g GRPCRoute) Validate() error {
	return genericValidationWithDistribution(g.NamingScheme, g.Distribution, g.Count)
}

type Service struct {
	GenericResourceWithDistribution `yaml:",inline"`
}

func (s Service) Validate() error {
	return genericValidationWithDistribution(s.NamingScheme, s.Distribution, s.Count)
}

type ResourceKind string

const (
	ResourceKindGatewayClasses ResourceKind = "GatewayClasses"
	ResourceKindGateways       ResourceKind = "Gateways"
	ResourceKindHTTPRoutes     ResourceKind = "HTTPRoutes"
	ResourceKindGRPCRoutes     ResourceKind = "GRPCRoutes"
	ResourceKindServices       ResourceKind = "Services"
)

type ResourceSelector struct {
	Kind      ResourceKind `yaml:"kind" json:"kind"`
	Name      string       `yaml:"name" json:"name"`
	Namespace string       `yaml:"namespace,omitempty" json:"namespace"`
}

type ResourceDelta struct {
	ResourceSelector ResourceSelector  `yaml:"selector" json:"selector"`
	Patches          []patch.JsonPatch `json:"patches,omitempty" yaml:"patches,omitempty"`
}
