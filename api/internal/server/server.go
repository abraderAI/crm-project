// Package server provides the HTTP server setup and router configuration.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/admin"
	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/billing"
	"github.com/abraderAI/crm-project/api/internal/board"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/health"
	"github.com/abraderAI/crm-project/api/internal/llm"
	"github.com/abraderAI/crm-project/api/internal/membership"
	"github.com/abraderAI/crm-project/api/internal/message"
	"github.com/abraderAI/crm-project/api/internal/middleware"
	"github.com/abraderAI/crm-project/api/internal/moderation"
	"github.com/abraderAI/crm-project/api/internal/notification"
	"github.com/abraderAI/crm-project/api/internal/org"
	"github.com/abraderAI/crm-project/api/internal/pipeline"
	"github.com/abraderAI/crm-project/api/internal/provision"
	"github.com/abraderAI/crm-project/api/internal/revision"
	"github.com/abraderAI/crm-project/api/internal/scoring"
	"github.com/abraderAI/crm-project/api/internal/search"
	"github.com/abraderAI/crm-project/api/internal/space"
	"github.com/abraderAI/crm-project/api/internal/telemetry"
	"github.com/abraderAI/crm-project/api/internal/thread"
	"github.com/abraderAI/crm-project/api/internal/upload"
	"github.com/abraderAI/crm-project/api/internal/voice"
	"github.com/abraderAI/crm-project/api/internal/vote"
	"github.com/abraderAI/crm-project/api/internal/webhook"
	ws "github.com/abraderAI/crm-project/api/internal/websocket"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// Config holds server dependencies.
