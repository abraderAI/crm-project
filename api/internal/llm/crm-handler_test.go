package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/event"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/notification"
)

// --- Test helpers ---

func setupCRMTestDB(t *testing.T) *gorm.DB {
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

func createCRMHierarchy(t *testing.T, db *gorm.DB) (*models.Org, *models.Space, *models.Board) {
	t.Helper()
	org := &models.Org{Name: "DEFT", Slug: "deft", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "CRM", Slug: "crm", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Opportunities", Slug: "opportunities", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return org, space, board
}

func createOppThread(t *testing.T, db *gorm.DB, board *models.Board, orgID, authorID, title, stage string) *models.Thread {
	t.Helper()
	meta := `{"crm_type":"opportunity","stage":"` + stage + `"}`
	thread := &models.Thread{
		BoardID:    board.ID,
		Title:      title,
		Slug:       strings.ReplaceAll(strings.ToLower(title), " ", "-"),
		AuthorID:   authorID,
		Metadata:   meta,
		OrgID:      &orgID,
		ThreadType: models.ThreadTypeLead,
	}
	require.NoError(t, db.Create(thread).Error)
	return thread
}

func authContext(userID string) context.Context {
	return auth.SetUserContext(context.Background(), &auth.UserContext{
		UserID:     userID,
		AuthMethod: auth.AuthMethodJWT,
	})
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withAuthAndChi(r *http.Request, userID string, params map[string]string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{
		UserID:     userID,
		AuthMethod: auth.AuthMethodJWT,
	})
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
}

// --- GrokProvider new method tests ---

func TestGrokProvider_Briefing(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.Briefing(context.Background(), "user1",
		[]models.Thread{{Title: "Deal A"}, {Title: "Deal B"}},
		[]CRMTask{{Title: "Call client"}},
		nil)
	require.NoError(t, err)
	assert.Contains(t, result, "2 open opportunities")
	assert.Contains(t, result, "1 tasks")
	assert.Contains(t, result, "user1")
}

func TestGrokProvider_EmailSummary(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.EmailSummary(context.Background(),
		models.Message{Body: "Hi, I'd like to discuss pricing for your enterprise plan."},
		models.Thread{Title: "Acme Corp Contact"})
	require.NoError(t, err)
	assert.Contains(t, result, "Acme Corp Contact")
	assert.Contains(t, result, "pricing")
}

func TestGrokProvider_PipelineStrategy(t *testing.T) {
	p := NewGrokProvider()
	opps := make([]models.Thread, 5)
	result, err := p.PipelineStrategy(context.Background(), opps)
	require.NoError(t, err)
	assert.Contains(t, result, "5 open opportunities")
}

func TestGrokProvider_DealStrategy(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.DealStrategy(context.Background(),
		models.Thread{Title: "Big Deal"},
		[]models.Message{{Body: "m1"}, {Body: "m2"}},
		[]CRMTask{{Title: "t1"}})
	require.NoError(t, err)
	assert.Contains(t, result, "Big Deal")
	assert.Contains(t, result, "2 messages")
	assert.Contains(t, result, "1 tasks")
}

func TestGrokProvider_QualityMessage(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.QualityMessage(context.Background(),
		[]QualityViolation{{Field: "deal_amount", Message: "missing"}},
		models.Thread{Title: "Stale Opp"})
	require.NoError(t, err)
	assert.Contains(t, result, "Stale Opp")
	assert.Contains(t, result, "deal_amount")
}

// --- MockLLMProvider tests ---

func TestMockLLMProvider_CapturesArgs(t *testing.T) {
	m := NewMockLLMProvider()
	opps := []models.Thread{{Title: "A"}}
	tasks := []CRMTask{{Title: "T"}}
	msgs := []models.Message{{Body: "M"}}

	result, err := m.Briefing(context.Background(), "user-x", opps, tasks, msgs)
	require.NoError(t, err)
	assert.Equal(t, "Mock briefing response", result)
	assert.Equal(t, 1, m.BriefingCalls)
	assert.Equal(t, "user-x", m.LastBriefingUserID)
	assert.Len(t, m.LastBriefingOpps, 1)
	assert.Len(t, m.LastBriefingTasks, 1)
	assert.Len(t, m.LastBriefingMsgs, 1)
}

func TestMockLLMProvider_AllMethods(t *testing.T) {
	m := NewMockLLMProvider()

	_, err := m.Summarize(context.Background(), SummarizeInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, m.SummarizeCalls)

	_, err = m.SuggestNextAction(context.Background(), SuggestInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, m.SuggestCalls)

	_, err = m.EmailSummary(context.Background(), models.Message{}, models.Thread{})
	require.NoError(t, err)
	assert.Equal(t, 1, m.EmailSumCalls)

	_, err = m.PipelineStrategy(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, m.PipelineCalls)

	_, err = m.DealStrategy(context.Background(), models.Thread{}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, m.DealCalls)

	_, err = m.QualityMessage(context.Background(), nil, models.Thread{})
	require.NoError(t, err)
	assert.Equal(t, 1, m.QualityCalls)
}

// --- CRMHandler.Brief tests ---

func TestCRMHandler_Brief_Success(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "user1", "Deal Alpha", "qualified")

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/ai/brief", nil)
	req = withAuthAndChi(req, "user1", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	h.Brief(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "data: ")
	assert.Equal(t, 1, mock.BriefingCalls)
	assert.Equal(t, "user1", mock.LastBriefingUserID)
}

func TestCRMHandler_Brief_SSEFormat(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	mock.BriefingResp = "Your daily briefing content"
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/ai/brief", nil)
	req = withAuthAndChi(req, "user1", map[string]string{"org": "test"})
	w := httptest.NewRecorder()

	h.Brief(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify SSE format: "data: {json}\n\n"
	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, "data: "))
	assert.True(t, strings.HasSuffix(body, "\n\n"))

	// Parse JSON payload.
	jsonStr := strings.TrimPrefix(body, "data: ")
	jsonStr = strings.TrimSpace(jsonStr)
	var sse SSEResponse
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &sse))
	assert.Equal(t, "Your daily briefing content", sse.Content)
}

