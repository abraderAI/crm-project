package middleware

import (
	"net/http"
	"strings"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// ContentType enforces application/json Content-Type on requests with bodies
// (POST, PUT, PATCH). GET, DELETE, OPTIONS, HEAD requests are exempt.
func ContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requiresBody(r.Method) && r.ContentLength > 0 {
			ct := r.Header.Get("Content-Type")
			if !isJSON(ct) {
				apierrors.BadRequest(w, "Content-Type must be application/json")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func requiresBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func isJSON(contentType string) bool {
	ct := strings.TrimSpace(strings.Split(contentType, ";")[0])
	return ct == "application/json"
}
