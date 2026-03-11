// Package telemetry provides OpenTelemetry instrumentation for HTTP requests,
// database queries, and custom application metrics.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	serviceName    = "deft-evolution-api"
	serviceVersion = "1.0.0"
)

// Config holds OpenTelemetry configuration.
type Config struct {
	Enabled     bool
	Endpoint    string // OTLP endpoint; empty uses stdout exporter.
	ServiceName string
}

// Provider holds initialized OTel providers for cleanup.
type Provider struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
}

// Shutdown gracefully shuts down all OTel providers.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.TracerProvider != nil {
		if err := p.TracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down tracer provider: %w", err)
		}
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutting down meter provider: %w", err)
		}
	}
	return nil
}

// Init initializes OpenTelemetry with the given configuration.
// Returns a Provider that must be shut down on application exit.
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	svcName := cfg.ServiceName
	if svcName == "" {
		svcName = serviceName
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(svcName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	provider := &Provider{}

	// Initialize tracer provider.
	tp, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("initializing tracer: %w", err)
	}
	provider.TracerProvider = tp
	otel.SetTracerProvider(tp)

	// Initialize meter provider.
	mp, err := initMeterProvider(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("initializing meter: %w", err)
	}
	provider.MeterProvider = mp
	otel.SetMeterProvider(mp)

	return provider, nil
}

// initTracerProvider creates a TracerProvider with the appropriate exporter.
func initTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if cfg.Endpoint != "" {
		exporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(cfg.Endpoint),
		)
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	return tp, nil
}

// initMeterProvider creates a MeterProvider with a stdout exporter.
func initMeterProvider(_ context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter, err := stdoutmetric.New()
	if err != nil {
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)

	return mp, nil
}

// Meter returns the global meter for recording metrics.
func Meter() metric.Meter {
	return otel.Meter(serviceName)
}