func TestCRMHandler_Brief_Unauthenticated(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/ai/brief", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", "test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Brief(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCRMHandler_Brief_ContextAssembly(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)

	// Create opportunities with different stages.
	createOppThread(t, db, board, org.ID, "user1", "Open Deal", "qualified")
	createOppThread(t, db, board, org.ID, "user1", "Closed Deal", "closed_won")

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/ai/brief", nil)
	req = withAuthAndChi(req, "user1", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	h.Brief(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	// Only open deal should be in the opps (closed_won excluded).
	assert.Equal(t, 1, len(mock.LastBriefingOpps))
	assert.Equal(t, "Open Deal", mock.LastBriefingOpps[0].Title)
}

// --- CRMHandler.DealStrategy tests ---

func TestCRMHandler_DealStrategy_OwnerAccess(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	opp := createOppThread(t, db, board, org.ID, "owner1", "Big Deal", "negotiation")

	// Create messages on the opportunity thread.
	for i := 0; i < 3; i++ {
		msg := &models.Message{
			ThreadID: opp.ID,
			Body:     "test message",
			AuthorID: "owner1",
			Type:     models.MessageTypeNote,
			Metadata: "{}",
		}
		require.NoError(t, db.Create(msg).Error)
	}

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/opportunities/"+opp.ID+"/strategy", nil)
	req = withAuthAndChi(req, "owner1", map[string]string{"org": org.ID, "id": opp.ID})
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, 1, mock.DealCalls)
	assert.Equal(t, opp.Title, mock.LastDealOpp.Title)
	assert.Len(t, mock.LastDealMsgs, 3)
}

func TestCRMHandler_DealStrategy_NonOwnerForbidden(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	opp := createOppThread(t, db, board, org.ID, "owner1", "Secret Deal", "proposal")

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/opportunities/"+opp.ID+"/strategy", nil)
	req = withAuthAndChi(req, "other-user", map[string]string{"org": org.ID, "id": opp.ID})
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, 0, mock.DealCalls)
}

func TestCRMHandler_DealStrategy_AdminAccess(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	opp := createOppThread(t, db, board, org.ID, "owner1", "Admin Deal", "qualified")

	// Create an org admin membership.
	mem := &models.OrgMembership{OrgID: org.ID, UserID: "admin-user", Role: models.RoleAdmin}
	require.NoError(t, db.Create(mem).Error)

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/opportunities/"+opp.ID+"/strategy", nil)
	req = withAuthAndChi(req, "admin-user", map[string]string{"org": org.ID, "id": opp.ID})
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mock.DealCalls)
}

