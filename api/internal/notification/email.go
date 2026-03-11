package notification

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// EmailSender is the interface for sending emails (Resend or mock).
type EmailSender interface {
	// SendEmail sends an email to the specified address.
	SendEmail(ctx context.Context, to, subject, htmlBody string) error
}

// UserEmailResolver resolves a user ID to an email address.
type UserEmailResolver interface {
	// ResolveEmail returns the email for a given user ID.
	ResolveEmail(ctx context.Context, userID string) (string, error)
}

// EmailProvider sends notifications via email.
type EmailProvider struct {
	sender   EmailSender
	resolver UserEmailResolver
	logger   *slog.Logger
}

// NewEmailProvider creates a new email notification provider.
func NewEmailProvider(sender EmailSender, resolver UserEmailResolver, logger *slog.Logger) *EmailProvider {
	return &EmailProvider{
		sender:   sender,
		resolver: resolver,
		logger:   logger,
	}
}

// Name returns the provider name.
func (p *EmailProvider) Name() string {
	return ChannelEmail
}

// Send sends a notification via email.
func (p *EmailProvider) Send(ctx context.Context, notif *models.Notification) error {
	email, err := p.resolver.ResolveEmail(ctx, notif.UserID)
	if err != nil {
		return fmt.Errorf("resolving email for user %s: %w", notif.UserID, err)
	}
	if email == "" {
		p.logger.Debug("no email for user, skipping", slog.String("user_id", notif.UserID))
		return nil
	}

	subject := notif.Title
	body := RenderNotificationEmail(notif)

	if err := p.sender.SendEmail(ctx, email, subject, body); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

// RenderNotificationEmail renders a notification as an HTML email body.
func RenderNotificationEmail(notif *models.Notification) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
<h2>%s</h2>
<p>%s</p>
<p style="color:#666;font-size:12px">Notification type: %s</p>
</body>
</html>`, notif.Title, notif.Body, notif.Type)
}

// RenderDigestEmail renders a digest email for multiple notifications.
func RenderDigestEmail(notifs []models.Notification) string {
	html := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
<h2>Your Notification Digest</h2>
<p>You have ` + fmt.Sprintf("%d", len(notifs)) + ` unread notifications:</p>
<ul>`
	for _, n := range notifs {
		html += fmt.Sprintf("<li><strong>%s</strong>: %s</li>", n.Title, n.Body)
	}
	html += `</ul>
</body>
</html>`
	return html
}

// NoOpEmailSender is a no-op email sender for testing/development.
type NoOpEmailSender struct {
	Logger *slog.Logger
	Sent   []SentEmail // Captured emails for testing.
}

// SentEmail records a sent email for testing.
type SentEmail struct {
	To      string
	Subject string
	Body    string
}

// SendEmail logs the email without actually sending it.
func (s *NoOpEmailSender) SendEmail(_ context.Context, to, subject, htmlBody string) error {
	if s.Logger != nil {
		s.Logger.Debug("no-op email sent",
			slog.String("to", to),
			slog.String("subject", subject),
		)
	}
	s.Sent = append(s.Sent, SentEmail{To: to, Subject: subject, Body: htmlBody})
	return nil
}

// NoOpEmailResolver returns empty email for all users (development/test).
type NoOpEmailResolver struct {
	Emails map[string]string // user_id → email
}

// ResolveEmail returns the mapped email or empty string.
func (r *NoOpEmailResolver) ResolveEmail(_ context.Context, userID string) (string, error) {
	if r.Emails == nil {
		return "", nil
	}
	return r.Emails[userID], nil
}
