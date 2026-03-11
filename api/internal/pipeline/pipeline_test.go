package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
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

// --- Test Helpers ---

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

func createTestHierarchy(t *testing.T, db *gorm.DB) (org *models.Org, space *models.Space, board *models.Board) {
	t.Helper()
	org = &models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space = &models.Space{OrgID: org.ID, Name: "CRM", Slug: "crm", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board = &models.Board{SpaceID: space.ID, Name: "Pipeline", Slug: "pipeline", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return
}

func createTestThread(t *testing.T, db *gorm.DB, boardID, title, meta string) *models.Thread {
	t.Helper()
	if meta == "" {
		meta = "{}"
	}
	thread := &models.Thread{BoardID: boardID, Title: title, Slug: "test-thread", AuthorID: "user-1", Metadata: meta}
	require.NoError(t, db.Create(thread).Error)
	return thread
}

// --- Config Tests ---

func TestDefaultStages(t *testing.T) {
	stages := DefaultStages()
	assert.Len(t, stages, 8)
	assert.Equal(t, StageNewLead, stages[0].Name)
	assert.Equal(t, StageClosedWon, stages[5].Name)
	assert.Equal(t, StageNurturing, stages[7].Name)
}

func TestDefaultStages_Order(t *testing.T) {
	stages := DefaultStages()
	for i, s := range stages {
		assert.Equal(t, i, s.Order, "stage %s should have order %d", s.Name, i)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Stages, 8)
}

func TestTransitionMap(t *testing.T) {
	tm := TransitionMap(DefaultStages())
	assert.Contains(t, tm[StageNewLead], StageContacted)
	assert.Contains(t, tm[StageNewLead], StageNurturing)
	assert.Contains(t, tm[StageNegotiation], StageClosedWon)
	assert.Empty(t, tm[StageClosedWon])
}

func TestValidateTransition_ValidForward(t *testing.T) {
	stages := DefaultStages()
	assert.NoError(t, ValidateTransition(stages, StageNewLead, StageContacted))
	assert.NoError(t, ValidateTransition(stages, StageContacted, StageQualified))
	assert.NoError(t, ValidateTransition(stages, StageQualified, StageProposal))
	assert.NoError(t, ValidateTransition(stages, StageProposal, StageNegotiation))
	assert.NoError(t, ValidateTransition(stages, StageNegotiation, StageClosedWon))
}

func TestValidateTransition_InvalidSkip(t *testing.T) {
	stages := DefaultStages()
	err := ValidateTransition(stages, StageNewLead, StageProposal)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestValidateTransition_FromEmpty_NewLead(t *testing.T) {
	stages := DefaultStages()
	assert.NoError(t, ValidateTransition(stages, "", StageNewLead))
}

func TestValidateTransition_FromEmpty_Nurturing(t *testing.T) {
	stages := DefaultStages()
	assert.NoError(t, ValidateTransition(stages, "", StageNurturing))
}

func TestValidateTransition_FromEmpty_Invalid(t *testing.T) {
	stages := DefaultStages()
	err := ValidateTransition(stages, "", StageProposal)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial stage")
}

func TestValidateTransition_ToClosedLost(t *testing.T) {
	stages := DefaultStages()
	for _, from := range []Stage{StageNewLead, StageContacted, StageQualified, StageProposal, StageNegotiation} {
		assert.NoError(t, ValidateTransition(stages, from, StageClosedLost))
	}
}

func TestValidateTransition_FromClosedLost_ToNurturing(t *testing.T) {
	stages := DefaultStages()
	assert.NoError(t, ValidateTransition(stages, StageClosedLost, StageNurturing))
}

func TestValidateTransition_UnknownCurrentStage(t *testing.T) {
	stages := DefaultStages()
	err := ValidateTransition(stages, Stage("unknown"), StageContacted)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown current stage")
}

func TestValidateTransition_NurturingPaths(t *testing.T) {
	stages := DefaultStages()
	assert.NoError(t, ValidateTransition(stages, StageNurturing, StageContacted))
	assert.NoError(t, ValidateTransition(stages, StageNurturing, StageQualified))
	assert.NoError(t, ValidateTransition(stages, StageNurturing, StageClosedLost))
}

func TestIsValidStage(t *testing.T) {
	stages := DefaultStages()
	assert.True(t, IsValidStage(stages, StageNewLead))
	assert.True(t, IsValidStage(stages, StageClosedWon))
	assert.False(t, IsValidStage(stages, Stage("nonexistent")))
	assert.False(t, IsValidStage(stages, Stage("")))
}

func TestIsFinalStage(t *testing.T) {
	stages := DefaultStages()
	assert.True(t, IsFinalStage(stages, StageClosedWon))
	assert.True(t, IsFinalStage(stages, StageClosedLost))
	assert.False(t, IsFinalStage(stages, StageNewLead))
	assert.False(t, IsFinalStage(stages, StageNurturing))
	assert.False(t, IsFinalStage(stages, Stage("unknown")))
}

func TestStageConstants(t *testing.T) {
	assert.Equal(t, Stage("new_lead"), StageNewLead)
	assert.Equal(t, Stage("contacted"), StageContacted)
	assert.Equal(t, Stage("qualified"), StageQualified)
	assert.Equal(t, Stage("proposal"), StageProposal)
	assert.Equal(t, Stage("negotiation"), StageNegotiation)
	assert.Equal(t, Stage("closed_won"), StageClosedWon)
	assert.Equal(t, Stage("closed_lost"), StageClosedLost)
	assert.Equal(t, Stage("nurturing"), StageNurturing)
}

func TestParseConfigFromMetadata_Empty(t *testing.T) {
	assert.Nil(t, ParseConfigFromMetadata(""))
	assert.Nil(t, ParseConfigFromMetadata("{}"))
}

func TestParseConfigFromMetadata_NoConfig(t *testing.T) {
	assert.Nil(t, ParseConfigFromMetadata(`{"other":"field"}`))
}

func TestParseConfigFromMetadata_InvalidJSON(t *testing.T) {
	assert.Nil(t, ParseConfigFromMetadata("not-json"))
}

func TestParseConfigFromMetadata_ValidConfig(t *testing.T) {
	meta := `{"pipeline_config":{"stages":[{"name":"lead","label":"Lead","order":0,"is_final":false,"transitions":["done"]},{"name":"done","label":"Done","order":1,"is_final":true,"transitions":[]}]}}`
	cfg := ParseConfigFromMetadata(meta)
	require.NotNil(t, cfg)
	assert.Len(t, cfg.Stages, 2)
	assert.Equal(t, Stage("lead"), cfg.Stages[0].Name)
}

func TestParseConfigFromMetadata_EmptyStages(t *testing.T) {
	meta := `{"pipeline_config":{"stages":[]}}`
	assert.Nil(t, ParseConfigFromMetadata(meta))
}

func TestParseConfigFromMetadata_InvalidConfigFormat(t *testing.T) {
	meta := `{"pipeline_config":"not-an-object"}`
	assert.Nil(t, ParseConfigFromMetadata(meta))
}

// --- Service Tests ---

func TestService_TransitionStage_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Test Lead", "{}")

	bus := event.NewBus()
	svc := NewService(db, bus)

	result, err := svc.TransitionStage(context.Background(), thread.ID, StageNewLead, "user-1")
	require.NoError(t, err)
	assert.Equal(t, thread.ID, result.ThreadID)
	assert.Equal(t, "", result.PreviousStage)
	assert.Equal(t, "new_lead", result.NewStage)

	// Verify metadata updated in DB.
	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	assert.Contains(t, updated.Metadata, `"stage":"new_lead"`)
}

func TestService_TransitionStage_ForwardProgression(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", `{"stage":"new_lead"}`)

	svc := NewService(db, event.NewBus())

	result, err := svc.TransitionStage(context.Background(), thread.ID, StageContacted, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "new_lead", result.PreviousStage)
	assert.Equal(t, "contacted", result.NewStage)
}

func TestService_TransitionStage_InvalidTransition(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", `{"stage":"new_lead"}`)

	svc := NewService(db, event.NewBus())

	_, err := svc.TransitionStage(context.Background(), thread.ID, StageClosedWon, "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestService_TransitionStage_FinalStage(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", `{"stage":"closed_won"}`)

	svc := NewService(db, event.NewBus())

	_, err := svc.TransitionStage(context.Background(), thread.ID, StageNurturing, "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestService_TransitionStage_InvalidStage(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", "{}")

	svc := NewService(db, event.NewBus())

	_, err := svc.TransitionStage(context.Background(), thread.ID, Stage("invalid"), "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid stage")
}

func TestService_TransitionStage_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())

	_, err := svc.TransitionStage(context.Background(), "nonexistent", StageNewLead, "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread not found")
}

