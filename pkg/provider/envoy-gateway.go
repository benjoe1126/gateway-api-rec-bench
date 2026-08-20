package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"onlab-bm/pkg/metrics"
	"os"

	egv1alpha1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/itchyny/json2yaml"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type EnvoyGatewayProvider struct {
	config *egv1alpha1.EnvoyGateway
}

func (e EnvoyGatewayProvider) LoadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config file error: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read config file error: %w", err)
	}
	//translate yaml to json
	asJson, err := yaml.ToJSON(data)
	if err != nil {
		return fmt.Errorf("convert to json error: %w", err)
	}
	return json.Unmarshal(asJson, e.config)
}

func (e EnvoyGatewayProvider) GenerateDefaultConfig() (string, error) {
	cfg := egv1alpha1.EnvoyGateway{
		TypeMeta: v1.TypeMeta{
			Kind:       egv1alpha1.KindEnvoyGateway,
			APIVersion: "gateway.envoyproxy.io/v1alpha1",
		},
		EnvoyGatewaySpec: egv1alpha1.EnvoyGatewaySpec{
			Gateway: &egv1alpha1.Gateway{
				ControllerName: "gateway.envoyproxy.io/gatewayclass-controller",
			},
			Admin: &egv1alpha1.EnvoyGatewayAdmin{
				EnablePprof: true,
			},
			Provider: &egv1alpha1.EnvoyGatewayProvider{
				Type:       egv1alpha1.ProviderTypeKubernetes,
				Kubernetes: &egv1alpha1.EnvoyGatewayKubernetesProvider{},
			},
			ExtensionAPIs: &egv1alpha1.ExtensionAPISettings{
				EnableBackend: true,
			},
			Logging: &egv1alpha1.EnvoyGatewayLogging{
				Level: map[egv1alpha1.EnvoyGatewayLogComponent]egv1alpha1.LogLevel{
					"default": egv1alpha1.LogLevelDebug,
				},
			},
			Telemetry: &egv1alpha1.EnvoyGatewayTelemetry{
				Metrics: &egv1alpha1.EnvoyGatewayMetrics{
					Prometheus: &egv1alpha1.EnvoyGatewayPrometheusProvider{
						Disable: false,
					},
					Sinks: []egv1alpha1.EnvoyGatewayMetricSink{
						{
							Type: egv1alpha1.MetricSinkTypeOpenTelemetry,
							OpenTelemetry: &egv1alpha1.EnvoyGatewayOpenTelemetrySink{
								Host:     "localhost",
								Port:     19002,
								Protocol: egv1alpha1.GRPCProtocol,
							},
						},
					},
				},
			},
		},
	}
	asJson, err := json.Marshal(&cfg)
	if err != nil {
		return "", fmt.Errorf("convert to yaml error: %w", err)
	}
	buffer := bytes.NewBuffer(asJson)
	output := bytes.NewBuffer(nil)
	if err := json2yaml.Convert(output, buffer); err != nil {
		return "", fmt.Errorf("convert to yaml error: %w", err)
	}
	return output.String(), nil
}

func (e EnvoyGatewayProvider) Setup() error {
	//TODO implement me
	panic("implement me")
}

func (e EnvoyGatewayProvider) Start(ctx context.Context) (chan metrics.Record, chan error) {
	//TODO implement me
	panic("implement me")
}

func NewEnvoyGatewayProvider() *EnvoyGatewayProvider {
	return &EnvoyGatewayProvider{
		config: nil,
	}
}
