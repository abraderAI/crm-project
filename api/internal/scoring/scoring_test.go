package scoring

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
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

// --- Engine Tests ---

func TestDefaultRules(t *testing.T) {
	rules := DefaultRules()
	assert.True(t, len(rules) >= 10)
}

func TestEvaluate_EmptyMetadata(t *testing.T) {
	bd := Evaluate(DefaultRules(), "{}")
	assert.Equal(t, 0, bd.TotalScore)
	assert.Len(t, bd.Rules, len(DefaultRules()))
}

func TestEvaluate_StageNewLead(t *testing.T) {
	bd := Evaluate(DefaultRules(), `{"stage":"new_lead"}`)
	assert.Equal(t, 5, bd.TotalScore)
}

func TestEvaluate_StageQualifiedWithPriority(t *testing.T) {
	bd := Evaluate(DefaultRules(), `{"stage":"qualified","priority":"high"}`)
	assert.Equal(t, 50, bd.TotalScore) // 30 (stage) + 20 (priority)
}

func TestEvaluate_WithCompanyAndEmail(t *testing.T) {
	bd := Evaluate(DefaultRules(), `{"stage":"new_lead","company":"Acme","contact_email":"a@b.com"}`)
	assert.Equal(t, 20, bd.TotalScore) // 5 (stage) + 10 (company) + 5 (email)
}

func TestEvaluate_HighDealValue(t *testing.T) {
	bd := Evaluate(DefaultRules(), `{"stage":"proposal","deal_value":50000}`)
	assert.Equal(t, 85, bd.TotalScore) // 50 (stage) + 25 (high value) + 10 (medium value)
}

func TestEvaluate_MediumDealValue(t *testing.T) {
	bd := Evaluate(DefaultRules(), `{"stage":"new_lead","deal_value":5000}`)
	assert.Equal(t, 15, bd.TotalScore) // 5 (stage) + 10 (medium value)
}

func TestEvaluate_InvalidJSON(t *testing.T) {
	bd := Evaluate(DefaultRules(), "not-json")
	assert.Equal(t, 0, bd.TotalScore)
}

func TestEvaluate_NilMetadata(t *testing.T) {
	bd := Evaluate(DefaultRules(), "")
	assert.Equal(t, 0, bd.TotalScore)
}

func TestEvaluate_CustomRules(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test_eq", Path: "status", Operator: "eq", Value: "active", Points: 50},
		{Name: "test_exists", Path: "name", Operator: "exists", Value: "", Points: 25},
	}
	bd := Evaluate(rules, `{"status":"active","name":"Test"}`)
	assert.Equal(t, 75, bd.TotalScore)
}

func TestEvaluate_ContainsOperator(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test_contains", Path: "description", Operator: "contains", Value: "urgent", Points: 30},
	}
	bd := Evaluate(rules, `{"description":"This is urgent please help"}`)
	assert.Equal(t, 30, bd.TotalScore)
}

func TestEvaluate_ContainsOperator_NoMatch(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test_contains", Path: "description", Operator: "contains", Value: "urgent", Points: 30},
	}
	bd := Evaluate(rules, `{"description":"Normal request"}`)
	assert.Equal(t, 0, bd.TotalScore)
}

func TestEvaluate_NumericComparisons(t *testing.T) {
	rules := []ScoringRule{
		{Name: "gt", Path: "value", Operator: "gt", Value: "100", Points: 10},
		{Name: "gte", Path: "value", Operator: "gte", Value: "200", Points: 20},
		{Name: "lt", Path: "count", Operator: "lt", Value: "5", Points: 5},
		{Name: "lte", Path: "count", Operator: "lte", Value: "3", Points: 3},
	}
	bd := Evaluate(rules, `{"value":200,"count":3}`)
	assert.Equal(t, 38, bd.TotalScore) // gt:10 + gte:20 + lt:5 + lte:3
}

func TestEvaluate_StringNumericValue(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test", Path: "amount", Operator: "gt", Value: "100", Points: 10},
	}
	bd := Evaluate(rules, `{"amount":"200"}`)
	assert.Equal(t, 10, bd.TotalScore)
}

func TestEvaluate_UnknownOperator(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test", Path: "x", Operator: "unknown", Value: "y", Points: 10},
	}
	bd := Evaluate(rules, `{"x":"y"}`)
	assert.Equal(t, 0, bd.TotalScore)
}

func TestEvaluate_NestedPath(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test", Path: "contact.email", Operator: "exists", Value: "", Points: 15},
	}
	bd := Evaluate(rules, `{"contact":{"email":"a@b.com"}}`)
	assert.Equal(t, 15, bd.TotalScore)
}

func TestEvaluate_MissingNestedPath(t *testing.T) {
	rules := []ScoringRule{
		{Name: "test", Path: "contact.email", Operator: "exists", Value: "", Points: 15},
	}
	bd := Evaluate(rules, `{"contact":{}}`)
	assert.Equal(t, 0, bd.TotalScore)
}

func TestResolveMetaPath(t *testing.T) {
	meta := map[string]any{"a": map[string]any{"b": "value"}}
	val, ok := resolveMetaPath(meta, "a.b")
	assert.True(t, ok)
	assert.Equal(t, "value", val)
}

