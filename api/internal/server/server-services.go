package server

import (
	"context"
	"log/slog"

	"github.com/abraderAI/crm-project/api/internal/admin"
	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/billing"
	"github.com/abraderAI/crm-project/api/internal/board"
	"github.com/abraderAI/crm-project/api/internal/channel"
	"github.com/abraderAI/crm-project/api/internal/channel/chat"
	emailpkg "github.com/abraderAI/crm-project/api/internal/channel/email"
	voicelk "github.com/abraderAI/crm-project/api/internal/channel/voice"
	"github.com/abraderAI/crm-project/api/internal/conversion"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/gdpr"
	"github.com/abraderAI/crm-project/api/internal/globalspace"
	"github.com/abraderAI/crm-project/api/internal/health"
	"github.com/abraderAI/crm-project/api/internal/llm"
	"github.com/abraderAI/crm-project/api/internal/membership"
	"github.com/abraderAI/crm-project/api/internal/message"
	"github.com/abraderAI/crm-project/api/internal/moderation"
	"github.com/abraderAI/crm-project/api/internal/notification"
	"github.com/abraderAI/crm-project/api/internal/org"
	"github.com/abraderAI/crm-project/api/internal/pipeline"
	"github.com/abraderAI/crm-project/api/internal/provision"
	"github.com/abraderAI/crm-project/api/internal/reporting"
	"github.com/abraderAI/crm-project/api/internal/revision"
	"github.com/abraderAI/crm-project/api/internal/scoring"
	"github.com/abraderAI/crm-project/api/internal/search"
	"github.com/abraderAI/crm-project/api/internal/space"
	"github.com/abraderAI/crm-project/api/internal/support"
	"github.com/abraderAI/crm-project/api/internal/thread"
	"github.com/abraderAI/crm-project/api/internal/tier"
	"github.com/abraderAI/crm-project/api/internal/upload"
	"github.com/abraderAI/crm-project/api/internal/voice"
	"github.com/abraderAI/crm-project/api/internal/vote"
	"github.com/abraderAI/crm-project/api/internal/webhook"
	ws "github.com/abraderAI/crm-project/api/internal/websocket"
)

// serverHandlers holds all initialized request handlers and selected services
// needed by the router's middleware and route closures.
type serverHandlers struct {
	jwtValidator       *auth.JWTValidator
	apiKeyService      *auth.APIKeyService
	apiKeyHandler      *auth.APIKeyHandler
	wsHub              *ws.Hub
	wsHandler          *ws.Handler
	healthHandler      *health.Handler
	notifHandler       *notification.Handler
	orgHandler         *org.Handler
	spaceHandler       *space.Handler
	boardHandler       *board.Handler
	threadHandler      *thread.Handler
	msgHandler         *message.Handler
	memberHandler      *membership.Handler
	billingHandler     *billing.Handler
	voteHandler        *vote.Handler
	modHandler         *moderation.Handler
	voiceHandler       *voice.Handler
	gdprHandler        *gdpr.Handler
	searchHandler      *search.Handler
	uploadHandler      *upload.Handler // may be nil
	webhookHandler     *webhook.Handler
	auditHandler       *audit.Handler
	adminService       *admin.Service
	adminHandler       *admin.Handler
	revisionHandler    *revision.Handler
	pipelineHandler    *pipeline.Handler
	llmHandler         *llm.Handler
	provisionHandler   *provision.Handler
	reportHandler      *reporting.Handler
	tierHandler        *tier.Handler
	conversionHandler  *conversion.Handler
	chatHandler        *chat.Handler
	channelHandler     *channel.Handler
	emailInboxHandler  *channel.EmailInboxHandler
	inboxWatcher       *emailpkg.InboxWatcher
	voiceLKWebhook     *voicelk.WebhookHandler
	voiceLKBridge      *voicelk.BridgeHandler
	voiceLKPhone       *voicelk.PhoneHandler
	globalSpaceHandler *globalspace.Handler
	supportHandler     *support.Handler
}

