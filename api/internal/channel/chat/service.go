package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/abraderAI/crm-project/api/internal/channel"
	"github.com/abraderAI/crm-project/api/internal/llm"
	"github.com/abraderAI/crm-project/api/internal/models"
	ws "github.com/abraderAI/crm-project/api/internal/websocket"
)

// Service provides business logic for the chat widget backend.
type Service struct {
	repo        *Repository
	llmProvider llm.LLMProvider
	wsHub       *ws.Hub
	jwtSecret   string
}

// NewService creates a new chat Service.
func NewService(repo *Repository, llmProvider llm.LLMProvider, wsHub *ws.Hub, jwtSecret string) *Service {
	return &Service{
		repo:        repo,
		llmProvider: llmProvider,
		wsHub:       wsHub,
		jwtSecret:   jwtSecret,
	}
}

// CreateSessionInput holds the data for creating a new chat session.
type CreateSessionInput struct {
	EmbedKey        string `json:"embed_key"`
	FingerprintHash string `json:"fingerprint_hash"`
}

// CreateSessionOutput holds the result of session creation.
type CreateSessionOutput struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	VisitorID string `json:"visitor_id"`
	ExpiresAt int64  `json:"expires_at"`
	Returning bool   `json:"returning"`
	Greeting  string `json:"greeting,omitempty"`
}