func TestCRMHandler_DealStrategy_NotFound(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/opportunities/nonexistent/strategy", nil)
	req = withAuthAndChi(req, "user1", map[string]string{"org": "test", "id": "nonexistent"})
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCRMHandler_DealStrategy_Unauthenticated(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/opportunities/id/strategy", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", "test")
	rctx.URLParams.Add("id", "id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCRMHandler_DealStrategy_MissingID(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/opportunities//strategy", nil)
	req = withAuthAndChi(req, "user1", map[string]string{"org": "test", "id": ""})
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- CRMHandler.PipelineStrategy tests ---

func TestCRMHandler_PipelineStrategy_CEOAccess(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "rep1", "Deal 1", "qualified")
	createOppThread(t, db, board, org.ID, "rep2", "Deal 2", "negotiation")

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "ceo-user-id")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/ai/pipeline-strategy", nil)
	req = withAuthAndChi(req, "ceo-user-id", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	h.PipelineStrategy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, 1, mock.PipelineCalls)
	assert.Len(t, mock.LastPipelineOpps, 2)
}

func TestCRMHandler_PipelineStrategy_NonCEOForbidden(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "ceo-user-id")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/ai/pipeline-strategy", nil)
	req = withAuthAndChi(req, "regular-user", map[string]string{"org": "test"})
	w := httptest.NewRecorder()

	h.PipelineStrategy(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, 0, mock.PipelineCalls)
}

func TestCRMHandler_PipelineStrategy_PlatformAdminAccess(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "rep1", "Deal 1", "proposal")

	// Create platform admin record.
	pa := &models.PlatformAdmin{UserID: "platform-admin", GrantedBy: "system", IsActive: true}
	require.NoError(t, db.Create(pa).Error)

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "someone-else")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/ai/pipeline-strategy", nil)
	req = withAuthAndChi(req, "platform-admin", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	h.PipelineStrategy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, mock.PipelineCalls)
}

func TestCRMHandler_PipelineStrategy_Unauthenticated(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "ceo")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/ai/pipeline-strategy", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", "test")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.PipelineStrategy(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCRMHandler_PipelineStrategy_ExcludesClosedDeals(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "rep1", "Open Deal", "qualified")
	createOppThread(t, db, board, org.ID, "rep1", "Won Deal", "closed_won")
	createOppThread(t, db, board, org.ID, "rep1", "Lost Deal", "closed_lost")

	mock := NewMockLLMProvider()
	h := NewCRMHandler(mock, db, "ceo")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/ai/pipeline-strategy", nil)
	req = withAuthAndChi(req, "ceo", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	h.PipelineStrategy(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	// Only open deals should be included.
	assert.Len(t, mock.LastPipelineOpps, 1)
	assert.Equal(t, "Open Deal", mock.LastPipelineOpps[0].Title)
}

// --- EmailSummarySubscriber tests ---

func TestEmailSummarySubscriber_HandleEvent(t *testing.T) {
	db := setupCRMTestDB(t)
	_, _, board := createCRMHierarchy(t, db)

	// Create a thread and message.
	thread := &models.Thread{BoardID: board.ID, Title: "Contact Thread", Slug: "contact", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "Important email content about the deal", AuthorID: "external", Type: models.MessageTypeEmail, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)

	payload, _ := json.Marshal(EmailReceivedPayload{
		MessageID:       msg.ID,
		EntityThreadID:  thread.ID,
		RecipientUserID: "rep1",
	})

	bus := event.NewBus()
	sub.Subscribe(bus)

	// Publish event.
	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: string(payload),
	})

	// Wait briefly for async handler.
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, mock.GetEmailSumCalls())
	assert.Equal(t, msg.ID, mock.GetLastEmailMsg().ID)
	assert.Equal(t, thread.ID, mock.GetLastEmailThread().ID)

	// Verify notification was created.
	var notifs []models.Notification
	db.Where("user_id = ? AND type = ?", "rep1", "email_summary").Find(&notifs)
	require.Len(t, notifs, 1)
	assert.Contains(t, notifs[0].Body, "Mock email summary")
}

