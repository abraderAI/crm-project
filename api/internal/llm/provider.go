// Package llm provides LLM enrichment for CRM threads via a provider-abstracted
// interface (GrokProvider → OpenAI/Anthropic swappable).
package llm

import (
	"context"
	"fmt"
	"time"
)

// LLMProvider defines the interface for AI enrichment operations.
// All LLM integrations MUST implement this interface.
type LLMProvider interface {
	// Summarize generates a summary of the provided content.
	Summarize(ctx context.Context, input SummarizeInput) (*Summary, error)
	// SuggestNextAction suggests the next action for a lead/thread.
	SuggestNextAction(ctx context.Context, input SuggestInput) (*Suggestion, error)
}

// SummarizeInput holds data for a summarization request.
type SummarizeInput struct {
	ThreadID string `json:"thread_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Metadata string `json:"metadata"`
}

// Summary holds the summarization result.
type Summary struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// SuggestInput holds data for a next-action suggestion request.
type SuggestInput struct {
	ThreadID string `json:"thread_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Stage    string `json:"stage"`
	Metadata string `json:"metadata"`
}

// Suggestion holds the next-action suggestion result.
type Suggestion struct {
	Action    string    `json:"action"`
	Reasoning string    `json:"reasoning"`
	CreatedAt time.Time `json:"created_at"`
}

// EnrichResult holds the combined enrichment result.
type EnrichResult struct {
	ThreadID   string      `json:"thread_id"`
	Summary    *Summary    `json:"summary,omitempty"`
	Suggestion *Suggestion `json:"suggestion,omitempty"`
}

// GrokProvider implements LLMProvider as a stub.
// In production, this would call the Grok API.
type GrokProvider struct{}

// NewGrokProvider creates a new Grok LLM provider stub.
func NewGrokProvider() *GrokProvider {
	return &GrokProvider{}
}

// Summarize generates a mock summary.
func (g *GrokProvider) Summarize(_ context.Context, input SummarizeInput) (*Summary, error) {
	if input.Title == "" && input.Body == "" {
		return nil, fmt.Errorf("title or body is required for summarization")
	}

	// Stub: return a deterministic summary based on input.
	summary := fmt.Sprintf("Lead '%s' is a potential opportunity. ", input.Title)
	if input.Body != "" {
		maxLen := 100
		body := input.Body
		if len(body) > maxLen {
			body = body[:maxLen]
		}
		summary += fmt.Sprintf("Key details: %s", body)
	}

	return &Summary{
		Text:      summary,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// SuggestNextAction generates a mock next-action suggestion.
func (g *GrokProvider) SuggestNextAction(_ context.Context, input SuggestInput) (*Suggestion, error) {
	if input.ThreadID == "" {
		return nil, fmt.Errorf("thread_id is required for suggestion")
	}

	// Stub: suggest based on current stage.
	action := "Follow up with the lead"
	reasoning := "Standard follow-up action"

	switch input.Stage {
	case "new_lead":
		action = "Schedule initial contact with the lead"
		reasoning = "New leads should be contacted within 24 hours for best conversion rates"
	case "contacted":
		action = "Send qualification questionnaire"
		reasoning = "Lead has been contacted, next step is to qualify their needs and budget"
	case "qualified":
		action = "Prepare and send proposal"
		reasoning = "Lead is qualified, a tailored proposal will move them forward"
	case "proposal":
		action = "Schedule proposal review meeting"
		reasoning = "Proposal is out, a review meeting accelerates decision making"
	case "negotiation":
		action = "Address remaining objections and prepare contract"
		reasoning = "In negotiation phase, focus on closing the deal"
	}

	return &Suggestion{
		Action:    action,
		Reasoning: reasoning,
		CreatedAt: time.Now().UTC(),
	}, nil
}
