package telemetry

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "deft-evolution-api/http"

// statusWriter wraps http.ResponseWriter to capture the status code.
// It also implements http.Hijacker and http.Flusher so that WebSocket
// upgrades and streaming responses work through the middleware chain.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker, required for WebSocket upgrades.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

// Flush implements http.Flusher, required for streaming responses.
func (w *statusWriter) Flush() {
	if fl, ok := w.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
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