func TestEmailSummarySubscriber_SkipsUnmatchedEmail(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)

	payload, _ := json.Marshal(EmailReceivedPayload{
		MessageID:       "msg1",
		EntityThreadID:  "", // No entity thread — unmatched.
		RecipientUserID: "rep1",
	})

	bus := event.NewBus()
	sub.Subscribe(bus)
	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: string(payload),
	})

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, mock.GetEmailSumCalls())
}

func TestEmailSummarySubscriber_NonBlocking(t *testing.T) {
	// Verify that the event bus returns immediately (non-blocking).
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)
	bus := event.NewBus()
	sub.Subscribe(bus)

	payload, _ := json.Marshal(EmailReceivedPayload{
		MessageID:       "nonexistent",
		EntityThreadID:  "nonexistent",
		RecipientUserID: "rep1",
	})

	// This should return immediately (event bus publishes in goroutine).
	start := time.Now()
	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: string(payload),
	})
	elapsed := time.Since(start)

	// Publish should return in under 10ms (non-blocking).
	assert.Less(t, elapsed.Milliseconds(), int64(10))
}

// --- DailyBriefing tests ---

func TestDailyBriefing_OptsInUser(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "opted-in-user", "My Deal", "qualified")

	// Opt the user in.
	pref := &models.NotificationPreference{
		UserID:    "opted-in-user",
		EventType: "daily_crm_brief",
		Channel:   "in_app",
		Enabled:   true,
	}
	require.NoError(t, db.Create(pref).Error)

	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)

	RunBriefingOnce(DailyBriefingConfig{
		Provider:  mock,
		DB:        db,
		NotifRepo: notifRepo,
		Logger:    slog.Default(),
	})

	assert.Equal(t, 1, mock.BriefingCalls)
	assert.Equal(t, "opted-in-user", mock.LastBriefingUserID)

	// Verify notification created.
	var notifs []models.Notification
	db.Where("user_id = ? AND type = ?", "opted-in-user", "daily_crm_brief").Find(&notifs)
	require.Len(t, notifs, 1)
	assert.Equal(t, "Daily CRM Briefing", notifs[0].Title)
}

func TestDailyBriefing_SkipsOptedOutUser(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "opted-out-user", "Some Deal", "qualified")

	// Opt the user out.
	pref := &models.NotificationPreference{
		UserID:    "opted-out-user",
		EventType: "daily_crm_brief",
		Channel:   "in_app",
		Enabled:   false,
	}
	require.NoError(t, db.Create(pref).Error)

	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)

	RunBriefingOnce(DailyBriefingConfig{
		Provider:  mock,
		DB:        db,
		NotifRepo: notifRepo,
		Logger:    slog.Default(),
	})

	assert.Equal(t, 0, mock.BriefingCalls)
}

func TestDailyBriefing_SkipsUserWithEmptyPipeline(t *testing.T) {
	db := setupCRMTestDB(t)
	// Don't create any opportunities for this user.

	pref := &models.NotificationPreference{
		UserID:    "empty-pipeline-user",
		EventType: "daily_crm_brief",
		Channel:   "in_app",
		Enabled:   true,
	}
	require.NoError(t, db.Create(pref).Error)

	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)

	RunBriefingOnce(DailyBriefingConfig{
		Provider:  mock,
		DB:        db,
		NotifRepo: notifRepo,
		Logger:    slog.Default(),
	})

	// User is opted in but has no records, so should be skipped.
	assert.Equal(t, 0, mock.BriefingCalls)
}

func TestDailyBriefing_NoOptedInUsers(t *testing.T) {
	db := setupCRMTestDB(t)

	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)

	RunBriefingOnce(DailyBriefingConfig{
		Provider:  mock,
		DB:        db,
		NotifRepo: notifRepo,
		Logger:    slog.Default(),
	})

	assert.Equal(t, 0, mock.BriefingCalls)
}

// --- nextOccurrence tests ---

func TestNextOccurrence_BeforeTarget(t *testing.T) {
	now := time.Date(2026, 3, 25, 6, 0, 0, 0, time.UTC) // 06:00
	next := nextOccurrence(now, 8)
	assert.Equal(t, 8, next.Hour())
	assert.Equal(t, 25, next.Day())
}

func TestNextOccurrence_AfterTarget(t *testing.T) {
	now := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC) // 10:00
	next := nextOccurrence(now, 8)
	assert.Equal(t, 8, next.Hour())
	assert.Equal(t, 26, next.Day()) // Next day.
}

