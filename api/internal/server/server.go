// Package server provides the HTTP server setup and router configuration.
package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/admin"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/eventbus"
	"github.com/abraderAI/crm-project/api/internal/middleware"
	"github.com/abraderAI/crm-project/api/internal/reporting"
	"github.com/abraderAI/crm-project/api/internal/telemetry"
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
	ChatJWTSecret       string // HMAC secret for chat session JWT signing.
	EventBus            *eventbus.Bus
	WSHub               *ws.Hub
	UploadDir           string // Directory for file uploads.
	MaxUpload           int64  // Maximum upload size in bytes.
	PlatformAdminUserID string // Bootstrap platform admin user ID.
	LiveKitWebhookToken string // Auth token for LiveKit webhook verification.
	InternalAPIKey      string // API key for internal bridge endpoints.
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

	// Initialise all handlers and wire event-bus subscriptions.
	h := newHandlers(cfg)

	// Health check endpoints (outside /v1 — no auth required).
	r.Get("/healthz", h.healthHandler.Healthz)
	r.Get("/readyz", h.healthHandler.Readyz)

	// API v1 subrouter.
	r.Route("/v1", func(v1 chi.Router) {
		// Public v1 root (no auth).
		v1.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"v1","status":"ok"}`))
		})

		// Public billing webhook endpoint (HMAC-verified, no JWT).
		v1.Post("/webhooks/billing", h.billingHandler.HandleWebhook)

		// Public LiveKit webhook endpoint (token-verified, no JWT).
		v1.Post("/webhooks/livekit", h.voiceLKWebhook.HandleWebhook)

		// Internal bridge API (X-Internal-Key auth, no JWT).
		v1.Get("/internal/contacts/lookup", h.voiceLKBridge.LookupContact)
		v1.Get("/internal/threads/{id}/summary", h.voiceLKBridge.GetThreadSummary)

		// WebSocket endpoint (auth via query param).
		v1.Get("/ws", h.wsHandler.Upgrade)

		// Chat widget endpoints (public — no auth middleware).
		v1.Post("/chat/session", h.chatHandler.CreateSession)
		v1.Post("/chat/message", h.chatHandler.SendMessage)

		// Chat session promotion (authenticated — called on user registration).
		v1.Group(func(promoteRouter chi.Router) {
			promoteRouter.Use(auth.DualAuth(h.jwtValidator, h.apiKeyService))
			promoteRouter.Post("/chat/promote", h.chatHandler.HandleSessionPromotion)
		})

		// Authenticated routes.
		v1.Group(func(authed chi.Router) {
			authed.Use(auth.DualAuth(h.jwtValidator, h.apiKeyService))
			authed.Use(admin.BanCheck(h.adminService))
			authed.Use(admin.OrgSuspensionCheck(h.adminService))
			authed.Use(admin.MaintenanceMode(h.adminService))
			authed.Use(admin.UserShadowSync(h.adminService))
			authed.Use(admin.APIUsageCounter(cfg.DB))
			authed.Use(admin.LoginEventRecorder(cfg.DB))

			// User tier and preferences endpoints.
			authed.Get("/me/tier", h.tierHandler.GetTier)
			authed.Get("/me/home-preferences", h.tierHandler.GetHomePreferences)
			authed.Put("/me/home-preferences", h.tierHandler.PutHomePreferences)

			// Conversion: self-service upgrade (Tier 2→3).
			authed.Post("/me/upgrade", h.conversionHandler.SelfServiceUpgrade)

			// Conversion: sales-assisted lead conversion (DEFT members only, checked in handler).
			authed.Post("/leads/{lead_id}/convert", h.conversionHandler.SalesConvert)

			// Search endpoint.
			authed.Get("/search", h.searchHandler.Search)

			// Upload endpoints.
			if h.uploadHandler != nil {
				authed.Post("/uploads", h.uploadHandler.Create)
				authed.Get("/uploads/{id}", h.uploadHandler.Get)
				authed.Get("/uploads/{id}/download", h.uploadHandler.Download)
				authed.Delete("/uploads/{id}", h.uploadHandler.Delete)
			}

			// Revision history endpoints.
			authed.Get("/revisions/{entityType}/{entityID}", h.revisionHandler.List)
			authed.Get("/revisions/{id}", h.revisionHandler.Get)

			// Org routes.
			authed.Post("/orgs", h.orgHandler.Create)
			authed.Get("/orgs", h.orgHandler.List)
			authed.Get("/orgs/{org}", h.orgHandler.Get)
			authed.Patch("/orgs/{org}", h.orgHandler.Update)
			authed.Delete("/orgs/{org}", h.orgHandler.Delete)

			// API key management routes.
			authed.Route("/orgs/{org}/api-keys", func(ak chi.Router) {
				ak.Post("/", h.apiKeyHandler.Create)
				ak.Get("/", h.apiKeyHandler.List)
				ak.Delete("/{id}", h.apiKeyHandler.Revoke)
			})

			// Voice call routes.
			authed.Route("/orgs/{org}/calls", func(call chi.Router) {
				call.Post("/", h.voiceHandler.LogCall)
				call.Get("/{call}", h.voiceHandler.GetTranscript)
				call.Post("/{call}/escalate", h.voiceHandler.Escalate)
			})

			// Admin routes — require platform admin.
			authed.Route("/admin", func(ar chi.Router) {
				ar.Use(admin.PlatformAdminOnly(h.adminService))

				// Legacy GDPR export.
				ar.Get("/users/{user}/export", h.gdprHandler.ExportUserData)

				// User management.
				ar.Get("/users", h.adminHandler.ListUsers)
				ar.Get("/users/{user_id}", h.adminHandler.GetUser)
				ar.Post("/users/{user_id}/ban", h.adminHandler.BanUser)
				ar.Post("/users/{user_id}/unban", h.adminHandler.UnbanUser)
				ar.Delete("/users/{user_id}/purge", h.adminHandler.PurgeUser)

				// Org management.
				ar.Get("/orgs", h.adminHandler.ListOrgs)
				ar.Get("/orgs/{org}", h.adminHandler.GetOrg)
				ar.Post("/orgs/{org}/suspend", h.adminHandler.SuspendOrg)
				ar.Post("/orgs/{org}/unsuspend", h.adminHandler.UnsuspendOrg)
				ar.Post("/orgs/{org}/transfer-ownership", h.adminHandler.TransferOwnership)
				ar.Delete("/orgs/{org}/purge", h.adminHandler.PurgeOrg)

				// Platform-wide audit log.
				ar.Get("/audit-log", h.adminHandler.ListAuditLog)

				// Platform admin management.
				ar.Get("/platform-admins", h.adminHandler.ListPlatformAdmins)
				ar.Post("/platform-admins", h.adminHandler.AddPlatformAdmin)
				ar.Delete("/platform-admins/{user_id}", h.adminHandler.RemovePlatformAdmin)

				// Phase B: Configuration & Monitoring.
				ar.Get("/settings", h.adminHandler.GetSettings)
				ar.Patch("/settings", h.adminHandler.PatchSettings)

				ar.Get("/rbac-policy", h.adminHandler.GetRBACPolicy)
				ar.Patch("/rbac-policy", h.adminHandler.PatchRBACPolicy)
				ar.Post("/rbac-policy/preview", h.adminHandler.PreviewRBACPolicy)

				ar.Get("/feature-flags", h.adminHandler.ListFeatureFlags)
				ar.Patch("/feature-flags/{key}", h.adminHandler.PatchFeatureFlag)

				ar.Get("/stats", h.adminHandler.GetStats)

				ar.Get("/webhooks/deliveries", h.adminHandler.ListWebhookDeliveries)

				ar.Get("/integrations/status", h.adminHandler.GetIntegrationHealth)

				// Conversion: platform admin user promotion (Tier 2→3).
				ar.Post("/users/{user_id}/promote", h.conversionHandler.AdminPromote)

				// Phase C: Advanced Admin Features.
				ar.Post("/users/{user_id}/impersonate", h.adminHandler.ImpersonateHandler)

				ar.Post("/exports", h.adminHandler.CreateExportHandler)
				ar.Get("/exports", h.adminHandler.ListExportsHandler)
				ar.Get("/exports/{id}", h.adminHandler.GetExportHandler)

				ar.Get("/api-usage", h.adminHandler.GetAPIUsageHandler)
				ar.Get("/llm-usage", h.adminHandler.GetLLMUsageHandler)

				ar.Get("/security/recent-logins", h.adminHandler.GetRecentLoginsHandler)
				ar.Get("/security/failed-auths", h.adminHandler.GetFailedAuthsHandler)

				// Phase 3: Platform admin reporting.
				ar.Get("/reports/support", h.reportHandler.GetAdminSupportMetrics)
				ar.Get("/reports/support/export", h.reportHandler.GetAdminSupportExport)
				ar.Get("/reports/sales", h.reportHandler.GetAdminSalesMetrics)
				ar.Get("/reports/sales/export", h.reportHandler.GetAdminSalesExport)
			})

			// Webhook routes.
			authed.Route("/orgs/{org}/webhooks", func(wh chi.Router) {
				wh.Post("/", h.webhookHandler.Create)
				wh.Get("/", h.webhookHandler.List)
				wh.Get("/{id}", h.webhookHandler.Get)
				wh.Delete("/{id}", h.webhookHandler.Delete)
				wh.Get("/{id}/deliveries", h.webhookHandler.ListDeliveries)
				wh.Post("/{id}/deliveries/{deliveryID}/replay", h.webhookHandler.Replay)
			})

			// Audit log route.
			authed.Get("/orgs/{org}/audit-log", h.auditHandler.List)

			// Org membership routes.
			authed.Route("/orgs/{org}/members", func(m chi.Router) {
				m.Post("/", h.memberHandler.AddOrgMember)
				m.Get("/", h.memberHandler.ListOrgMembers)
				m.Patch("/{userID}", h.memberHandler.UpdateOrgMember)
				m.Delete("/{userID}", h.memberHandler.RemoveOrgMember)
			})

			// Billing routes.
			authed.Route("/orgs/{org}/billing", func(bl chi.Router) {
				bl.Get("/", h.billingHandler.GetBillingStatus)
				bl.Post("/customers", h.billingHandler.CreateCustomer)
				bl.Post("/invoices", h.billingHandler.CreateInvoice)
			})

			// Vote weight table.
			authed.Get("/vote/weights", h.voteHandler.GetWeightTable)

			// Moderation flag routes.
			authed.Route("/orgs/{org}/flags", func(fl chi.Router) {
				fl.Post("/", h.modHandler.CreateFlag)
				fl.Get("/", h.modHandler.ListFlags)
				fl.Post("/{flag}/resolve", h.modHandler.ResolveFlag)
				fl.Post("/{flag}/dismiss", h.modHandler.DismissFlag)
			})

			// Space routes.
			authed.Route("/orgs/{org}/spaces", func(sp chi.Router) {
				sp.Post("/", h.spaceHandler.Create)
				sp.Get("/", h.spaceHandler.List)
				sp.Get("/{space}", h.spaceHandler.Get)
				sp.Patch("/{space}", h.spaceHandler.Update)
				sp.Delete("/{space}", h.spaceHandler.Delete)

				// Space membership routes.
				sp.Route("/{space}/members", func(m chi.Router) {
					m.Post("/", h.memberHandler.AddSpaceMember)
					m.Get("/", h.memberHandler.ListSpaceMembers)
					m.Delete("/{userID}", h.memberHandler.RemoveSpaceMember)
				})

				// Board routes.
				sp.Route("/{space}/boards", func(bd chi.Router) {
					bd.Post("/", h.boardHandler.Create)
					bd.Get("/", h.boardHandler.List)
					bd.Get("/{board}", h.boardHandler.Get)
					bd.Patch("/{board}", h.boardHandler.Update)
					bd.Delete("/{board}", h.boardHandler.Delete)
					bd.Post("/{board}/lock", h.boardHandler.Lock)
					bd.Post("/{board}/unlock", h.boardHandler.Unlock)

					// Board membership routes.
					bd.Route("/{board}/members", func(m chi.Router) {
						m.Post("/", h.memberHandler.AddBoardMember)
						m.Get("/", h.memberHandler.ListBoardMembers)
						m.Delete("/{userID}", h.memberHandler.RemoveBoardMember)
					})

					// Thread routes.
					bd.Route("/{board}/threads", func(th chi.Router) {
						th.Post("/", h.threadHandler.Create)
						th.Get("/", h.threadHandler.List)
						th.Get("/{thread}", h.threadHandler.Get)
						th.Patch("/{thread}", h.threadHandler.Update)
						th.Delete("/{thread}", h.threadHandler.Delete)
						th.Post("/{thread}/pin", h.threadHandler.Pin)
						th.Post("/{thread}/unpin", h.threadHandler.Unpin)
						th.Post("/{thread}/lock", h.threadHandler.Lock)
						th.Post("/{thread}/unlock", h.threadHandler.Unlock)
						th.Post("/{thread}/vote", h.voteHandler.Toggle)
						th.Post("/{thread}/move", h.modHandler.MoveThread)
						th.Post("/{thread}/merge", h.modHandler.MergeThread)
						th.Post("/{thread}/hide", h.modHandler.HideThread)
						th.Post("/{thread}/unhide", h.modHandler.UnhideThread)

						// CRM pipeline routes.
						th.Post("/{thread}/stage", h.pipelineHandler.TransitionStage)
						th.Post("/{thread}/enrich", h.llmHandler.Enrich)
						th.Post("/{thread}/provision", h.provisionHandler.Provision)

						// Message routes.
						th.Route("/{thread}/messages", func(sg chi.Router) {
							sg.Post("/", h.msgHandler.Create)
							sg.Get("/", h.msgHandler.List)
							sg.Get("/{message}", h.msgHandler.Get)
							sg.Patch("/{message}", h.msgHandler.Update)
							sg.Delete("/{message}", h.msgHandler.Delete)
						})
					})
				})
			})

			// Channel gateway routes.
			authed.Route("/orgs/{org}/channels", func(ch chi.Router) {
				ch.Get("/health", h.channelHandler.GetHealth)
				ch.Get("/dlq", h.channelHandler.ListDLQ)
				ch.Post("/dlq/{id}/retry", h.channelHandler.RetryDLQ)
				ch.Post("/dlq/{id}/dismiss", h.channelHandler.DismissDLQ)
				ch.Get("/{type}", h.channelHandler.GetConfig)
				ch.Put("/{type}", h.channelHandler.PutConfig)

				// Voice phone number admin routes.
				ch.Get("/voice/numbers", h.voiceLKPhone.ListNumbers)
				ch.Post("/voice/numbers/search", h.voiceLKPhone.SearchNumbers)
				ch.Post("/voice/numbers/purchase", h.voiceLKPhone.PurchaseNumber)
			})

			// Reporting routes (admin/owner role required).
			authed.Route("/orgs/{org}/reports", func(rpt chi.Router) {
				rpt.Use(reporting.RequireOrgAdminOrOwner(cfg.DB))
				rpt.Get("/support", h.reportHandler.GetSupportMetrics)
				rpt.Get("/support/export", h.reportHandler.GetSupportExport)
				rpt.Get("/sales", h.reportHandler.GetSalesMetrics)
				rpt.Get("/sales/export", h.reportHandler.GetSalesExport)
			})

			// Pipeline stages route.
			authed.Get("/orgs/{org}/pipeline/stages", h.pipelineHandler.GetStages)

			// Notification routes.
			authed.Route("/notifications", func(n chi.Router) {
				n.Get("/", h.notifHandler.List)
				n.Patch("/{id}/read", h.notifHandler.MarkRead)
				n.Post("/mark-all-read", h.notifHandler.MarkAllRead)
				n.Get("/preferences", h.notifHandler.GetPreferences)
				n.Put("/preferences", h.notifHandler.UpdatePreferences)
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
