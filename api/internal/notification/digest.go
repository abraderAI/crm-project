package notification

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// DigestEngine aggregates unread notifications and sends digest emails.
type DigestEngine struct {
	repo     *Repository
	sender   EmailSender
	resolver UserEmailResolver
	logger   *slog.Logger
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewDigestEngine creates a new digest engine.
func NewDigestEngine(repo *Repository, sender EmailSender, resolver UserEmailResolver, logger *slog.Logger, interval time.Duration) *DigestEngine {
	if interval == 0 {
		interval = 24 * time.Hour
	}
	return &DigestEngine{
		repo:     repo,
		sender:   sender,
		resolver: resolver,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the digest background loop.
func (d *DigestEngine) Start() {
	d.wg.Add(1)
	go d.run()
}

// Stop signals the digest engine to stop and waits for it to finish.
func (d *DigestEngine) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}

// run is the background loop that periodically sends digests.
func (d *DigestEngine) run() {
	defer d.wg.Done()

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.sendDigests(context.Background(), "daily")
		case <-d.stopCh:
			return
		}
	}
}

// sendDigests sends digest emails to all users with the given frequency.
func (d *DigestEngine) sendDigests(ctx context.Context, frequency string) {
	userIDs, err := d.repo.GetUsersWithDigestEnabled(ctx, frequency)
	if err != nil {
		d.logger.Error("failed to get digest users", slog.String("error", err.Error()))
		return
	}

	for _, userID := range userIDs {
		if err := d.sendUserDigest(ctx, userID); err != nil {
			d.logger.Error("failed to send digest",
				slog.String("user_id", userID),
				slog.String("error", err.Error()),
			)
		}
	}
}

// SendUserDigest sends a digest email for a specific user. Exported for testing.
func (d *DigestEngine) SendUserDigest(ctx context.Context, userID string) error {
	return d.sendUserDigest(ctx, userID)
}

// sendUserDigest aggregates unreads and sends a digest email.
func (d *DigestEngine) sendUserDigest(ctx context.Context, userID string) error {
	notifs, err := d.repo.GetUnreadNotifications(ctx, userID)
	if err != nil {
		return err
	}
	if len(notifs) == 0 {
		return nil
	}

	email, err := d.resolver.ResolveEmail(ctx, userID)
	if err != nil {
		return err
	}
	if email == "" {
		return nil
	}

	body := RenderDigestEmail(notifs)
	return d.sender.SendEmail(ctx, email, "Your Notification Digest", body)
}

// SendDigestsNow triggers digest sending immediately (for testing).
func (d *DigestEngine) SendDigestsNow(ctx context.Context, frequency string) {
	d.sendDigests(ctx, frequency)
}

// CreateDigestNotification creates a summary notification for a digest.
func CreateDigestNotification(userID string, count int) *models.Notification {
	return &models.Notification{
		UserID: userID,
		Type:   TypeDigest,
		Title:  "Notification Digest",
		Body:   fmt.Sprintf("You have %d unread notifications", count),
	}
}
