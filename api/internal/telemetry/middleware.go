package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "deft-evolution-api/http"

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPTrace returns middleware that creates a span for each HTTP request.
func HTTPTrace(next http.Handler) http.Handler {
	tracer := otel.Tracer(tracerName)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

		ctx, span := tracer.Start(r.Context(), spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.target", r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			),
		)
		defer span.End()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(sw, r.WithContext(ctx))

		duration := time.Since(start)
		span.SetAttributes(
			attribute.Int("http.status_code", sw.status),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		if sw.status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", sw.status))
		}

		// Record metrics.
		recordHTTPMetrics(r.Method, r.URL.Path, sw.status, duration)
	})
}
