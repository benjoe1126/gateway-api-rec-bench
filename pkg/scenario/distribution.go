package scenario

type DistributionType string

type Target struct {
	ParentName string `yaml:"parentName" json:"parentName"`
	Weight     uint8  `yaml:"weight" json:"weight"`
}

const (
	Weighted DistributionType = "weighted"
	Uniform  DistributionType = "uniform"
	Random   DistributionType = "random"
)

type Distribution struct {
	Type    DistributionType `yaml:"type" json:"type" jsonschema:"enum=random,enum=uniform,enum=weighted"`
	Targets []Target         `yaml:"targets,omitempty" json:"targets,omitempty"`
}
