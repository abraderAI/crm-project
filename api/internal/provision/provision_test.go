package provision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/billing"
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

func createClosedWonThread(t *testing.T, db *gorm.DB, boardID string) *models.Thread {
	t.Helper()
	thread := &models.Thread{
		BoardID:  boardID,
		Title:    "Acme Corp Deal",
		Slug:     "acme-corp-deal",
		AuthorID: "u1",
		Metadata: `{"stage":"closed_won","company":"Acme Corp","contact_email":"john@acme.com","deal_value":50000}`,
	}
	require.NoError(t, db.Create(thread).Error)
	return thread
}

// mockBillingProvider implements billing.BillingProvider for testing.
type mockBillingProvider struct {
	createCustomerErr    error
	createCustomerResult *billing.Customer
	createCustomerCalls  int
}

func (m *mockBillingProvider) CreateCustomer(_ context.Context, input billing.CreateCustomerInput) (*billing.Customer, error) {
	m.createCustomerCalls++
	if m.createCustomerErr != nil {
		return nil, m.createCustomerErr
	}
	if m.createCustomerResult != nil {
		return m.createCustomerResult, nil
	}
	return &billing.Customer{
		ID:         "cust-1",
		OrgID:      input.OrgID,
		ExternalID: "fp_cust_123",
		Name:       input.Name,
		Email:      input.Email,
	}, nil
}

func (m *mockBillingProvider) CreateInvoice(_ context.Context, _ billing.CreateInvoiceInput) (*billing.Invoice, error) {
	return nil, nil
}

func (m *mockBillingProvider) GetPaymentStatus(_ context.Context, _ string) (*billing.PaymentStatus, error) {
	return nil, nil
}

func (m *mockBillingProvider) HandleWebhook(_ context.Context, _ billing.WebhookPayload) (*billing.WebhookResult, error) {
	return nil, nil
}

// --- Service Tests ---

func TestService_ProvisionCustomer_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	bp := &mockBillingProvider{}
	svc := NewService(db, bp, event.NewBus())

	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)
	assert.NotEmpty(t, result.CustomerOrgID)
	assert.NotEmpty(t, result.CustomerOrgSlug)
	assert.Len(t, result.SpacesCreated, 3)
	assert.Len(t, result.BoardsCreated, 6)
	assert.NotEmpty(t, result.BillingCustomer)
	assert.Equal(t, thread.ID, result.CRMThreadID)
	assert.Contains(t, result.Message, "Acme Corp")
}

func TestService_ProvisionCustomer_VerifyOrg(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)

	var org models.Org
	require.NoError(t, db.First(&org, "id = ?", result.CustomerOrgID).Error)
	assert.Equal(t, "Acme Corp", org.Name)
	assert.Contains(t, org.Description, "provisioned from lead")
}

func TestService_ProvisionCustomer_VerifySpaces(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)

	var spaces []models.Space
	require.NoError(t, db.Where("org_id = ?", result.CustomerOrgID).Find(&spaces).Error)
	assert.Len(t, spaces, 3)
}

func TestService_ProvisionCustomer_VerifyBoards(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)

	for _, spaceID := range result.SpacesCreated {
		var boards []models.Board
		require.NoError(t, db.Where("space_id = ?", spaceID).Find(&boards).Error)
		assert.Len(t, boards, 2, "each space should have 2 boards")
	}
}

func TestService_ProvisionCustomer_VerifyThreadUpdated(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)

	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated.Metadata), &meta))
	assert.Equal(t, result.CustomerOrgID, meta["customer_org_id"])
	assert.NotEmpty(t, meta["provisioned_at"])
}

func TestService_ProvisionCustomer_VerifyConfirmMessage(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	_, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)

	var msgs []models.Message
	require.NoError(t, db.Where("thread_id = ?", thread.ID).Find(&msgs).Error)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Body, "Acme Corp")
	assert.Equal(t, models.MessageTypeSystem, msgs[0].Type)
}

func TestService_ProvisionCustomer_CustomInput(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{
		CompanyName:  "Override Corp",
		ContactEmail: "override@test.com",
	})
	require.NoError(t, err)
	assert.Contains(t, result.Message, "Override Corp")
}

func TestService_ProvisionCustomer_NoBillingProvider(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, nil, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)
	assert.Empty(t, result.BillingCustomer)
	assert.NotEmpty(t, result.CustomerOrgID)
}

func TestService_ProvisionCustomer_EmptyThreadID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	_, err := svc.ProvisionCustomer(context.Background(), "", "u1", ProvisionInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread_id is required")
}

func TestService_ProvisionCustomer_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	_, err := svc.ProvisionCustomer(context.Background(), "nonexistent", "u1", ProvisionInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread not found")
}