type Config struct {
	DB                  *gorm.DB
	Logger              *slog.Logger
	CORSOrigins         []string
	RBACPolicy          *config.RBACPolicy
	IssuerURL           string // Clerk issuer URL for JWT validation.
	WebhookSecret       string // HMAC secret for billing webhook verification.
	EventBus            *eventbus.Bus
	WSHub               *ws.Hub
	UploadDir           string // Directory for file uploads.
	MaxUpload           int64  // Maximum upload size in bytes.
	PlatformAdminUserID string // Bootstrap platform admin user ID.
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

	// Initialize WebSocket hub (use provided or create new).
	wsHub := cfg.WSHub
	if wsHub == nil {
		wsHub = ws.NewHub(cfg.Logger)
	}
	wsHandler := ws.NewHandler(wsHub, jwtValidator, cfg.Logger, cfg.CORSOrigins)

	// Initialize notification system.
	notifRepo := notification.NewRepository(cfg.DB)
	notifHandler := notification.NewHandler(notifRepo)
	// Initialize event bus.
	eventBus := event.NewBus()

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

	// Initialize billing service.
	billingProvider := billing.NewFlexPointProvider(cfg.WebhookSecret)
	billingService := billing.NewService(billingProvider, cfg.DB)
	billingHandler := billing.NewHandler(billingService, cfg.WebhookSecret)

	voteRepo := vote.NewRepository(cfg.DB)
	voteService := vote.NewService(voteRepo, nil)
	voteHandler := vote.NewHandler(voteService)

	modRepo := moderation.NewRepository(cfg.DB)
	modService := moderation.NewService(modRepo)
	modHandler := moderation.NewHandler(modService)

	// Voice provider (stub).
	voiceProvider := voice.NewStubProvider(cfg.DB)
	voiceHandler := voice.NewHandler(voiceProvider)

	// GDPR service.
	gdprService := gdpr.NewService(cfg.DB)
	gdprHandler := gdpr.NewHandler(gdprService)

	// Phase 5: Advanced API features.
	searchRepo := search.NewRepository(cfg.DB)
	searchHandler := search.NewHandler(searchRepo)

	// Upload storage provider.
	uploadDir := cfg.UploadDir
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	maxUpload := cfg.MaxUpload
	if maxUpload <= 0 {
		maxUpload = 104857600 // 100MB default
	}
	var uploadHandler *upload.Handler
	if storage, err := upload.NewLocalStorage(uploadDir); err == nil {
		uploadService := upload.NewService(cfg.DB, storage, maxUpload)
		uploadHandler = upload.NewHandler(uploadService, maxUpload)
	}

	webhookService := webhook.NewService(cfg.DB)
	webhookHandler := webhook.NewHandler(webhookService)
	// Subscribe webhook service to all events.
	eventBus.SubscribeAll(webhookService.HandleEvent)

	auditService := audit.NewService(cfg.DB)
	auditHandler := audit.NewHandler(auditService)

	// Admin service and handler.
	adminService := admin.NewService(cfg.DB)
	adminHandler := admin.NewHandler(adminService, auditService, gdprService)

	revisionRepo := revision.NewRepository(cfg.DB)
	revisionHandler := revision.NewHandler(revisionRepo)

	// Phase 7: CRM application layer.
	pipelineService := pipeline.NewService(cfg.DB, eventBus)
	pipelineHandler := pipeline.NewHandler(pipelineService)

	scoringService := scoring.NewService(cfg.DB, eventBus)
	// Subscribe scoring to pipeline stage changes.
	eventBus.Subscribe(event.PipelineStageChanged, scoringService.HandleStageChanged)

	llmProvider := llm.NewGrokProvider()
	llmHandler := llm.NewHandler(llmProvider, cfg.DB, eventBus)

	provisionService := provision.NewService(cfg.DB, billingProvider, eventBus)
	provisionHandler := provision.NewHandler(provisionService)
	// Subscribe provisioning to pipeline stage changes (auto-provision on closed_won).
	eventBus.Subscribe(event.PipelineStageChanged, provisionService.HandleStageChanged)

	// API v1 subrouter.
	r.Route("/v1", func(v1 chi.Router) {
		// Public v1 root (no auth).
		v1.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"v1","status":"ok"}`))
		})

		// Public billing webhook endpoint (HMAC-verified, no JWT).
		v1.Post("/webhooks/billing", billingHandler.HandleWebhook)

		// WebSocket endpoint (auth via query param).
		v1.Get("/ws", wsHandler.Upgrade)

		// Authenticated routes.
		v1.Group(func(authed chi.Router) {
			authed.Use(auth.DualAuth(jwtValidator, apiKeyService))
			authed.Use(admin.BanCheck(adminService))
			authed.Use(admin.OrgSuspensionCheck(adminService))
			authed.Use(admin.UserShadowSync(adminService))

			// Search endpoint.
			authed.Get("/search", searchHandler.Search)

			// Upload endpoints.
			if uploadHandler != nil {
				authed.Post("/uploads", uploadHandler.Create)
				authed.Get("/uploads/{id}", uploadHandler.Get)
				authed.Get("/uploads/{id}/download", uploadHandler.Download)
				authed.Delete("/uploads/{id}", uploadHandler.Delete)
			}

			// Revision history endpoints.
			authed.Get("/revisions/{entityType}/{entityID}", revisionHandler.List)
			authed.Get("/revisions/{id}", revisionHandler.Get)

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

			// Admin routes — require platform admin.
			authed.Route("/admin", func(ar chi.Router) {
				ar.Use(admin.PlatformAdminOnly(adminService))

				// Legacy GDPR export.
				ar.Get("/users/{user}/export", gdprHandler.ExportUserData)

				// User management.
				ar.Get("/users", adminHandler.ListUsers)
				ar.Get("/users/{user_id}", adminHandler.GetUser)
				ar.Post("/users/{user_id}/ban", adminHandler.BanUser)
				ar.Post("/users/{user_id}/unban", adminHandler.UnbanUser)
				ar.Delete("/users/{user_id}/purge", adminHandler.PurgeUser)

				// Org management.
				ar.Get("/orgs", adminHandler.ListOrgs)
				ar.Get("/orgs/{org}", adminHandler.GetOrg)
				ar.Post("/orgs/{org}/suspend", adminHandler.SuspendOrg)
				ar.Post("/orgs/{org}/unsuspend", adminHandler.UnsuspendOrg)
				ar.Post("/orgs/{org}/transfer-ownership", adminHandler.TransferOwnership)
				ar.Delete("/orgs/{org}/purge", adminHandler.PurgeOrg)

				// Platform-wide audit log.
				ar.Get("/audit-log", adminHandler.ListAuditLog)

				// Platform admin management.
				ar.Get("/platform-admins", adminHandler.ListPlatformAdmins)
				ar.Post("/platform-admins", adminHandler.AddPlatformAdmin)
				ar.Delete("/platform-admins/{user_id}", adminHandler.RemovePlatformAdmin)
			})
			// Webhook routes.
			authed.Route("/orgs/{org}/webhooks", func(wh chi.Router) {
				wh.Post("/", webhookHandler.Create)
				wh.Get("/", webhookHandler.List)
				wh.Get("/{id}", webhookHandler.Get)
				wh.Delete("/{id}", webhookHandler.Delete)
				wh.Get("/{id}/deliveries", webhookHandler.ListDeliveries)
				wh.Post("/{id}/deliveries/{deliveryID}/replay", webhookHandler.Replay)
			})

			// Audit log route.
			authed.Get("/orgs/{org}/audit-log", auditHandler.List)

			// Org membership routes.
			authed.Route("/orgs/{org}/members", func(m chi.Router) {
				m.Post("/", memberHandler.AddOrgMember)
				m.Get("/", memberHandler.ListOrgMembers)
				m.Patch("/{userID}", memberHandler.UpdateOrgMember)
				m.Delete("/{userID}", memberHandler.RemoveOrgMember)
			})

			// Billing routes.
			authed.Route("/orgs/{org}/billing", func(bl chi.Router) {
				bl.Get("/", billingHandler.GetBillingStatus)
				bl.Post("/customers", billingHandler.CreateCustomer)
				bl.Post("/invoices", billingHandler.CreateInvoice)
			})

			// Vote weight table.
			authed.Get("/vote/weights", voteHandler.GetWeightTable)

			// Moderation flag routes.
			authed.Route("/orgs/{org}/flags", func(fl chi.Router) {
				fl.Post("/", modHandler.CreateFlag)
				fl.Get("/", modHandler.ListFlags)
				fl.Post("/{flag}/resolve", modHandler.ResolveFlag)
				fl.Post("/{flag}/dismiss", modHandler.DismissFlag)
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
						th.Post("/{thread}/vote", voteHandler.Toggle)
						th.Post("/{thread}/move", modHandler.MoveThread)
						th.Post("/{thread}/merge", modHandler.MergeThread)
						th.Post("/{thread}/hide", modHandler.HideThread)
						th.Post("/{thread}/unhide", modHandler.UnhideThread)

						// CRM pipeline routes.
						th.Post("/{thread}/stage", pipelineHandler.TransitionStage)
						th.Post("/{thread}/enrich", llmHandler.Enrich)
						th.Post("/{thread}/provision", provisionHandler.Provision)

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

			// Pipeline stages route.
			authed.Get("/orgs/{org}/pipeline/stages", pipelineHandler.GetStages)

			// Notification routes.
			authed.Route("/notifications", func(n chi.Router) {
				n.Get("/", notifHandler.List)
				n.Patch("/{id}/read", notifHandler.MarkRead)
				n.Post("/mark-all-read", notifHandler.MarkAllRead)
				n.Get("/preferences", notifHandler.GetPreferences)
				n.Put("/preferences", notifHandler.UpdatePreferences)
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
