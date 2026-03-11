package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func createHierarchy(t *testing.T, db *gorm.DB) (*models.Org, *models.Space, *models.Board) {
	t.Helper()
	org := &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "CRM", Slug: "crm", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Pipeline", Slug: "pipeline", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return org, space, board
}

// mockProvider implements LLMProvider for testing.
type mockProvider struct {
	summarizeErr   error
	suggestErr     error
	summarizeResp  *Summary
	suggestResp    *Suggestion
	summarizeCalls int
	suggestCalls   int
}

func (m *mockProvider) Summarize(_ context.Context, _ SummarizeInput) (*Summary, error) {
	m.summarizeCalls++
	if m.summarizeErr != nil {
		return nil, m.summarizeErr
	}
	if m.summarizeResp != nil {
		return m.summarizeResp, nil
	}
	return &Summary{Text: "Test summary"}, nil
}

func (m *mockProvider) SuggestNextAction(_ context.Context, _ SuggestInput) (*Suggestion, error) {
	m.suggestCalls++
	if m.suggestErr != nil {
		return nil, m.suggestErr
	}
	if m.suggestResp != nil {
		return m.suggestResp, nil
	}
	return &Suggestion{Action: "Test action", Reasoning: "Test reasoning"}, nil
}

// --- GrokProvider Tests ---

