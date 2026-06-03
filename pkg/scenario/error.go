package scenario

import "errors"

var (
	InvalidNameError         = errors.New("name must match regexp: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$")
	InvalidDistributionError = errors.New("distribution must be one of [Uniform, Weighted, Random]")
	InvalidCountError        = errors.New("count must be 0 or greater")
)
