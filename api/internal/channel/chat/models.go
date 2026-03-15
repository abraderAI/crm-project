// Package chat provides the AI web chat widget backend: anonymous session auth,
// fingerprint-based visitor tracking, AI-powered chat responses, lead capture,
// and human escalation.
package chat

import "github.com/abraderAI/crm-project/api/internal/models"

// Type aliases so the rest of the chat package can use the short names.
type (
	ChatSession = models.ChatSession
	ChatVisitor = models.ChatVisitor
)