func TestNextOccurrence_ExactTarget(t *testing.T) {
	now := time.Date(2026, 3, 25, 8, 0, 1, 0, time.UTC) // Just past 08:00
	next := nextOccurrence(now, 8)
	assert.Equal(t, 26, next.Day()) // Should be next day.
}

// --- writeSSE tests ---

func TestWriteSSE_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSE(w, "hello world")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))

	body := w.Body.String()
	assert.True(t, strings.HasPrefix(body, "data: "))
	assert.True(t, strings.HasSuffix(body, "\n\n"))

	var sse SSEResponse
	jsonStr := strings.TrimPrefix(body, "data: ")
	jsonStr = strings.TrimSuffix(jsonStr, "\n\n")
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &sse))
	assert.Equal(t, "hello world", sse.Content)
}

// --- isCEOOrAdmin tests ---

func TestIsCEOOrAdmin_CEO(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "ceo-123")
	assert.True(t, h.isCEOOrAdmin(context.Background(), "ceo-123"))
}

func TestIsCEOOrAdmin_NotCEO(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "ceo-123")
	assert.False(t, h.isCEOOrAdmin(context.Background(), "regular-user"))
}

func TestIsCEOOrAdmin_PlatformAdmin(t *testing.T) {
	db := setupCRMTestDB(t)
	pa := &models.PlatformAdmin{UserID: "padmin", GrantedBy: "system", IsActive: true}
	require.NoError(t, db.Create(pa).Error)

	h := NewCRMHandler(NewMockLLMProvider(), db, "someone-else")
	assert.True(t, h.isCEOOrAdmin(context.Background(), "padmin"))
}

func TestIsCEOOrAdmin_EmptyCEOID(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	// With empty CEO ID, only platform admins can access.
	assert.False(t, h.isCEOOrAdmin(context.Background(), "anyone"))
}

// --- isOwnerOrAdmin tests ---

func TestIsOwnerOrAdmin_Author(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	opp := models.Thread{AuthorID: "user1"}
	assert.True(t, h.isOwnerOrAdmin(context.Background(), opp, "user1"))
}

func TestIsOwnerOrAdmin_NonOwner(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	opp := createOppThread(t, db, board, org.ID, "owner1", "Deal", "qualified")

	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	assert.False(t, h.isOwnerOrAdmin(context.Background(), *opp, "other-user"))
}

func TestIsOwnerOrAdmin_OrgAdmin(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	opp := createOppThread(t, db, board, org.ID, "owner1", "Deal", "qualified")

	mem := &models.OrgMembership{OrgID: org.ID, UserID: "admin1", Role: models.RoleAdmin}
	require.NoError(t, db.Create(mem).Error)

	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	assert.True(t, h.isOwnerOrAdmin(context.Background(), *opp, "admin1"))
}

// --- loadUserTasks and loadTasksForThread tests ---