// newHandlers initialises all domain services and HTTP handlers from cfg,
// wires event-bus subscriptions, and returns a serverHandlers value ready for
// use in the router.
func newHandlers(cfg Config) serverHandlers {
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

	// Health check.
	healthHandler := health.NewHandler(cfg.DB)

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
	var uploadService *upload.Service
	if storage, err := upload.NewLocalStorage(uploadDir); err == nil {
		uploadService = upload.NewService(cfg.DB, storage, maxUpload)
		uploadHandler = upload.NewHandler(uploadService, maxUpload)
	}

	webhookService := webhook.NewService(cfg.DB)
	webhookHandler := webhook.NewHandler(webhookService)
	// Subscribe webhook service to all events.
	eventBus.SubscribeAll(webhookService.HandleEvent)

	auditService := audit.NewService(cfg.DB)
	auditHandler := audit.NewHandler(auditService)

	// Admin service and handler.
	// WithClerkKey enables server-side profile enrichment from the Clerk Backend API
	// when JWT tokens do not include email/name claims.
	adminService := admin.NewService(cfg.DB).WithClerkKey(cfg.ClerkSecretKey)
	adminHandler := admin.NewHandler(adminService, auditService, gdprService, cfg.RBACPolicy)

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

	// IO Phase 1: Channel Gateway + EmailInbox management.
	channelRepo := channel.NewRepository(cfg.DB)
	channelSvc := channel.NewService(channelRepo)
	channelHandler := channel.NewHandler(channelSvc)

	// Email inbox CRUD + watcher.
	var inboxStorage upload.StorageProvider
	if storage, err := upload.NewLocalStorage(uploadDir); err == nil {
		inboxStorage = storage
	}
	inboxWatcher := emailpkg.NewInboxWatcher(cfg.DB, inboxStorage, cfg.Logger)
	inboxRepo := channel.NewEmailInboxRepository(cfg.DB)
	inboxSvc := channel.NewEmailInboxService(inboxRepo)
	emailInboxHandler := channel.NewEmailInboxHandler(inboxSvc, channelSvc, inboxWatcher)

	// Start inbox watcher in background (non-blocking; reconnects on failure).
	go func() {
		if err := inboxWatcher.Start(context.Background()); err != nil {
			cfg.Logger.Error("inbox watcher start failed", slog.String("error", err.Error()))
		}
	}()

	// IO Phase 3: Voice LiveKit integration.
	lkProvider := voicelk.NewMockProvider()
	voiceLKService := voicelk.NewService(cfg.DB, lkProvider, eventBus)
	voiceLKWebhook := voicelk.NewWebhookHandler(voiceLKService, cfg.LiveKitWebhookToken)
	voiceLKPhone := voicelk.NewPhoneHandler(lkProvider, cfg.DB)
	voiceLKBridge := voicelk.NewBridgeHandler(voiceLKService, cfg.InternalAPIKey)

	// Reporting.
	reportRepo := reporting.NewRepository(cfg.DB)
	reportService := reporting.NewService(reportRepo)
	reportHandler := reporting.NewHandler(reportService, cfg.DB)

	// Tier resolution service.
	tierRepo := tier.NewRepository(cfg.DB)
	tierService := tier.NewService(tierRepo)
	tierHandler := tier.NewHandler(tierService)

	// Conversion service (Tier 2→3 upgrade flows).
	conversionService := conversion.NewService(cfg.DB)
	conversionHandler := conversion.NewHandler(conversionService, auditService)

	// Global space handler (forum, support, leads — slug-based access).
	globalSpaceRepo := globalspace.NewRepository(cfg.DB)
	globalSpaceService := globalspace.NewService(globalSpaceRepo, cfg.EventBus, uploadService)
	globalSpaceService.SetVoteService(voteService)
	globalSpaceHandler := globalspace.NewHandler(globalSpaceService)

	// Support ticket entry handler.
	supportRepo := support.NewRepository(cfg.DB)
	supportSvc := support.NewService(supportRepo, eventBus)
	supportHandler := support.NewHandler(supportSvc)

	// Inject ticket numberer into globalspace so new support tickets get #N assigned.
	globalspace.SetTicketNumberer(supportRepo)

	// IO Phase 4: AI Web Chat Widget.
	chatJWTSecret := cfg.ChatJWTSecret
	if chatJWTSecret == "" {
		chatJWTSecret = "chat-default-secret" // Default for dev; must be overridden in prod.
	}
	chatRepo := chat.NewRepository(cfg.DB)
	chatService := chat.NewService(chatRepo, llmProvider, wsHub, chatJWTSecret)
	chatHandler := chat.NewHandler(chatService, chatJWTSecret)

	return serverHandlers{
		jwtValidator:       jwtValidator,
		apiKeyService:      apiKeyService,
		apiKeyHandler:      apiKeyHandler,
		wsHub:              wsHub,
		wsHandler:          wsHandler,
		healthHandler:      healthHandler,
		notifHandler:       notifHandler,
		orgHandler:         orgHandler,
		spaceHandler:       spaceHandler,
		boardHandler:       boardHandler,
		threadHandler:      threadHandler,
		msgHandler:         msgHandler,
		memberHandler:      memberHandler,
		billingHandler:     billingHandler,
		voteHandler:        voteHandler,
		modHandler:         modHandler,
		voiceHandler:       voiceHandler,
		gdprHandler:        gdprHandler,
		searchHandler:      searchHandler,
		uploadHandler:      uploadHandler,
		webhookHandler:     webhookHandler,
		auditHandler:       auditHandler,
		adminService:       adminService,
		adminHandler:       adminHandler,
		revisionHandler:    revisionHandler,
		pipelineHandler:    pipelineHandler,
		llmHandler:         llmHandler,
		provisionHandler:   provisionHandler,
		reportHandler:      reportHandler,
		tierHandler:        tierHandler,
		conversionHandler:  conversionHandler,
		chatHandler:        chatHandler,
		channelHandler:     channelHandler,
		emailInboxHandler:  emailInboxHandler,
		inboxWatcher:       inboxWatcher,
		voiceLKWebhook:     voiceLKWebhook,
		voiceLKBridge:      voiceLKBridge,
		voiceLKPhone:       voiceLKPhone,
		globalSpaceHandler: globalSpaceHandler,
		supportHandler:     supportHandler,
	}
}