// CreateSession validates the embed key, creates or finds a visitor,
// creates a session, and returns a signed JWT.
func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (*CreateSessionOutput, error) {
	if input.EmbedKey == "" {
		return nil, fmt.Errorf("embed_key is required")
	}
	if input.FingerprintHash == "" {
		return nil, fmt.Errorf("fingerprint_hash is required")
	}

	// Validate embed key against channel config.
	cfg, err := s.repo.FindChannelConfigByEmbedKey(ctx, input.EmbedKey)
	if err != nil {
		return nil, fmt.Errorf("looking up embed key: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("invalid embed key")
	}

	// Parse chat config for greeting message.
	var chatCfg channel.ChatConfig
	if cfg.Settings != "" && cfg.Settings != "{}" {
		_ = json.Unmarshal([]byte(cfg.Settings), &chatCfg)
	}

	// Find or create visitor.
	visitor, isNew, err := s.repo.FindOrCreateVisitor(ctx, cfg.OrgID, input.FingerprintHash)
	if err != nil {
		return nil, fmt.Errorf("resolving visitor: %w", err)
	}

	// Create session.
	expiresAt := time.Now().Add(SessionTokenDuration)
	session := &ChatSession{
		OrgID:           cfg.OrgID,
		EmbedKey:        input.EmbedKey,
		FingerprintHash: input.FingerprintHash,
		VisitorID:       visitor.ID,
		ExpiresAt:       expiresAt,
	}

	// If returning visitor has an existing thread, link it.
	if !isNew && visitor.LastThreadID != "" {
		session.ThreadID = visitor.LastThreadID
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	// Update visitor's last session.
	visitor.LastSessionID = session.ID
	if err := s.repo.UpdateVisitor(ctx, visitor); err != nil {
		return nil, fmt.Errorf("updating visitor session link: %w", err)
	}

	// Issue JWT.
	token, err := IssueSessionToken(s.jwtSecret, SessionClaims{
		SessionID: session.ID,
		OrgID:     cfg.OrgID,
		VisitorID: visitor.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("issuing token: %w", err)
	}

	return &CreateSessionOutput{
		Token:     token,
		SessionID: session.ID,
		VisitorID: visitor.ID,
		ExpiresAt: expiresAt.Unix(),
		Returning: !isNew,
		Greeting:  chatCfg.WidgetTheme.Greeting,
	}, nil
}

// HandleChatMessage processes an inbound chat message from a widget visitor.
// It creates a CRM thread if needed, stores the user message, generates an AI
// response, and broadcasts the response via WebSocket.
func (s *Service) HandleChatMessage(ctx context.Context, claims *SessionClaims, messageBody string) (*ChatMessageResponse, error) {
	if messageBody == "" {
		return nil, fmt.Errorf("message body is required")
	}

	// Load session.
	session, err := s.repo.FindSession(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("finding session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Get or create thread.
	threadID := session.ThreadID
	if threadID == "" {
		thread, err := s.createChatThread(ctx, session)
		if err != nil {
			return nil, fmt.Errorf("creating chat thread: %w", err)
		}
		threadID = thread.ID
		session.ThreadID = threadID
		if err := s.repo.UpdateSession(ctx, session); err != nil {
			return nil, fmt.Errorf("linking thread to session: %w", err)
		}
		// Update visitor's last thread.
		visitor, err := s.repo.FindVisitor(ctx, session.VisitorID)
		if err == nil && visitor != nil {
			visitor.LastThreadID = threadID
			_ = s.repo.UpdateVisitor(ctx, visitor)
		}
	}

	// Store user message.
	userMsg := &models.Message{
		ThreadID: threadID,
		Body:     messageBody,
		AuthorID: "visitor:" + session.VisitorID,
		Type:     models.MessageTypeComment,
		Metadata: fmt.Sprintf(`{"source":"chat_widget","session_id":%q,"visitor_id":%q}`, session.ID, session.VisitorID),
	}
	if err := s.repo.CreateMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("storing user message: %w", err)
	}

	// Check for lead capture (email/name in message).
	s.detectAndCaptureLead(ctx, session, messageBody)

	// Check for escalation intent.
	if s.detectEscalation(messageBody) && !session.Escalated {
		if err := s.handleEscalation(ctx, session); err != nil {
			return nil, fmt.Errorf("handling escalation: %w", err)
		}
		return &ChatMessageResponse{
			Type:    "escalation",
			Message: "Connecting you to a human agent. Please wait...",
		}, nil
	}

	// Generate AI response.
	aiResponse, err := s.generateAIResponse(ctx, session, messageBody)
	if err != nil {
		// On AI failure, return a fallback message.
		aiResponse = "I apologize, but I'm having trouble processing your request. Please try again or ask to speak with a human agent."
	}

	// Store AI response.
	aiMsg := &models.Message{
		ThreadID: threadID,
		Body:     aiResponse,
		AuthorID: "ai",
		Type:     models.MessageTypeComment,
		Metadata: fmt.Sprintf(`{"source":"ai_responder","session_id":%q}`, session.ID),
	}
	if err := s.repo.CreateMessage(ctx, aiMsg); err != nil {
		return nil, fmt.Errorf("storing AI response: %w", err)
	}

	// Broadcast AI response via WebSocket.
	if s.wsHub != nil {
		s.wsHub.Broadcast(fmt.Sprintf("chat:%s", session.ID), ws.BroadcastMessage{
			Type:    "chat.message",
			Channel: fmt.Sprintf("chat:%s", session.ID),
			Payload: map[string]any{
				"message_id": aiMsg.ID,
				"body":       aiResponse,
				"author":     "ai",
				"type":       "ai_response",
			},
		})
	}

	return &ChatMessageResponse{
		Type:      "ai_response",
		Message:   aiResponse,
		MessageID: aiMsg.ID,
	}, nil
}

// ChatMessageResponse is the response sent back to the widget after processing a message.
type ChatMessageResponse struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	MessageID string `json:"message_id,omitempty"`
}

// createChatThread creates a new CRM thread for a chat session.
func (s *Service) createChatThread(ctx context.Context, session *ChatSession) (*models.Thread, error) {
	board, err := s.repo.FindFirstBoardInOrg(ctx, session.OrgID)
	if err != nil {
		return nil, fmt.Errorf("finding board: %w", err)
	}
	if board == nil {
		return nil, fmt.Errorf("no board found in org %s", session.OrgID)
	}

	slugID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating slug: %w", err)
	}

	thread := &models.Thread{
		BoardID:  board.ID,
		Title:    fmt.Sprintf("Chat session %s", session.ID[:8]),
		Slug:     fmt.Sprintf("chat-%s", slugID.String()[:8]),
		AuthorID: "system",
		Metadata: fmt.Sprintf(
			`{"source":"chat_widget","session_id":%q,"visitor_id":%q,"channel_type":"chat"}`,
			session.ID, session.VisitorID),
	}
	if err := s.repo.CreateThread(ctx, thread); err != nil {
		return nil, err
	}
	return thread, nil
}

// generateAIResponse uses the LLM provider to generate a response based on
// conversation history, RAG context from global-docs, and the org's system prompt.
func (s *Service) generateAIResponse(ctx context.Context, session *ChatSession, latestMessage string) (string, error) {
	if s.llmProvider == nil {
		// Stub mode: check global-docs for relevant content.
		if ragContext := s.buildRAGContext(ctx, latestMessage); ragContext != "" {
			return fmt.Sprintf("Based on our documentation: %s", ragContext), nil
		}
		return "Thank you for your message. How can I help you today?", nil
	}

	// Build conversation context.
	messages, _ := s.repo.ListThreadMessages(ctx, session.ThreadID)
	var history strings.Builder
	for _, msg := range messages {
		role := "visitor"
		if msg.AuthorID == "ai" {
			role = "assistant"
		}
		fmt.Fprintf(&history, "%s: %s\n", role, msg.Body)
	}

	// Retrieve RAG context from global-docs.
	ragSection := s.buildRAGContext(ctx, latestMessage)

	// Get system prompt from channel config.
	systemPrompt := "You are a helpful customer support assistant."
	cfg, err := s.repo.FindChannelConfigByEmbedKey(ctx, session.EmbedKey)
	if err == nil && cfg != nil {
		var chatCfg channel.ChatConfig
		if json.Unmarshal([]byte(cfg.Settings), &chatCfg) == nil && chatCfg.AISystemPrompt != "" {
			systemPrompt = chatCfg.AISystemPrompt
		}
	}

	// Build the full prompt with RAG context.
	var body strings.Builder
	if ragSection != "" {
		fmt.Fprintf(&body, "Relevant documentation:\n%s\n\n", ragSection)
	}
	fmt.Fprintf(&body, "Conversation history:\n%s\nLatest message: %s\n\nRespond helpfully to the visitor.", history.String(), latestMessage)

	// Use Summarize as a general-purpose LLM call.
	result, err := s.llmProvider.Summarize(ctx, llm.SummarizeInput{
		ThreadID: session.ThreadID,
		Title:    systemPrompt,
		Body:     body.String(),
		Metadata: fmt.Sprintf(`{"session_id":%q}`, session.ID),
	})
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}
	return result.Text, nil
}

// buildRAGContext retrieves relevant documentation from global-docs for the given query.
// Returns a formatted string of relevant content, or empty string if no matches.
func (s *Service) buildRAGContext(ctx context.Context, query string) string {
	// Extract keywords from the query (use first few significant words).
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	// Search for each significant word (skip very short words).
	seen := make(map[string]bool)
	var results []models.Thread
	for _, word := range words {
		if len(word) < 3 {
			continue
		}
		matches, err := s.repo.SearchGlobalDocs(ctx, word, 3)
		if err != nil || len(matches) == 0 {
			continue
		}
		for _, m := range matches {
			if !seen[m.ID] {
				seen[m.ID] = true
				results = append(results, m)
			}
		}
		if len(results) >= 5 {
			break
		}
	}

	if len(results) == 0 {
		return ""
	}

	// Format results as context.
	var buf strings.Builder
	for i, thread := range results {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&buf, "- %s: %s\n", thread.Title, truncate(thread.Body, 200))
	}
	return buf.String()
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
