package scenario

type DistributionType string

type Target struct {
	ParentName string `yaml:"parentName"`
	Weight     uint8  `yaml:"weight"`
}

const (
	Weighted DistributionType = "weighted"
	Uniform  DistributionType = "uniform"
	Random   DistributionType = "random"
)

type Distribution struct {
	Type    DistributionType `yaml:"type"`
	Seed    uint64           `yaml:"seed;omitempty"`
	Targets []Target         `yaml:"targets;omitempty"`
}
