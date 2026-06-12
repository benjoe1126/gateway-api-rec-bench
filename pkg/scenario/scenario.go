package scenario

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"time"

	"gopkg.in/yaml.v3"
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
