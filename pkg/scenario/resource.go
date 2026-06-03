package scenario

import (
	"errors"
	"fmt"
	"regexp"
)

type Resource interface {
	Validate() error
}

type GenericResource struct {
	Count        uint64 `yaml:"count"`
	NamingScheme string `yaml:"namingScheme"`
}

type GenericResourceWithDistribution struct {
	GenericResource
	Distribution DistributionType `yaml:"distribution"`
}

type Resources struct {
	GatewayClasses GatewayClass `yaml:"gatewayClasses"`
	Gateways       Gateway      `yaml:"gateways"`
	HTTPRoutes     HTTPRoute    `yaml:"httpRoutes"`
	GRPCRoutes     GRPCRoute    `yaml:"grpcRoutes"`
	Services       Service      `yaml:"services"`
}

type GatewayClass struct {
	GenericResource
}
type Gateway struct {
	GenericResourceWithDistribution
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
	GenericResourceWithDistribution
}

func (h HTTPRoute) Validate() error {
	return genericValidationWithDistribution(h.NamingScheme, h.Distribution, h.Count)
}

type GRPCRoute struct {
	GenericResourceWithDistribution
}

func (g GRPCRoute) Validate() error {
	return genericValidationWithDistribution(g.NamingScheme, g.Distribution, g.Count)
}

type Service struct {
	GenericResourceWithDistribution
}

func (s Service) Validate() error {
	return genericValidationWithDistribution(s.NamingScheme, s.Distribution, s.Count)
}
