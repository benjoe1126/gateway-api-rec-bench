package provider

import "testing"

func TestConfigGeneration(t *testing.T) {
	pr := NewEnvoyGatewayProvider()
	t.Log(pr.GenerateDefaultConfig())
}
