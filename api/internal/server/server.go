// Package server provides the HTTP server setup and router configuration.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/health"
	"github.com/abraderAI/crm-project/api/internal/middleware"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// Config holds server dependencies.
type Config struct {
	DB          *gorm.DB
	Logger      *slog.Logger
	CORSOrigins []string
	RBACPolicy  *config.RBACPolicy
	IssuerURL   string // Clerk issuer URL for JWT validation.
}

// NewRouter creates and configures the Chi router with all middleware and routes.
func NewRouter(cfg Config) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack.
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(cfg.Logger))
	r.Use(middleware.Logging(cfg.Logger))
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.ContentType)

	// Health check endpoints (outside /v1 — no auth required).
	healthHandler := health.NewHandler(cfg.DB)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	// Initialize auth components.
	jwtValidator := auth.NewJWTValidator(cfg.IssuerURL)
	apiKeyService := auth.NewAPIKeyService(cfg.DB)
	apiKeyHandler := auth.NewAPIKeyHandler(apiKeyService)

	// API v1 subrouter.
	r.Route("/v1", func(v1 chi.Router) {
		// Public v1 root (no auth).
		v1.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"v1","status":"ok"}`))
		})

		// Authenticated routes.
		v1.Group(func(authed chi.Router) {
			authed.Use(auth.DualAuth(jwtValidator, apiKeyService))

			// API key management routes.
			authed.Route("/orgs/{org}/api-keys", func(ak chi.Router) {
				ak.Post("/", apiKeyHandler.Create)
				ak.Get("/", apiKeyHandler.List)
				ak.Delete("/{id}", apiKeyHandler.Revoke)
			})

			// Placeholder for future authenticated routes (Phase 4+).
		})
	})

	// Catch-all for unknown routes returns RFC 7807.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		apierrors.NotFound(w, "the requested resource was not found")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		apierrors.WriteProblem(w, apierrors.ProblemDetail{
			Type:   "https://httpstatuses.com/405",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: "the requested method is not allowed for this resource",
		})
	})

	return r
}

// JWTValidator returns a configured JWT validator from the server config.
// Exposed for live tests to set test keys.
func NewJWTValidatorFromConfig(cfg Config) *auth.JWTValidator {
	return auth.NewJWTValidator(cfg.IssuerURL)
}
