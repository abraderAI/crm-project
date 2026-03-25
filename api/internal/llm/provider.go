// Package llm provides LLM enrichment for CRM threads via a provider-abstracted
// interface (GrokProvider → OpenAI/Anthropic swappable).
package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// LLMProvider defines the interface for AI enrichment operations.
// All LLM integrations MUST implement this interface.
type LLMProvider interface {
	// Summarize generates a summary of the provided content.
	Summarize(ctx context.Context, input SummarizeInput) (*Summary, error)
	// SuggestNextAction suggests the next action for a lead/thread.
	SuggestNextAction(ctx context.Context, input SuggestInput) (*Suggestion, error)

	// Briefing generates a daily briefing for a sales rep.
	Briefing(ctx context.Context, userID string, opps []models.Thread, tasks []CRMTask, msgs []models.Message) (string, error)
	// EmailSummary generates a 1-2 line summary of an inbound email.
	EmailSummary(ctx context.Context, email models.Message, entityThread models.Thread) (string, error)
	// PipelineStrategy generates a CEO-level strategic analysis of the pipeline.
	PipelineStrategy(ctx context.Context, opps []models.Thread) (string, error)
	// DealStrategy generates a "Close This Deal Now" strategy for a specific opportunity.
	DealStrategy(ctx context.Context, opp models.Thread, messages []models.Message, tasks []CRMTask) (string, error)
	// QualityMessage generates a human-readable message for data quality violations.
	QualityMessage(ctx context.Context, violations []QualityViolation, record models.Thread) (string, error)
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

// CRMTask represents a follow-up task for LLM context assembly.
// This is a simplified representation used by the LLM package to avoid
// a circular dependency on a crmtask package.
type CRMTask struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AssignedTo  string `json:"assigned_to"`
	DueDate     string `json:"due_date,omitempty"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	ParentID    string `json:"parent_id"`
}

// QualityViolation represents a data quality issue on a CRM record.
type QualityViolation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
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

// Briefing generates a deterministic daily briefing stub.
func (g *GrokProvider) Briefing(_ context.Context, userID string, opps []models.Thread, tasks []CRMTask, _ []models.Message) (string, error) {
	return fmt.Sprintf("Good morning! You have %d open opportunities and %d tasks. User: %s. Focus on high-priority deals today.", len(opps), len(tasks), userID), nil
}

// EmailSummary generates a deterministic email summary stub.
func (g *GrokProvider) EmailSummary(_ context.Context, email models.Message, entityThread models.Thread) (string, error) {
	snippet := email.Body
	if len(snippet) > 80 {
		snippet = snippet[:80]
	}
	return fmt.Sprintf("📧 New email on %s: %s", entityThread.Title, snippet), nil
}

// PipelineStrategy generates a deterministic pipeline strategy stub.
func (g *GrokProvider) PipelineStrategy(_ context.Context, opps []models.Thread) (string, error) {
	return fmt.Sprintf("Pipeline overview: %d open opportunities across all stages. Recommend focusing on deals in negotiation stage for quick wins.", len(opps)), nil
}

// DealStrategy generates a deterministic deal strategy stub.
func (g *GrokProvider) DealStrategy(_ context.Context, opp models.Thread, messages []models.Message, tasks []CRMTask) (string, error) {
	return fmt.Sprintf("Deal strategy for '%s': %d messages exchanged, %d tasks pending. Recommend scheduling a follow-up meeting to advance this opportunity.", opp.Title, len(messages), len(tasks)), nil
}

// QualityMessage generates a deterministic quality message stub.
func (g *GrokProvider) QualityMessage(_ context.Context, violations []QualityViolation, record models.Thread) (string, error) {
	fields := make([]string, len(violations))
	for i, v := range violations {
		fields[i] = v.Field
	}
	return fmt.Sprintf("Data quality alert for '%s': %d issues found. Please update: %v", record.Title, len(violations), fields), nil
}

// MockLLMProvider implements LLMProvider with configurable return values
// and call argument capture for testing.
type MockLLMProvider struct {
	mu sync.Mutex

	// Configurable responses.
	SummarizeResp *Summary
	SummarizeErr  error
	SuggestResp   *Suggestion
	SuggestErr    error
	BriefingResp  string
	BriefingErr   error
	EmailSumResp  string
	EmailSumErr   error
	PipelineResp  string
	PipelineErr   error
	DealResp      string
	DealErr       error
	QualityResp   string
	QualityErr    error

	// Call counters and captured args.
	SummarizeCalls int
	SuggestCalls   int
	BriefingCalls  int
	EmailSumCalls  int
	PipelineCalls  int
	DealCalls      int
	QualityCalls   int

	// Captured arguments for assertions.
	LastBriefingUserID    string
	LastBriefingOpps      []models.Thread
	LastBriefingTasks     []CRMTask
	LastBriefingMsgs      []models.Message
	LastEmailMsg          models.Message
	LastEmailThread       models.Thread
	LastPipelineOpps      []models.Thread
	LastDealOpp           models.Thread
	LastDealMsgs          []models.Message
	LastDealTasks         []CRMTask
	LastQualityViolations []QualityViolation
	LastQualityRecord     models.Thread
}

// NewMockLLMProvider creates a MockLLMProvider with sensible defaults.
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		BriefingResp: "Mock briefing response",
		EmailSumResp: "Mock email summary",
		PipelineResp: "Mock pipeline strategy",
		DealResp:     "Mock deal strategy",
		QualityResp:  "Mock quality message",
	}
}

// Summarize implements LLMProvider.
func (m *MockLLMProvider) Summarize(_ context.Context, _ SummarizeInput) (*Summary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SummarizeCalls++
	if m.SummarizeErr != nil {
		return nil, m.SummarizeErr
	}
	if m.SummarizeResp != nil {
		return m.SummarizeResp, nil
	}
	return &Summary{Text: "Mock summary", CreatedAt: time.Now().UTC()}, nil
}

// SuggestNextAction implements LLMProvider.
func (m *MockLLMProvider) SuggestNextAction(_ context.Context, _ SuggestInput) (*Suggestion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SuggestCalls++
	if m.SuggestErr != nil {
		return nil, m.SuggestErr
	}
	if m.SuggestResp != nil {
		return m.SuggestResp, nil
	}
	return &Suggestion{Action: "Mock action", Reasoning: "Mock reasoning", CreatedAt: time.Now().UTC()}, nil
}

// Briefing implements LLMProvider.
func (m *MockLLMProvider) Briefing(_ context.Context, userID string, opps []models.Thread, tasks []CRMTask, msgs []models.Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BriefingCalls++
	m.LastBriefingUserID = userID
	m.LastBriefingOpps = opps
	m.LastBriefingTasks = tasks
	m.LastBriefingMsgs = msgs
	return m.BriefingResp, m.BriefingErr
}

// EmailSummary implements LLMProvider.
func (m *MockLLMProvider) EmailSummary(_ context.Context, email models.Message, entityThread models.Thread) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EmailSumCalls++
	m.LastEmailMsg = email
	m.LastEmailThread = entityThread
	return m.EmailSumResp, m.EmailSumErr
}

// PipelineStrategy implements LLMProvider.
func (m *MockLLMProvider) PipelineStrategy(_ context.Context, opps []models.Thread) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PipelineCalls++
	m.LastPipelineOpps = opps
	return m.PipelineResp, m.PipelineErr
}

// DealStrategy implements LLMProvider.
func (m *MockLLMProvider) DealStrategy(_ context.Context, opp models.Thread, messages []models.Message, tasks []CRMTask) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DealCalls++
	m.LastDealOpp = opp
	m.LastDealMsgs = messages
	m.LastDealTasks = tasks
	return m.DealResp, m.DealErr
}

// QualityMessage implements LLMProvider.
func (m *MockLLMProvider) QualityMessage(_ context.Context, violations []QualityViolation, record models.Thread) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QualityCalls++
	m.LastQualityViolations = violations
	m.LastQualityRecord = record
	return m.QualityResp, m.QualityErr
}

// GetEmailSumCalls returns the call count (thread-safe).
func (m *MockLLMProvider) GetEmailSumCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.EmailSumCalls
}

// GetLastEmailMsg returns the last email message (thread-safe).
func (m *MockLLMProvider) GetLastEmailMsg() models.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.LastEmailMsg
}

// GetLastEmailThread returns the last email thread (thread-safe).
func (m *MockLLMProvider) GetLastEmailThread() models.Thread {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.LastEmailThread
}