func TestGrokProvider_Summarize_Success(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.Summarize(context.Background(), SummarizeInput{
		ThreadID: "t1", Title: "Big Deal", Body: "Details about the opportunity",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Text, "Big Deal")
	assert.Contains(t, result.Text, "Details about the opportunity")
	assert.False(t, result.CreatedAt.IsZero())
}

func TestGrokProvider_Summarize_TitleOnly(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.Summarize(context.Background(), SummarizeInput{
		ThreadID: "t1", Title: "Deal",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Text, "Deal")
}

func TestGrokProvider_Summarize_EmptyInputs(t *testing.T) {
	p := NewGrokProvider()
	_, err := p.Summarize(context.Background(), SummarizeInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title or body is required")
}

func TestGrokProvider_Summarize_LongBody(t *testing.T) {
	p := NewGrokProvider()
	longBody := ""
	for i := 0; i < 200; i++ {
		longBody += "a"
	}
	result, err := p.Summarize(context.Background(), SummarizeInput{
		Title: "Lead", Body: longBody,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Text)
}

func TestGrokProvider_SuggestNextAction_NewLead(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.SuggestNextAction(context.Background(), SuggestInput{
		ThreadID: "t1", Stage: "new_lead",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Action, "initial contact")
	assert.False(t, result.CreatedAt.IsZero())
}

func TestGrokProvider_SuggestNextAction_Contacted(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.SuggestNextAction(context.Background(), SuggestInput{
		ThreadID: "t1", Stage: "contacted",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Action, "qualification")
}

func TestGrokProvider_SuggestNextAction_Qualified(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.SuggestNextAction(context.Background(), SuggestInput{
		ThreadID: "t1", Stage: "qualified",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Action, "proposal")
}

func TestGrokProvider_SuggestNextAction_Proposal(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.SuggestNextAction(context.Background(), SuggestInput{
		ThreadID: "t1", Stage: "proposal",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Action, "review meeting")
}

func TestGrokProvider_SuggestNextAction_Negotiation(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.SuggestNextAction(context.Background(), SuggestInput{
		ThreadID: "t1", Stage: "negotiation",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Action, "objections")
}

func TestGrokProvider_SuggestNextAction_DefaultStage(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.SuggestNextAction(context.Background(), SuggestInput{
		ThreadID: "t1", Stage: "unknown_stage",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Action, "Follow up")
}

func TestGrokProvider_SuggestNextAction_EmptyThreadID(t *testing.T) {
	p := NewGrokProvider()
	_, err := p.SuggestNextAction(context.Background(), SuggestInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread_id is required")
}

// --- Handler Tests ---

func TestHandler_Enrich_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Big Deal", Slug: "big-deal", AuthorID: "u1", Metadata: `{"stage":"qualified"}`}
	require.NoError(t, db.Create(thread).Error)

	mp := &mockProvider{}
	bus := event.NewBus()
	h := NewHandler(mp, db, bus)

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/enrich", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Enrich(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result EnrichResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, thread.ID, result.ThreadID)
	assert.NotNil(t, result.Summary)
	assert.NotNil(t, result.Suggestion)

	// Verify metadata updated.
	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	assert.Contains(t, updated.Metadata, "llm_summary")
	assert.Contains(t, updated.Metadata, "llm_next_action")
}

func TestHandler_Enrich_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	mp := &mockProvider{}
	h := NewHandler(mp, db, event.NewBus())

	req := httptest.NewRequest("POST", "/threads/nonexistent/enrich", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Enrich(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Enrich_EmptyThreadParam(t *testing.T) {
	db := setupTestDB(t)
	mp := &mockProvider{}
	h := NewHandler(mp, db, event.NewBus())

	req := httptest.NewRequest("POST", "/threads//enrich", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Enrich(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Enrich_ProviderErrors(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead-err", AuthorID: "u1", Metadata: `{"stage":"new_lead"}`}
	require.NoError(t, db.Create(thread).Error)

	mp := &mockProvider{
		summarizeErr: fmt.Errorf("summarize failed"),
		suggestErr:   fmt.Errorf("suggest failed"),
	}
	h := NewHandler(mp, db, event.NewBus())

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/enrich", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Enrich(w, req)
	// Should still succeed with partial results.
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Enrich_PublishesEvent(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead-evt", AuthorID: "u1", Metadata: `{}`}
	require.NoError(t, db.Create(thread).Error)

	mp := &mockProvider{}
	bus := event.NewBus()
	received := make(chan event.Event, 1)
	bus.Subscribe(event.LeadEnriched, func(e event.Event) {
		received <- e
	})

	h := NewHandler(mp, db, bus)

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/enrich", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Enrich(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	evt := <-received
	assert.Equal(t, event.LeadEnriched, evt.Type)
	assert.Equal(t, thread.ID, evt.EntityID)
}

func TestHandler_Enrich_NilEventBus(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead-nil-bus", AuthorID: "u1", Metadata: `{}`}
	require.NoError(t, db.Create(thread).Error)

	mp := &mockProvider{}
	h := NewHandler(mp, db, nil)

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/enrich", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Enrich(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- extractStageFromMeta Tests ---

func TestExtractStageFromMeta_Empty(t *testing.T) {
	assert.Equal(t, "", extractStageFromMeta(""))
	assert.Equal(t, "", extractStageFromMeta("{}"))
}

func TestExtractStageFromMeta_HasStage(t *testing.T) {
	assert.Equal(t, "qualified", extractStageFromMeta(`{"stage":"qualified"}`))
}

func TestExtractStageFromMeta_NoStage(t *testing.T) {
	assert.Equal(t, "", extractStageFromMeta(`{"other":"field"}`))
}

func TestExtractStageFromMeta_InvalidJSON(t *testing.T) {
	assert.Equal(t, "", extractStageFromMeta("not-json"))
}

func TestExtractStageFromMeta_NonStringStage(t *testing.T) {
	assert.Equal(t, "", extractStageFromMeta(`{"stage":123}`))
}

// --- Integration: enrichThread via mock ---

func TestEnrichThread_StoresResults(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Deal", Slug: "deal-store", AuthorID: "u1", Metadata: `{"stage":"proposal"}`}
	require.NoError(t, db.Create(thread).Error)

	mp := &mockProvider{
		summarizeResp: &Summary{Text: "Custom summary"},
		suggestResp:   &Suggestion{Action: "Schedule call", Reasoning: "Time sensitive"},
	}
	h := NewHandler(mp, db, event.NewBus())

	result, err := h.enrichThread(context.Background(), thread.ID)
	require.NoError(t, err)
	assert.Equal(t, "Custom summary", result.Summary.Text)
	assert.Equal(t, "Schedule call", result.Suggestion.Action)

	// Verify DB updated.
	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated.Metadata), &meta))
	assert.Equal(t, "Custom summary", meta["llm_summary"])
	assert.Equal(t, "Schedule call", meta["llm_next_action"])
}

func TestEnrichThread_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	mp := &mockProvider{}
	h := NewHandler(mp, db, event.NewBus())

	_, err := h.enrichThread(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread not found")
}
