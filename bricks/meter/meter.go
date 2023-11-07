package meter

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type Config struct {
	ServiceName           string
	ServiceNamespace      string
	DeploymentEnvironment string
	OTELEndpoint          string
	Insecure              bool
	RuntimeMetrics        bool
	HostMetrics           bool
}

func Init(ctx context.Context, cfg Config) (func(ctx context.Context) error, error) {
	otlpOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.OTELEndpoint)}
	if cfg.Insecure {
		otlpOpts = append(otlpOpts, otlpmetrichttp.WithInsecure())
	}
	exp, err := otlpmetrichttp.New(ctx, otlpOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed init metrics exporter: %w", err)
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceNamespace(cfg.ServiceNamespace),
		semconv.DeploymentEnvironment(cfg.DeploymentEnvironment),
	)
	meterProvider := metric.NewMeterProvider(metric.WithResource(res), metric.WithReader(metric.NewPeriodicReader(exp)))
	if cfg.HostMetrics {
		if err := host.Start(host.WithMeterProvider(meterProvider)); err != nil {
			return nil, fmt.Errorf("failed start host metrics: %w", err)
		}
	}
	if cfg.RuntimeMetrics {
		if err := runtime.Start(); err != nil {
			return nil, fmt.Errorf("failed start runtime metrics: %w", err)
		}
	}
	otel.SetMeterProvider(meterProvider)
	return meterProvider.Shutdown, nil
}
