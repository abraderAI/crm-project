package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "deft-evolution-api/metrics"

var (
	initOnce          sync.Once
	requestCounter    metric.Int64Counter
	latencyHistogram  metric.Float64Histogram
	activeConnections metric.Int64UpDownCounter
	metricsInitErr    error
)

// initMetrics lazily initializes all metric instruments.
func initMetrics() {
	initOnce.Do(func() {
		meter := otel.Meter(meterName)

		requestCounter, metricsInitErr = meter.Int64Counter(
			"http.server.request_count",
			metric.WithDescription("Total number of HTTP requests"),
			metric.WithUnit("{request}"),
		)
		if metricsInitErr != nil {
			return
		}

		latencyHistogram, metricsInitErr = meter.Float64Histogram(
			"http.server.request_duration_ms",
			metric.WithDescription("HTTP request latency in milliseconds"),
			metric.WithUnit("ms"),
		)
		if metricsInitErr != nil {
			return
		}

		activeConnections, metricsInitErr = meter.Int64UpDownCounter(
			"http.server.active_connections",
			metric.WithDescription("Number of active HTTP connections"),
			metric.WithUnit("{connection}"),
		)
	})
}

// recordHTTPMetrics records request count and latency metrics.
func recordHTTPMetrics(method, path string, status int, duration time.Duration) {
	initMetrics()
	if metricsInitErr != nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", path),
		attribute.Int("http.status_code", status),
	}

	ctx := context.Background()
	requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	latencyHistogram.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
}

// IncrementActiveConnections increments the active connections gauge.
func IncrementActiveConnections() {
	initMetrics()
	if metricsInitErr != nil {
		return
	}
	activeConnections.Add(context.Background(), 1)
}

// DecrementActiveConnections decrements the active connections gauge.
func DecrementActiveConnections() {
	initMetrics()
	if metricsInitErr != nil {
		return
	}
	activeConnections.Add(context.Background(), -1)
}

// RecordWebhookResult records webhook delivery success/failure metrics.
func RecordWebhookResult(success bool) {
	initMetrics()
	meter := otel.Meter(meterName)
	counter, err := meter.Int64Counter(
		"webhook.delivery_count",
		metric.WithDescription("Webhook delivery attempts"),
		metric.WithUnit("{delivery}"),
	)
	if err != nil {
		return
	}

	status := "success"
	if !success {
		status = "failure"
	}
	counter.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}

// MetricsStatus returns whether metrics were initialized successfully.
func MetricsStatus() (bool, error) {
	initMetrics()
	if metricsInitErr != nil {
		return false, fmt.Errorf("metrics init failed: %w", metricsInitErr)
	}
	return true, nil
}