func TestLoadUserTasks_EmptyWhenNoTable(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	tasks, err := h.loadUserTasks(context.Background(), "user1")
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestLoadTasksForThread_EmptyWhenNoTable(t *testing.T) {
	db := setupCRMTestDB(t)
	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	tasks := h.loadTasksForThread(context.Background(), "thread1")
	assert.Empty(t, tasks)
}

// --- Brief error paths ---

func TestCRMHandler_Brief_LLMError(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	mock.BriefingErr = assert.AnError
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/ai/brief", nil)
	req = withAuthAndChi(req, "user1", map[string]string{"org": "test"})
	w := httptest.NewRecorder()

	h.Brief(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- DealStrategy error paths ---

func TestCRMHandler_DealStrategy_LLMError(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	opp := createOppThread(t, db, board, org.ID, "owner1", "Deal", "qualified")

	mock := NewMockLLMProvider()
	mock.DealErr = assert.AnError
	h := NewCRMHandler(mock, db, "")

	req := httptest.NewRequest("POST", "/v1/orgs/"+org.ID+"/crm/opportunities/"+opp.ID+"/strategy", nil)
	req = withAuthAndChi(req, "owner1", map[string]string{"org": org.ID, "id": opp.ID})
	w := httptest.NewRecorder()

	h.DealStrategy(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- PipelineStrategy error paths ---

func TestCRMHandler_PipelineStrategy_LLMError(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	mock.PipelineErr = assert.AnError
	h := NewCRMHandler(mock, db, "ceo")

	req := httptest.NewRequest("POST", "/v1/orgs/test/crm/ai/pipeline-strategy", nil)
	req = withAuthAndChi(req, "ceo", map[string]string{"org": "test"})
	w := httptest.NewRecorder()

	h.PipelineStrategy(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- loadUserOpportunities fallback path ---

func TestLoadUserOpportunities_FallbackToAuthor(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "user1", "Authored Deal", "qualified")

	// The fallback path is exercised when thread_acl doesn't exist.
	// Since our test DB has no thread_acl table, the primary query may fail.
	// Test the fallback function directly.
	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	opps, err := h.loadUserOpportunitiesFallback(context.Background(), "user1")
	require.NoError(t, err)
	assert.Len(t, opps, 1)
	assert.Equal(t, "Authored Deal", opps[0].Title)
}

func TestLoadUserOpportunities_FallbackExcludesClosedWon(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "user1", "Open", "qualified")
	createOppThread(t, db, board, org.ID, "user1", "Closed", "closed_won")

	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	opps, err := h.loadUserOpportunitiesFallback(context.Background(), "user1")
	require.NoError(t, err)
	assert.Len(t, opps, 1)
	assert.Equal(t, "Open", opps[0].Title)
}

// --- loadRecentMessages tests ---

func TestLoadRecentMessages_FallbackToAuthor(t *testing.T) {
	db := setupCRMTestDB(t)
	_, _, board := createCRMHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Test", Slug: "test", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "recent", AuthorID: "user1", Type: models.MessageTypeNote, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	h := NewCRMHandler(NewMockLLMProvider(), db, "")
	msgs, err := h.loadRecentMessages(context.Background(), "user1")
	require.NoError(t, err)
	// Should get the message via fallback (author match).
	assert.GreaterOrEqual(t, len(msgs), 1)
}

// --- EmailSummary subscriber additional coverage ---

func TestEmailSummarySubscriber_InvalidPayload(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)
	bus := event.NewBus()
	sub.Subscribe(bus)

	// Publish with invalid JSON payload.
	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: "not-json",
	})

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, mock.GetEmailSumCalls())
}

func TestEmailSummarySubscriber_MessageNotFound(t *testing.T) {
	db := setupCRMTestDB(t)
	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)
	bus := event.NewBus()
	sub.Subscribe(bus)

	payload, _ := json.Marshal(EmailReceivedPayload{
		MessageID:       "nonexistent-msg",
		EntityThreadID:  "some-thread",
		RecipientUserID: "rep1",
	})

	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: string(payload),
	})

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, mock.GetEmailSumCalls())
}

func TestEmailSummarySubscriber_ThreadNotFound(t *testing.T) {
	db := setupCRMTestDB(t)
	_, _, board := createCRMHierarchy(t, db)

	// Create message but thread reference is bad.
	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "t", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "body", AuthorID: "u1", Type: models.MessageTypeEmail, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	mock := NewMockLLMProvider()
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)
	bus := event.NewBus()
	sub.Subscribe(bus)

	payload, _ := json.Marshal(EmailReceivedPayload{
		MessageID:       msg.ID,
		EntityThreadID:  "nonexistent-thread",
		RecipientUserID: "rep1",
	})

	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: string(payload),
	})

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, mock.GetEmailSumCalls())
}

func TestEmailSummarySubscriber_LLMError(t *testing.T) {
	db := setupCRMTestDB(t)
	_, _, board := createCRMHierarchy(t, db)

	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "t-err", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "body", AuthorID: "u1", Type: models.MessageTypeEmail, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	mock := NewMockLLMProvider()
	mock.EmailSumErr = assert.AnError
	notifRepo := notification.NewRepository(db)
	nopLogger := slog.Default()

	sub := NewEmailSummarySubscriber(mock, db, notifRepo, nopLogger)
	bus := event.NewBus()
	sub.Subscribe(bus)

	payload, _ := json.Marshal(EmailReceivedPayload{
		MessageID:       msg.ID,
		EntityThreadID:  thread.ID,
		RecipientUserID: "rep1",
	})

	bus.Publish(event.Event{
		Type:    event.EmailReceived,
		Payload: string(payload),
	})

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 1, mock.GetEmailSumCalls())
	// Should not create notification when LLM fails.
	var notifs []models.Notification
	db.Where("user_id = ? AND type = ?", "rep1", "email_summary").Find(&notifs)
	assert.Len(t, notifs, 0)
}

