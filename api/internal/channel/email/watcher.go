package email

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/channel"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/upload"
)

// InboxWatcher manages live IMAP IDLE connections for all enabled EmailInbox
// records in the database. Each inbox gets its own IDLEManager (keyed by inbox
// ID) that reconnects automatically on failure. Call Start once at server
// startup and Stop during graceful shutdown.
type InboxWatcher struct {
	db       *gorm.DB
	emailSvc *Service
	registry *IDLEManagerRegistry
	logger   *slog.Logger
}

// NewInboxWatcher creates a watcher that will use the given database and
// storage provider for attachment handling. A nil logger falls back to the
// default slog logger.
func NewInboxWatcher(db *gorm.DB, storage upload.StorageProvider, logger *slog.Logger) *InboxWatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &InboxWatcher{
		db:       db,
		emailSvc: NewService(db, storage, nil),
		registry: NewIDLEManagerRegistry(),
		logger:   logger,
	}
}

// Start loads all enabled EmailInbox records from the database and starts an
// IMAP IDLE manager for each one. It is safe to call Start before the HTTP
// server begins accepting requests.
func (w *InboxWatcher) Start(ctx context.Context) error {
	var inboxes []models.EmailInbox
	if err := w.db.WithContext(ctx).
		Where("enabled = ? AND deleted_at IS NULL", true).
		Find(&inboxes).Error; err != nil {
		return fmt.Errorf("loading email inboxes: %w", err)
	}

	for _, inbox := range inboxes {
		w.startInbox(inbox)
	}

	w.logger.Info("email inbox watcher started", "inboxes", len(inboxes))
	return nil
}

// Stop gracefully shuts down all IDLE managers.
func (w *InboxWatcher) Stop() {
	w.registry.StopAll()
	w.logger.Info("email inbox watcher stopped")
}

// RestartInbox stops the existing IDLE manager for an inbox (if any) and
// starts a new one when the inbox is enabled. Used after config changes.
func (w *InboxWatcher) RestartInbox(inbox models.EmailInbox) {
	w.registry.Deregister(inbox.ID)
	if inbox.Enabled {
		w.startInbox(inbox)
	}
}

// startInbox registers and starts a single IDLE manager for inbox.
func (w *InboxWatcher) startInbox(inbox models.EmailInbox) {
	mailbox := inbox.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}

	// Capture immutable fields for the closure; never capture inbox by pointer.
	inboxID := inbox.ID
	orgID := inbox.OrgID
	routingAction := inbox.RoutingAction
	provider := NewLiveIMAPProvider(w.logger)

	emailCfg := channel.EmailConfig{
		IMAPHost: inbox.IMAPHost,
		IMAPPort: inbox.IMAPPort,
		Username: inbox.Username,
		Password: inbox.Password,
		Mailbox:  mailbox,
	}

	cfg := IDLEManagerConfig{
		OrgID:       orgID,
		EmailConfig: emailCfg,
		Provider:    provider,
		OnMessage: func(uid uint32) {
			ctx := context.Background()

			// FetchMessage is safe here: it is called from within the IDLE
			// loop handler, which runs after IDLE has been exited.
			msg, err := provider.FetchMessage(ctx, uid)
			if err != nil {
				w.logger.Error("fetching IMAP message",
					"inbox_id", inboxID,
					"uid", uid,
					"error", err,
				)
				return
			}

			if _, err := w.emailSvc.ProcessInbound(ctx, orgID, routingAction, msg); err != nil {
				w.logger.Error("processing inbound email",
					"inbox_id", inboxID,
					"uid", uid,
					"error", err,
				)
			}
		},
		Logger: w.logger,
	}

	mgr := NewIDLEManager(cfg)
	w.registry.Register(inboxID, mgr)
}