func TestResolveMetaPath_Missing(t *testing.T) {
	meta := map[string]any{"a": "b"}
	_, ok := resolveMetaPath(meta, "x")
	assert.False(t, ok)
}

func TestResolveMetaPath_NotMap(t *testing.T) {
	meta := map[string]any{"a": "string"}
	_, ok := resolveMetaPath(meta, "a.b")
	assert.False(t, ok)
}

func TestCompareNumeric_InvalidThreshold(t *testing.T) {
	assert.False(t, compareNumeric(100.0, "gt", "not-a-number"))
}

func TestCompareNumeric_NonNumericValue(t *testing.T) {
	assert.False(t, compareNumeric(true, "gt", "10"))
}

func TestCompareNumeric_IntValue(t *testing.T) {
	assert.True(t, compareNumeric(100, "gt", "50"))
}

func TestParseRulesFromMetadata_Empty(t *testing.T) {
	assert.Nil(t, ParseRulesFromMetadata(""))
	assert.Nil(t, ParseRulesFromMetadata("{}"))
}

func TestParseRulesFromMetadata_NoRules(t *testing.T) {
	assert.Nil(t, ParseRulesFromMetadata(`{"other":"field"}`))
}

func TestParseRulesFromMetadata_Valid(t *testing.T) {
	meta := `{"scoring_rules":[{"name":"custom","path":"stage","operator":"eq","value":"hot","points":50}]}`
	rules := ParseRulesFromMetadata(meta)
	require.NotNil(t, rules)
	assert.Len(t, rules, 1)
	assert.Equal(t, "custom", rules[0].Name)
}

func TestParseRulesFromMetadata_InvalidJSON(t *testing.T) {
	assert.Nil(t, ParseRulesFromMetadata("not-json"))
}

func TestParseRulesFromMetadata_EmptyRules(t *testing.T) {
	assert.Nil(t, ParseRulesFromMetadata(`{"scoring_rules":[]}`))
}

func TestParseRulesFromMetadata_InvalidFormat(t *testing.T) {
	assert.Nil(t, ParseRulesFromMetadata(`{"scoring_rules":"not-array"}`))
}

// --- Service Tests ---

func TestService_ScoreThread_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead", AuthorID: "u1", Metadata: `{"stage":"qualified","priority":"high"}`}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, event.NewBus())
	bd, err := svc.ScoreThread(context.Background(), thread.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, bd.TotalScore) // 30 + 20

	// Verify score saved in metadata.
	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	var meta map[string]any
	require.NoError(t, json.Unmarshal([]byte(updated.Metadata), &meta))
	assert.Equal(t, float64(50), meta["lead_score"])
}

func TestService_ScoreThread_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	_, err := svc.ScoreThread(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "thread not found")
}

func TestService_ScoreThread_EmptyThreadID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	_, err := svc.ScoreThread(context.Background(), "")
	assert.Error(t, err)
}

func TestService_GetScore_Success(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead2", AuthorID: "u1", Metadata: `{"stage":"new_lead"}`}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, event.NewBus())
	bd, err := svc.GetScore(context.Background(), thread.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, bd.TotalScore)
}

func TestService_GetScore_ThreadNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	_, err := svc.GetScore(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestService_GetScore_EmptyID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	_, err := svc.GetScore(context.Background(), "")
	assert.Error(t, err)
}

func TestService_HandleStageChanged(t *testing.T) {
	db := setupTestDB(t)
	_, _, board := createHierarchy(t, db)
	thread := &models.Thread{BoardID: board.ID, Title: "Lead", Slug: "lead3", AuthorID: "u1", Metadata: `{"stage":"proposal"}`}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, event.NewBus())
	svc.HandleStageChanged(event.Event{EntityType: "thread", EntityID: thread.ID})

	// Verify score updated.
	var updated models.Thread
	require.NoError(t, db.First(&updated, "id = ?", thread.ID).Error)
	assert.Contains(t, updated.Metadata, "lead_score")
}

func TestService_HandleStageChanged_NonThread(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	// Should not panic.
	svc.HandleStageChanged(event.Event{EntityType: "org", EntityID: "some-id"})
}

func TestService_HandleStageChanged_EmptyEntityID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db, event.NewBus())
	svc.HandleStageChanged(event.Event{EntityType: "thread", EntityID: ""})
}

func TestService_CustomOrgRules(t *testing.T) {
	db := setupTestDB(t)
	customRules := `{"scoring_rules":[{"name":"hot_lead","path":"temperature","operator":"eq","value":"hot","points":100}]}`
	org := &models.Org{Name: "Custom Org", Slug: "custom-scoring", Metadata: customRules}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "Sales", Slug: "sales", Type: models.SpaceTypeCRM, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Leads", Slug: "leads", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "Hot", Slug: "hot", AuthorID: "u1", Metadata: `{"temperature":"hot"}`}
	require.NoError(t, db.Create(thread).Error)

	svc := NewService(db, event.NewBus())
	bd, err := svc.ScoreThread(context.Background(), thread.ID)
	require.NoError(t, err)
	assert.Equal(t, 100, bd.TotalScore)
}