func TestService_ProvisionCustomer_NotClosedWon(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "not-won", AuthorID: "u1", Metadata: `{"stage":"proposal"}`}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, nil, event.NewBus())
	_, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed_won")
}

func TestService_ProvisionCustomer_AlreadyProvisioned(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{
		BoardID: board.ID, Title: "Lead", Slug: "already-prov", AuthorID: "u1",
		Metadata: `{"stage":"closed_won","customer_org_id":"existing-id"}`,
	}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, nil, event.NewBus())
	_, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already provisioned")
}

func TestService_ProvisionCustomer_PublishesEvent(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	bus := event.NewBus()
	received := make(chan event.Event, 1)
	bus.Subscribe(event.CustomerProvisioned, func(e event.Event) {
		received <- e
	})

	svc := NewService(db, &mockBillingProvider{}, bus)
	_, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)

	evt := <-received
	assert.Equal(t, event.CustomerProvisioned, evt.Type)
	assert.Equal(t, thread.ID, evt.EntityID)
}

func TestService_ProvisionCustomer_NilEventBus(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, nil, nil)
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)
	assert.NotEmpty(t, result.CustomerOrgID)
}

func TestService_ProvisionCustomer_FallbackToTitle(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{
		BoardID: board.ID, Title: "Custom Title Lead", Slug: "title-fallback", AuthorID: "u1",
		Metadata: `{"stage":"closed_won"}`,
	}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, nil, event.NewBus())
	result, err := svc.ProvisionCustomer(context.Background(), thread.ID, "u1", ProvisionInput{})
	require.NoError(t, err)
	assert.Contains(t, result.Message, "Custom Title Lead")
}

func TestService_HandleStageChanged_ClosedWon(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())

	payload := `{"new_stage":"closed_won"}`
	svc.HandleStageChanged(event.Event{EntityType: "thread", EntityID: thread.ID, UserID: "u1", Payload: payload})

	// Verify provisioning occurred.
	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	assert.Contains(t, updated.Metadata, "customer_org_id")
}

func TestService_HandleStageChanged_NotClosedWon(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	// Should not panic.
	svc.HandleStageChanged(event.Event{EntityType: "thread", EntityID: "some-id", Payload: `{"new_stage":"proposal"}`})
}

func TestService_HandleStageChanged_NonThread(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	svc.HandleStageChanged(event.Event{EntityType: "org", EntityID: "some-id"})
}

func TestService_HandleStageChanged_EmptyEntityID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	svc.HandleStageChanged(event.Event{EntityType: "thread", EntityID: ""})
}

func TestService_HandleStageChanged_InvalidPayload(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	svc.HandleStageChanged(event.Event{EntityType: "thread", EntityID: "x", Payload: "not-json"})
}

// --- Handler Tests ---

func TestHandler_Provision_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	h := NewHandler(svc)

	body := `{"company_name":"Test Corp","contact_email":"test@test.com"}`
	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/provision", strings.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = auth.SetUserContext(ctx, &auth.UserContext{UserID: "u1"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Provision(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var result ProvisionResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.NotEmpty(t, result.CustomerOrgID)
}

func TestHandler_Provision_EmptyBody(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/provision", strings.NewReader(""))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = auth.SetUserContext(ctx, &auth.UserContext{UserID: "u1"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.Provision(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Provision_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/threads/nonexistent/provision", strings.NewReader("{}"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Provision(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Provision_EmptyThreadParam(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, nil, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/threads//provision", strings.NewReader("{}"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Provision(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Provision_NotClosedWon(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "not-won-h", AuthorID: "u1", Metadata: `{"stage":"qualified"}`}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, nil, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/provision", strings.NewReader("{}"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Provision(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Provision_NoAuthContext(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := createClosedWonThread(t, db, board.ID)

	svc := NewService(db, &mockBillingProvider{}, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/provision", strings.NewReader("{}"))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Provision(w, req)
	assert.Equal(t, http.StatusCreated, w.Code) // Should work with empty userID.
}

// --- parseMetadata Tests ---

func TestParseMetadata_Empty(t *testing.T) {
	m := parseMetadata("")
	assert.Empty(t, m)
}

func TestParseMetadata_EmptyObj(t *testing.T) {
	m := parseMetadata("{}")
	assert.Empty(t, m)
}

func TestParseMetadata_Valid(t *testing.T) {
	m := parseMetadata(`{"stage":"closed_won","company":"Acme"}`)
	assert.Equal(t, "closed_won", m["stage"])
	assert.Equal(t, "Acme", m["company"])
}

func TestParseMetadata_Invalid(t *testing.T) {
	m := parseMetadata("not-json")
	assert.Empty(t, m)
}
