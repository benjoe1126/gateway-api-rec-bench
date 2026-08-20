package provider

import (
	"context"
	"onlab-bm/pkg/metrics"
)

type Provider interface {
	LoadConfig(path string) error
	GenerateDefaultConfig() (string, error)
	Setup() error
	Start(ctx context.Context) (chan metrics.Record, chan error)
}
