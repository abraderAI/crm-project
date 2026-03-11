package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// Recovery recovers from panics and returns a 500 RFC 7807 response.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					reqID := GetRequestID(r.Context())
					logger.Error("panic recovered",
						slog.Any("error", err),
						slog.String("request_id", reqID),
						slog.String("stack", string(debug.Stack())),
					)
					apierrors.InternalError(w, "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