func TestService_TransitionStage_EmptyThreadID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())

	_, err := svc.TransitionStage(context.Background(), "", StageNewLead, "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread_id is required")
}

func TestService_TransitionStage_EmptyStage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())

	_, err := svc.TransitionStage(context.Background(), "some-id", "", "user-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stage is required")
}

func TestService_TransitionStage_PublishesEvent(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", "{}")

	bus := event.NewBus()
	received := make(chan event.Event, 1)
	bus.Subscribe(event.PipelineStageChanged, func(e event.Event) {
		received <- e
	})

	svc := NewService(db, bus)
	_, err := svc.TransitionStage(context.Background(), thread.ID, StageNewLead, "user-1")
	require.NoError(t, err)

	evt := <-received
	assert.Equal(t, event.PipelineStageChanged, evt.Type)
	assert.Equal(t, thread.ID, evt.EntityID)
}

func TestService_TransitionStage_NilEventBus(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", "{}")

	svc := NewService(db, nil)
	result, err := svc.TransitionStage(context.Background(), thread.ID, StageNewLead, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "new_lead", result.NewStage)
}

func TestService_GetStages_Default(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())

	stages := svc.GetStages(context.Background(), "")
	assert.Len(t, stages, 8)
}

func TestService_GetStages_CustomFromOrg(t *testing.T) {
	db := setupTestDB(t)
	customCfg := `{"pipeline_config":{"stages":[{"name":"start","label":"Start","order":0,"is_final":false,"transitions":["end"]},{"name":"end","label":"End","order":1,"is_final":true,"transitions":[]}]}}`
	org := &models.Org{Name: "Custom Org", Slug: "custom-org", Metadata: customCfg}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db, event.NewBus())
	stages := svc.GetStages(context.Background(), org.ID)
	assert.Len(t, stages, 2)
	assert.Equal(t, Stage("start"), stages[0].Name)
}

