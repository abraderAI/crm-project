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
	"github.com/abraderAI/crm-project/api/internal/health"
	"github.com/abraderAI/crm-project/api/internal/membership"
	"github.com/abraderAI/crm-project/api/internal/message"
	"github.com/abraderAI/crm-project/api/internal/middleware"
	"github.com/abraderAI/crm-project/api/internal/org"
	"github.com/abraderAI/crm-project/api/internal/space"
	"github.com/abraderAI/crm-project/api/internal/thread"
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

	// Initialize domain components.
	resolver := NewResolver(cfg.DB)
	boardLockChecker := NewBoardLockChecker(cfg.DB)
	threadLockChecker := NewThreadLockChecker(cfg.DB)
	memberRepoAdapter := NewMemberRepoAdapter(cfg.DB)

	orgRepo := org.NewRepository(cfg.DB)
	orgSvc := org.NewService(orgRepo)
	orgHandler := org.NewHandler(orgSvc, memberRepoAdapter)

	spaceRepo := space.NewRepository(cfg.DB)
	spaceSvc := space.NewService(spaceRepo)
	spaceHandler := space.NewHandler(spaceSvc, resolver)

	boardRepo := board.NewRepository(cfg.DB)
	boardSvc := board.NewService(boardRepo)
	boardHandler := board.NewHandler(boardSvc, resolver)

	threadRepo := thread.NewRepository(cfg.DB)
	threadSvc := thread.NewService(threadRepo, boardLockChecker)
	threadHandler := thread.NewHandler(threadSvc, resolver)

	msgRepo := message.NewRepository(cfg.DB)
	msgSvc := message.NewService(msgRepo, threadLockChecker)
	msgHandler := message.NewHandler(msgSvc, resolver)

	memberRepo := membership.NewRepository(cfg.DB)
	memberSvc := membership.NewService(memberRepo)
	memberHandler := membership.NewHandler(memberSvc)

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

			// Org CRUD.
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

			// Org membership.
			authed.Post("/orgs/{org}/members", memberHandler.AddOrgMember)
			authed.Get("/orgs/{org}/members", memberHandler.ListOrgMembers)
			authed.Patch("/orgs/{org}/members/{id}", memberHandler.UpdateOrgMember)
			authed.Delete("/orgs/{org}/members/{id}", memberHandler.RemoveOrgMember)

			// Space CRUD.
			authed.Post("/orgs/{org}/spaces", spaceHandler.Create)
			authed.Get("/orgs/{org}/spaces", spaceHandler.List)
			authed.Get("/orgs/{org}/spaces/{space}", spaceHandler.Get)
			authed.Patch("/orgs/{org}/spaces/{space}", spaceHandler.Update)
			authed.Delete("/orgs/{org}/spaces/{space}", spaceHandler.Delete)

			// Space membership.
			authed.Post("/orgs/{org}/spaces/{space}/members", memberHandler.AddSpaceMember)
			authed.Get("/orgs/{org}/spaces/{space}/members", memberHandler.ListSpaceMembers)
			authed.Patch("/orgs/{org}/spaces/{space}/members/{id}", memberHandler.UpdateSpaceMember)
			authed.Delete("/orgs/{org}/spaces/{space}/members/{id}", memberHandler.RemoveSpaceMember)

			// Board CRUD.
			authed.Post("/orgs/{org}/spaces/{space}/boards", boardHandler.Create)
			authed.Get("/orgs/{org}/spaces/{space}/boards", boardHandler.List)
			authed.Get("/orgs/{org}/spaces/{space}/boards/{board}", boardHandler.Get)
			authed.Patch("/orgs/{org}/spaces/{space}/boards/{board}", boardHandler.Update)
			authed.Delete("/orgs/{org}/spaces/{space}/boards/{board}", boardHandler.Delete)
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/lock", boardHandler.Lock)
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/unlock", boardHandler.Unlock)

			// Board membership.
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/members", memberHandler.AddBoardMember)
			authed.Get("/orgs/{org}/spaces/{space}/boards/{board}/members", memberHandler.ListBoardMembers)
			authed.Patch("/orgs/{org}/spaces/{space}/boards/{board}/members/{id}", memberHandler.UpdateBoardMember)
			authed.Delete("/orgs/{org}/spaces/{space}/boards/{board}/members/{id}", memberHandler.RemoveBoardMember)

			// Thread CRUD.
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/threads", threadHandler.Create)
			authed.Get("/orgs/{org}/spaces/{space}/boards/{board}/threads", threadHandler.List)
			authed.Get("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}", threadHandler.Get)
			authed.Patch("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}", threadHandler.Update)
			authed.Delete("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}", threadHandler.Delete)
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/pin", threadHandler.Pin)
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/unpin", threadHandler.Unpin)
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/lock", threadHandler.Lock)
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/unlock", threadHandler.Unlock)

			// Message CRUD.
			authed.Post("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages", msgHandler.Create)
			authed.Get("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages", msgHandler.List)
			authed.Get("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}", msgHandler.Get)
			authed.Patch("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}", msgHandler.Update)
			authed.Delete("/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}", msgHandler.Delete)
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