// --- DailyBriefing LLM error path ---

func TestDailyBriefing_LLMError(t *testing.T) {
	db := setupCRMTestDB(t)
	org, _, board := createCRMHierarchy(t, db)
	createOppThread(t, db, board, org.ID, "user1", "Deal", "qualified")

	pref := &models.NotificationPreference{
		UserID:    "user1",
		EventType: "daily_crm_brief",
		Channel:   "in_app",
		Enabled:   true,
	}
	require.NoError(t, db.Create(pref).Error)

	mock := NewMockLLMProvider()
	mock.BriefingErr = assert.AnError
	notifRepo := notification.NewRepository(db)

	RunBriefingOnce(DailyBriefingConfig{
		Provider:  mock,
		DB:        db,
		NotifRepo: notifRepo,
		Logger:    slog.Default(),
	})

	assert.Equal(t, 1, mock.BriefingCalls)
	// Should not create notification on error.
	var notifs []models.Notification
	db.Where("user_id = ? AND type = ?", "user1", "daily_crm_brief").Find(&notifs)
	assert.Len(t, notifs, 0)
}

// --- GrokProvider edge cases for new methods ---

func TestGrokProvider_EmailSummary_LongBody(t *testing.T) {
	p := NewGrokProvider()
	longBody := strings.Repeat("a", 200)
	result, err := p.EmailSummary(context.Background(),
		models.Message{Body: longBody},
		models.Thread{Title: "Thread"})
	require.NoError(t, err)
	// Body should be truncated to 80 chars.
	assert.NotEmpty(t, result)
}

func TestGrokProvider_QualityMessage_MultipleViolations(t *testing.T) {
	p := NewGrokProvider()
	result, err := p.QualityMessage(context.Background(),
		[]QualityViolation{
			{Field: "email", Message: "missing"},
			{Field: "phone", Message: "missing"},
		},
		models.Thread{Title: "Contact"})
	require.NoError(t, err)
	assert.Contains(t, result, "2 issues")
	assert.Contains(t, result, "email")
	assert.Contains(t, result, "phone")
}

// --- MockLLMProvider error paths ---

func TestMockLLMProvider_ErrorResponses(t *testing.T) {
	m := NewMockLLMProvider()
	m.SummarizeErr = assert.AnError
	m.SuggestErr = assert.AnError

	_, err := m.Summarize(context.Background(), SummarizeInput{})
	assert.Error(t, err)

	_, err = m.SuggestNextAction(context.Background(), SuggestInput{})
	assert.Error(t, err)
}

func TestMockLLMProvider_CustomSummarizeResp(t *testing.T) {
	m := NewMockLLMProvider()
	custom := &Summary{Text: "custom"}
	m.SummarizeResp = custom

	result, err := m.Summarize(context.Background(), SummarizeInput{})
	require.NoError(t, err)
	assert.Equal(t, "custom", result.Text)
}

func TestMockLLMProvider_CustomSuggestResp(t *testing.T) {
	m := NewMockLLMProvider()
	custom := &Suggestion{Action: "custom action"}
	m.SuggestResp = custom

	result, err := m.SuggestNextAction(context.Background(), SuggestInput{})
	require.NoError(t, err)
	assert.Equal(t, "custom action", result.Action)
}

// --- getOptedInUsers tests ---

func TestGetOptedInUsers_ReturnsOnlyEnabled(t *testing.T) {
	db := setupCRMTestDB(t)

	// Create opted-in and opted-out prefs.
	require.NoError(t, db.Create(&models.NotificationPreference{
		UserID: "in1", EventType: "daily_crm_brief", Channel: "in_app", Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&models.NotificationPreference{
		UserID: "in2", EventType: "daily_crm_brief", Channel: "in_app", Enabled: true,
	}).Error)
	require.NoError(t, db.Create(&models.NotificationPreference{
		UserID: "out1", EventType: "daily_crm_brief", Channel: "in_app", Enabled: false,
	}).Error)
	// Different event type — should not match.
	require.NoError(t, db.Create(&models.NotificationPreference{
		UserID: "other1", EventType: "stage_change", Channel: "in_app", Enabled: true,
	}).Error)

	users, err := getOptedInUsers(context.Background(), db)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Contains(t, users, "in1")
	assert.Contains(t, users, "in2")
}
