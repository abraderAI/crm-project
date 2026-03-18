package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/abraderAI/crm-project/api/internal/models"
	ws "github.com/abraderAI/crm-project/api/internal/websocket"
)

// emailRegex matches common email patterns.
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// detectAndCaptureLead checks if the message contains an email or name and
// updates the visitor and thread metadata accordingly.
func (s *Service) detectAndCaptureLead(ctx context.Context, session *ChatSession, message string) {
	visitor, err := s.repo.FindVisitor(ctx, session.VisitorID)
	if err != nil || visitor == nil {
		return
	}

	updated := false

	// Detect email.
	if email := emailRegex.FindString(message); email != "" && visitor.ContactEmail == "" {
		visitor.ContactEmail = email
		updated = true
	}

	// Detect name pattern (e.g. "my name is ..." or "I'm ...").
	nameLower := strings.ToLower(message)
	if visitor.ContactName == "" {
		if idx := strings.Index(nameLower, "my name is "); idx != -1 {
			name := extractName(message[idx+11:])
			if name != "" {
				visitor.ContactName = name
				updated = true
			}
		} else if idx := strings.Index(nameLower, "i'm "); idx != -1 {
			name := extractName(message[idx+4:])
			if name != "" {
				visitor.ContactName = name
				updated = true
			}
		}
	}

	if updated {
		_ = s.repo.UpdateVisitor(ctx, visitor)
		// Update thread metadata with lead info.
		if session.ThreadID != "" {
			meta := fmt.Sprintf(`{"source":"chat_widget","contact_email":%q,"contact_name":%q,"visitor_id":%q}`,
				visitor.ContactEmail, visitor.ContactName, visitor.ID)
			_ = s.repo.UpdateThreadMetadata(ctx, session.ThreadID, meta)
		}
		// Create or update a lead record with anon_session_id linked to the visitor.
		_, _ = s.repo.CreateOrUpdateLead(ctx, visitor.ID, visitor.ContactEmail, visitor.ContactName, "chatbot")
	}
}

// extractName extracts a name from text following "my name is" or "I'm".
func extractName(text string) string {
	text = strings.TrimSpace(text)
	// Take up to first punctuation or end of segment.
	for i, c := range text {
		if c == '.' || c == ',' || c == '!' || c == '?' || c == '\n' {
			text = text[:i]
			break
		}
	}
	text = strings.TrimSpace(text)
	// Limit to reasonable name length.
	if len(text) > 100 {
		text = text[:100]
	}
	// Must contain at least one letter.
	hasLetter := false
	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return ""
	}
	return text
}

// escalationPatterns are phrases that indicate a visitor wants to speak to a human.
var escalationPatterns = []string{
	"speak to a human",
	"talk to a person",
	"real person",
	"human agent",
	"speak to someone",
	"talk to an agent",
	"connect me to",
	"transfer me to",
	"escalate",
	"speak with a representative",
	"live agent",
	"customer service",
}

// detectEscalation checks if the message contains escalation intent.
func (s *Service) detectEscalation(message string) bool {
	lower := strings.ToLower(message)
	for _, pattern := range escalationPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// handleEscalation marks the session as escalated, creates a support thread
// in the global-support space, and broadcasts a notification to DEFT support.
func (s *Service) handleEscalation(ctx context.Context, session *ChatSession) error {
	session.Escalated = true
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("marking session as escalated: %w", err)
	}

	// Update thread metadata.
	if session.ThreadID != "" {
		meta := fmt.Sprintf(`{"source":"chat_widget","escalated":true,"escalated_at":%q,"visitor_id":%q}`,
			time.Now().UTC().Format(time.RFC3339), session.VisitorID)
		_ = s.repo.UpdateThreadMetadata(ctx, session.ThreadID, meta)
	}

	// Create a support thread in global-support space for DEFT support agents.
	_ = s.createSupportThread(ctx, session)

	// Broadcast escalation to CRM agents.
	if s.wsHub != nil {
		s.wsHub.Broadcast(fmt.Sprintf("escalation:%s", session.OrgID), ws.BroadcastMessage{
			Type:    "chat.escalated",
			Channel: fmt.Sprintf("escalation:%s", session.OrgID),
			Payload: map[string]any{
				"session_id": session.ID,
				"org_id":     session.OrgID,
				"thread_id":  session.ThreadID,
				"visitor_id": session.VisitorID,
			},
		})
	}
	return nil
}