func TestService_GetStages_OrgNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())

	stages := svc.GetStages(context.Background(), "nonexistent")
	assert.Len(t, stages, 8) // falls back to defaults
}

func TestExtractStage(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		expected string
	}{
		{"empty", "", ""},
		{"empty object", "{}", ""},
		{"has stage", `{"stage":"new_lead"}`, "new_lead"},
		{"no stage", `{"other":"field"}`, ""},
		{"invalid json", "not-json", ""},
		{"nested", `{"stage":"qualified","other":"val"}`, "qualified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractStage(tt.metadata))
		})
	}
}

// --- Handler Tests ---

func TestHandler_TransitionStage_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createTestHierarchy(t, db)
	thread := createTestThread(t, db, board.ID, "Lead", "{}")

	svc := NewService(db, event.NewBus())
	h := NewHandler(svc)

	body, _ := json.Marshal(TransitionInput{Stage: StageNewLead})
	req := httptest.NewRequest("POST", "/threads/"+thread.ID+"/stage", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", thread.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.TransitionStage(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var result TransitionResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "new_lead", result.NewStage)
}

func TestHandler_TransitionStage_EmptyStage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	h := NewHandler(svc)

	body, _ := json.Marshal(TransitionInput{Stage: ""})
	req := httptest.NewRequest("POST", "/threads/x/stage", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "x")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.TransitionStage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransitionStage_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("POST", "/threads/x/stage", bytes.NewReader([]byte("invalid")))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "x")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.TransitionStage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransitionStage_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	h := NewHandler(svc)

	body, _ := json.Marshal(TransitionInput{Stage: StageNewLead})
	req := httptest.NewRequest("POST", "/threads/nonexistent/stage", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.TransitionStage(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetStages(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	h := NewHandler(svc)

	req := httptest.NewRequest("GET", "/orgs/test-org/pipeline/stages", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", "test-org")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetStages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	stages, ok := resp["stages"].([]any)
	require.True(t, ok)
	assert.Len(t, stages, 8)
}

func TestHandler_TransitionStage_EmptyThreadParam(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	h := NewHandler(svc)

	body, _ := json.Marshal(TransitionInput{Stage: StageNewLead})
	req := httptest.NewRequest("POST", "/threads//stage", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("thread", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.TransitionStage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
