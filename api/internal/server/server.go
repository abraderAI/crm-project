// Package server provides the HTTP server setup and router configuration.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/board"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/health"
	"github.com/abraderAI/crm-project/api/internal/membership"
	"github.com/abraderAI/crm-project/api/internal/message"
	"github.com/abraderAI/crm-project/api/internal/middleware"
	"github.com/abraderAI/crm-project/api/internal/org"
	"github.com/abraderAI/crm-project/api/internal/space"
	"github.com/abraderAI/crm-project/api/internal/telemetry"
	"github.com/abraderAI/crm-project/api/internal/thread"
	"github.com/abraderAI/crm-project/api/internal/voice"
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
	r.Use(telemetry.HTTPTrace)
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

	// Initialize domain services.
	orgRepo := org.NewRepository(cfg.DB)
	orgService := org.NewService(orgRepo)
	orgHandler := org.NewHandler(orgService)

	spaceRepo := space.NewRepository(cfg.DB)
	spaceService := space.NewService(spaceRepo)
	spaceHandler := space.NewHandler(spaceService)

	boardRepo := board.NewRepository(cfg.DB)
	boardService := board.NewService(boardRepo)
	boardHandler := board.NewHandler(boardService)

	threadRepo := thread.NewRepository(cfg.DB)
	threadService := thread.NewService(threadRepo)
	threadHandler := thread.NewHandler(threadService)

	msgRepo := message.NewRepository(cfg.DB)
	msgService := message.NewService(msgRepo)
	msgHandler := message.NewHandler(msgService)

	memberRepo := membership.NewRepository(cfg.DB)
	memberHandler := membership.NewHandler(memberRepo)

	// Voice provider (stub).
	voiceProvider := voice.NewStubProvider(cfg.DB)
	voiceHandler := voice.NewHandler(voiceProvider)

	// GDPR service.
	gdprService := gdpr.NewService(cfg.DB)
	gdprHandler := gdpr.NewHandler(gdprService)

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

			// Org routes.
			authed.Post("/orgs", orgHandler.Create)
			authed.Get("/orgs", orgHandler.List)
			authed.Get("/orgs/{org}", orgHandler.Get)
			authed.Patch("/orgs/{org}", orgHandler.Update)
			authed.Delete("/orgs/{org}", orgHandler.Delete)

			// API key management routes.
			authed.Route("/orgs/{org}/api-keys", func(ak chi.Router) {
				ak.Post("/", apiKeyHandler.Create)
				ak.Get("/", apiKeyHandler.List)
				ak.Delete("/{id}", apiKeyHandler.Revoke)
			})

			// Voice call routes.
			authed.Route("/orgs/{org}/calls", func(call chi.Router) {
				call.Post("/", voiceHandler.LogCall)
				call.Get("/{call}", voiceHandler.GetTranscript)
				call.Post("/{call}/escalate", voiceHandler.Escalate)
			})

			// Admin routes (GDPR).
			authed.Route("/admin", func(admin chi.Router) {
				admin.Get("/users/{user}/export", gdprHandler.ExportUserData)
				admin.Delete("/users/{user}/purge", gdprHandler.PurgeUser)
				admin.Delete("/orgs/{org}/purge", gdprHandler.PurgeOrg)
			})

			// Org membership routes.
			authed.Route("/orgs/{org}/members", func(m chi.Router) {
				m.Post("/", memberHandler.AddOrgMember)
				m.Get("/", memberHandler.ListOrgMembers)
				m.Patch("/{userID}", memberHandler.UpdateOrgMember)
				m.Delete("/{userID}", memberHandler.RemoveOrgMember)
			})

			// Space routes.
			authed.Route("/orgs/{org}/spaces", func(sp chi.Router) {
				sp.Post("/", spaceHandler.Create)
				sp.Get("/", spaceHandler.List)
				sp.Get("/{space}", spaceHandler.Get)
				sp.Patch("/{space}", spaceHandler.Update)
				sp.Delete("/{space}", spaceHandler.Delete)

				// Space membership routes.
				sp.Route("/{space}/members", func(m chi.Router) {
					m.Post("/", memberHandler.AddSpaceMember)
					m.Get("/", memberHandler.ListSpaceMembers)
					m.Delete("/{userID}", memberHandler.RemoveSpaceMember)
				})

				// Board routes.
				sp.Route("/{space}/boards", func(bd chi.Router) {
					bd.Post("/", boardHandler.Create)
					bd.Get("/", boardHandler.List)
					bd.Get("/{board}", boardHandler.Get)
					bd.Patch("/{board}", boardHandler.Update)
					bd.Delete("/{board}", boardHandler.Delete)
					bd.Post("/{board}/lock", boardHandler.Lock)
					bd.Post("/{board}/unlock", boardHandler.Unlock)

					// Board membership routes.
					bd.Route("/{board}/members", func(m chi.Router) {
						m.Post("/", memberHandler.AddBoardMember)
						m.Get("/", memberHandler.ListBoardMembers)
						m.Delete("/{userID}", memberHandler.RemoveBoardMember)
					})

					// Thread routes.
					bd.Route("/{board}/threads", func(th chi.Router) {
						th.Post("/", threadHandler.Create)
						th.Get("/", threadHandler.List)
						th.Get("/{thread}", threadHandler.Get)
						th.Patch("/{thread}", threadHandler.Update)
						th.Delete("/{thread}", threadHandler.Delete)
						th.Post("/{thread}/pin", threadHandler.Pin)
						th.Post("/{thread}/unpin", threadHandler.Unpin)
						th.Post("/{thread}/lock", threadHandler.Lock)
						th.Post("/{thread}/unlock", threadHandler.Unlock)

						// Message routes.
						th.Route("/{thread}/messages", func(msg chi.Router) {
							msg.Post("/", msgHandler.Create)
							msg.Get("/", msgHandler.List)
							msg.Get("/{message}", msgHandler.Get)
							msg.Patch("/{message}", msgHandler.Update)
							msg.Delete("/{message}", msgHandler.Delete)
						})
					})
				})
			})
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

// NewJWTValidatorFromConfig returns a configured JWT validator from the server config.
// Exposed for live tests to set test keys.
func NewJWTValidatorFromConfig(cfg Config) *auth.JWTValidator {
	return auth.NewJWTValidator(cfg.IssuerURL)
}