// createSupportThread creates a thread in the global-support space for DEFT agents.
func (s *Service) createSupportThread(ctx context.Context, session *ChatSession) error {
	board, err := s.repo.FindGlobalSupportBoard(ctx)
	if err != nil || board == nil {
		return fmt.Errorf("no global-support board available")
	}

	// Collect visitor info for the support thread.
	visitorName := "Anonymous visitor"
	visitorEmail := ""
	visitor, err := s.repo.FindVisitor(ctx, session.VisitorID)
	if err == nil && visitor != nil {
		if visitor.ContactName != "" {
			visitorName = visitor.ContactName
		}
		visitorEmail = visitor.ContactEmail
	}

	slugID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generating slug: %w", err)
	}

	thread := &models.Thread{
		BoardID:    board.ID,
		Title:      fmt.Sprintf("Chat escalation — %s", visitorName),
		Slug:       fmt.Sprintf("escalation-%s", slugID.String()[:8]),
		AuthorID:   "system",
		ThreadType: models.ThreadTypeSupport,
		Visibility: models.ThreadVisibilityDeftOnly,
		OrgID:      &session.OrgID,
		Metadata: fmt.Sprintf(
			`{"source":"chat_escalation","session_id":%q,"visitor_id":%q,"visitor_email":%q,"original_thread_id":%q}`,
			session.ID, session.VisitorID, visitorEmail, session.ThreadID),
	}
	if err := s.repo.CreateThread(ctx, thread); err != nil {
		return fmt.Errorf("creating support thread: %w", err)
	}

	// Post the escalation context as first message.
	body := fmt.Sprintf("Visitor %s has requested to speak with a human agent.", visitorName)
	if visitorEmail != "" {
		body += fmt.Sprintf("\nContact email: %s", visitorEmail)
	}
	body += fmt.Sprintf("\nOriginal chat session: %s", session.ID)

	msg := &models.Message{
		ThreadID: thread.ID,
		Body:     body,
		AuthorID: "system",
		Type:     models.MessageTypeSystem,
		Metadata: fmt.Sprintf(`{"source":"chat_escalation","session_id":%q}`, session.ID),
	}
	_ = s.repo.CreateMessage(ctx, msg)

	return nil
}

// PromoteSession links an anonymous session to a newly registered user.
// It updates the lead record's user_id and status from anonymous to registered.
func (s *Service) PromoteSession(anonSessionID, userID string) error {
	ctx := context.Background()
	return s.repo.PromoteAnonymousSession(ctx, anonSessionID, userID)
}

// ResumeAfterEscalationTimeout is called when no human agent answers within
// the configured timeout. It resumes the AI conversation with an apology.
func (s *Service) ResumeAfterEscalationTimeout(ctx context.Context, sessionID string) error {
	session, err := s.repo.FindSession(ctx, sessionID)
	if err != nil || session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	if !session.Escalated {
		return nil
	}

	// Store apology message.
	apology := "I apologize, but all of our agents are currently busy. I'm here to help — please let me know how I can assist you."
	msg := &models.Message{
		ThreadID: session.ThreadID,
		Body:     apology,
		AuthorID: "ai",
		Type:     models.MessageTypeComment,
		Metadata: fmt.Sprintf(`{"source":"ai_responder","escalation_timeout":true,"session_id":%q}`, session.ID),
	}
	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("storing escalation timeout message: %w", err)
	}

	// Broadcast the apology.
	if s.wsHub != nil {
		s.wsHub.Broadcast(fmt.Sprintf("chat:%s", session.ID), ws.BroadcastMessage{
			Type:    "chat.message",
			Channel: fmt.Sprintf("chat:%s", session.ID),
			Payload: map[string]any{
				"message_id":          msg.ID,
				"body":                apology,
				"author":              "ai",
				"type":                "escalation_timeout",
				"escalation_resolved": true,
			},
		})
	}

	session.Escalated = false
	return s.repo.UpdateSession(ctx, session)
}
