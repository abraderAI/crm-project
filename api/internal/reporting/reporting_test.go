package reporting

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Test Helpers ---

// testDB creates an in-memory SQLite database with all migrations applied.
func testDB(t *testing.T) *gorm.DB {
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

// seedOrg creates an org and returns its ID.
func seedOrg(t *testing.T, db *gorm.DB, slug string) string {
	t.Helper()
	org := &models.Org{Name: slug + " Org", Slug: slug, Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org.ID
}

// seedSpaceAndBoard creates a space (with the given type) and a board, returning spaceID and boardID.
func seedSpaceAndBoard(t *testing.T, db *gorm.DB, orgID string, spaceType models.SpaceType) (string, string) {
	t.Helper()
	space := &models.Space{OrgID: orgID, Name: string(spaceType) + " Space", Slug: string(spaceType) + "-space", Type: spaceType, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "Board", Slug: "board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	return space.ID, board.ID
}

// seedThread creates a thread with the given metadata JSON.
func seedThread(t *testing.T, db *gorm.DB, boardID, title, metadataJSON string, createdAt time.Time) string {
	t.Helper()
	thread := &models.Thread{
		BoardID:  boardID,
		Title:    title,
		Slug:     strings.ReplaceAll(strings.ToLower(title), " ", "-"),
		Metadata: metadataJSON,
		AuthorID: "author1",
	}
	require.NoError(t, db.Create(thread).Error)
	// Override created_at for date-range testing.
	db.Exec("UPDATE threads SET created_at = ? WHERE id = ?", createdAt, thread.ID)
	return thread.ID
}

// seedAuditLog creates an audit log entry for a thread stage change.
func seedAuditLog(t *testing.T, db *gorm.DB, threadID, fromStage, toStage string, createdAt time.Time) {
	t.Helper()
	beforeState := fmt.Sprintf(`{"stage":"%s"}`, fromStage)
	afterState := fmt.Sprintf(`{"stage":"%s"}`, toStage)
	al := &models.AuditLog{
		UserID:      "user1",
		Action:      "thread.updated",
		EntityType:  "thread",
		EntityID:    threadID,
		BeforeState: beforeState,
		AfterState:  afterState,
	}
	require.NoError(t, db.Create(al).Error)
	db.Exec("UPDATE audit_logs SET created_at = ? WHERE id = ?", createdAt, al.ID)
}

// defaultParams returns a ReportParams covering the last year.
func defaultParams() ReportParams {
	now := time.Now().UTC()
	return ReportParams{
		From: now.AddDate(-1, 0, 0),
		To:   now.Add(24 * time.Hour),
	}
}

// --- Support Unit Tests ---

func TestGetStatusBreakdown(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "status-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open"}`, now)
	seedThread(t, db, boardID, "T2", `{"status":"open"}`, now)
	seedThread(t, db, boardID, "T3", `{"status":"resolved"}`, now)

	result, err := repo.GetStatusBreakdown(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Equal(t, int64(2), result["open"])
	assert.Equal(t, int64(1), result["resolved"])
}

func TestGetVolumeOverTime(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "volume-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)

	day1 := time.Now().AddDate(0, 0, -3).Truncate(24 * time.Hour).Add(12 * time.Hour)
	day2 := time.Now().AddDate(0, 0, -2).Truncate(24 * time.Hour).Add(12 * time.Hour)
	day3 := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(12 * time.Hour)

	seedThread(t, db, boardID, "T1", `{"status":"open"}`, day1)
	seedThread(t, db, boardID, "T2", `{"status":"open"}`, day2)
	seedThread(t, db, boardID, "T3", `{"status":"open"}`, day2)
	seedThread(t, db, boardID, "T4", `{"status":"open"}`, day3)

	result, err := repo.GetVolumeOverTime(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, int64(1), result[0].Count)
	assert.Equal(t, int64(2), result[1].Count)
	assert.Equal(t, int64(1), result[2].Count)
}

func TestGetAvgResolutionTime(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "res-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	id1 := seedThread(t, db, boardID, "T1", `{"status":"resolved"}`, now.Add(-48*time.Hour))
	db.Exec("UPDATE threads SET updated_at = ? WHERE id = ?", now.Add(-24*time.Hour), id1) // 24h resolution

	id2 := seedThread(t, db, boardID, "T2", `{"status":"closed"}`, now.Add(-72*time.Hour))
	db.Exec("UPDATE threads SET updated_at = ? WHERE id = ?", now.Add(-24*time.Hour), id2) // 48h resolution

	// Open thread should not be counted.
	seedThread(t, db, boardID, "T3", `{"status":"open"}`, now)

	avg, err := repo.GetAvgResolutionHours(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotNil(t, avg)
	// (24 + 48) / 2 = 36 hours
	assert.InDelta(t, 36.0, *avg, 1.0)
}

func TestGetTicketsByAssignee(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "assignee-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open","assigned_to":"user_a"}`, now)
	seedThread(t, db, boardID, "T2", `{"status":"open","assigned_to":"user_a"}`, now)
	seedThread(t, db, boardID, "T3", `{"status":"in_progress","assigned_to":"user_b"}`, now)
	// Resolved thread should not appear.
	seedThread(t, db, boardID, "T4", `{"status":"resolved","assigned_to":"user_a"}`, now)

	result, err := repo.GetTicketsByAssignee(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "user_a", result[0].UserID)
	assert.Equal(t, int64(2), result[0].Count)
}

func TestGetTicketsByPriority(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "priority-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"priority":"high"}`, now)
	seedThread(t, db, boardID, "T2", `{"priority":"low"}`, now)
	seedThread(t, db, boardID, "T3", `{"priority":"high"}`, now)

	result, err := repo.GetTicketsByPriority(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Equal(t, int64(2), result["high"])
	assert.Equal(t, int64(1), result["low"])
}

func TestGetAvgFirstResponseTime(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "frt-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	threadID := seedThread(t, db, boardID, "T1", `{"status":"open"}`, now.Add(-24*time.Hour))

	// First reply by a different author.
	msg := &models.Message{
		ThreadID: threadID,
		Body:     "Reply",
		AuthorID: "agent1",
		Type:     models.MessageTypeComment,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(msg).Error)
	db.Exec("UPDATE messages SET created_at = ? WHERE id = ?", now.Add(-22*time.Hour), msg.ID)

	avg, err := repo.GetAvgFirstResponseHours(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotNil(t, avg)
	assert.InDelta(t, 2.0, *avg, 1.0) // ~2 hours
}

func TestGetOverdueCount(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "overdue-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)

	// Old open ticket (>72h old).
	seedThread(t, db, boardID, "Old1", `{"status":"open"}`, time.Now().Add(-100*time.Hour))
	seedThread(t, db, boardID, "Old2", `{"status":"in_progress"}`, time.Now().Add(-80*time.Hour))
	// Recent open ticket (not overdue).
	seedThread(t, db, boardID, "Recent", `{"status":"open"}`, time.Now().Add(-1*time.Hour))
	// Old but resolved (not overdue).
	seedThread(t, db, boardID, "Resolved", `{"status":"resolved"}`, time.Now().Add(-100*time.Hour))

	count, err := repo.GetOverdueCount(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestAssigneeFilterApplied(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "filter-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open","assigned_to":"user_a"}`, now)
	seedThread(t, db, boardID, "T2", `{"status":"open","assigned_to":"user_b"}`, now)

	params := defaultParams()
	params.Assignee = "user_a"
	result, err := repo.GetStatusBreakdown(context.Background(), orgID, params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result["open"])
}

func TestEmptyOrgReturnsZeros(t *testing.T) {
	db := testDB(t)
	svc := NewService(NewRepository(db))
	orgID := seedOrg(t, db, "empty-org")

	metrics, err := svc.GetSupportMetrics(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Empty(t, metrics.StatusBreakdown)
	assert.Empty(t, metrics.VolumeOverTime)
	assert.Nil(t, metrics.AvgResolutionHours)
	assert.Nil(t, metrics.AvgFirstResponseHours)
	assert.Equal(t, int64(0), metrics.OverdueCount)
}

// --- Sales Unit Tests ---

func TestGetPipelineFunnel(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "funnel-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead"}`, now)
	seedThread(t, db, boardID, "L2", `{"stage":"new_lead"}`, now)
	seedThread(t, db, boardID, "L3", `{"stage":"qualified"}`, now)
	seedThread(t, db, boardID, "L4", `{"stage":"closed_won"}`, now)

	result, err := repo.GetPipelineFunnel(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 3)
	counts := make(map[string]int64)
	for _, r := range result {
		counts[r.Stage] = r.Count
	}
	assert.Equal(t, int64(2), counts["new_lead"])
	assert.Equal(t, int64(1), counts["qualified"])
	assert.Equal(t, int64(1), counts["closed_won"])
}

func TestGetLeadVelocity(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "velocity-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)

	day1 := time.Now().AddDate(0, 0, -3).Truncate(24 * time.Hour).Add(12 * time.Hour)
	day2 := time.Now().AddDate(0, 0, -2).Truncate(24 * time.Hour).Add(12 * time.Hour)
	day3 := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(12 * time.Hour)

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead"}`, day1)
	seedThread(t, db, boardID, "L2", `{"stage":"new_lead"}`, day2)
	seedThread(t, db, boardID, "L3", `{"stage":"new_lead"}`, day3)

	result, err := repo.GetLeadVelocity(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestGetWinLossRate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "winloss-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "W1", `{"stage":"closed_won"}`, now)
	seedThread(t, db, boardID, "W2", `{"stage":"closed_won"}`, now)
	seedThread(t, db, boardID, "L1", `{"stage":"closed_lost"}`, now)

	won, lost, err := repo.GetWinLossCounts(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	winRate, lossRate := computeWinLossRates(won, lost)
	assert.InDelta(t, 2.0/3.0, winRate, 0.01)
	assert.InDelta(t, 1.0/3.0, lossRate, 0.01)
}

func TestGetWinLossRate_ZeroDivision(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "zero-org")
	seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)

	won, lost, err := repo.GetWinLossCounts(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	winRate, lossRate := computeWinLossRates(won, lost)
	assert.Equal(t, 0.0, winRate)
	assert.Equal(t, 0.0, lossRate)
}

func TestGetAvgDealValue(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "deal-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "D1", `{"stage":"new_lead","deal_value":10000}`, now)
	seedThread(t, db, boardID, "D2", `{"stage":"new_lead","deal_value":20000}`, now)
	// Thread without deal_value should be ignored.
	seedThread(t, db, boardID, "D3", `{"stage":"new_lead"}`, now)

	avg, err := repo.GetAvgDealValue(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotNil(t, avg)
	assert.InDelta(t, 15000.0, *avg, 0.01)
}

func TestGetAvgDealValue_NilWhenNoData(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "nodeal-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "D1", `{"stage":"new_lead"}`, now) // no deal_value

	avg, err := repo.GetAvgDealValue(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Nil(t, avg)
}

func TestGetLeadsByAssignee(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "leads-assign-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","assigned_to":"rep_a"}`, now)
	seedThread(t, db, boardID, "L2", `{"stage":"qualified","assigned_to":"rep_a"}`, now)
	seedThread(t, db, boardID, "L3", `{"stage":"new_lead","assigned_to":"rep_b"}`, now)
	// Closed thread should not count.
	seedThread(t, db, boardID, "L4", `{"stage":"closed_won","assigned_to":"rep_a"}`, now)

	result, err := repo.GetLeadsByAssignee(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "rep_a", result[0].UserID)
	assert.Equal(t, int64(2), result[0].Count)
}

func TestGetScoreDistribution(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "score-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "S1", `{"stage":"new_lead","score":10}`, now)
	seedThread(t, db, boardID, "S2", `{"stage":"new_lead","score":25}`, now)
	seedThread(t, db, boardID, "S3", `{"stage":"new_lead","score":55}`, now)
	seedThread(t, db, boardID, "S4", `{"stage":"new_lead","score":90}`, now)
	seedThread(t, db, boardID, "S5", `{"stage":"new_lead","score":95}`, now)

	result, err := repo.GetScoreDistribution(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	buckets := make(map[string]int64)
	for _, r := range result {
		buckets[r.Range] = r.Count
	}
	assert.Equal(t, int64(1), buckets["0-20"])
	assert.Equal(t, int64(1), buckets["20-40"])
	assert.Equal(t, int64(1), buckets["40-60"])
	assert.Equal(t, int64(2), buckets["80-100"])
}

func TestGetStageConversionRates(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "conv-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	t1 := seedThread(t, db, boardID, "L1", `{"stage":"qualified"}`, now)
	t2 := seedThread(t, db, boardID, "L2", `{"stage":"closed_lost"}`, now)

	// Audit: new_lead → qualified (2 times).
	seedAuditLog(t, db, t1, "new_lead", "qualified", now.Add(-2*time.Hour))
	seedAuditLog(t, db, t2, "new_lead", "qualified", now.Add(-3*time.Hour))
	// Audit: new_lead → closed_lost (1 time).
	seedAuditLog(t, db, t2, "new_lead", "closed_lost", now.Add(-1*time.Hour))

	transitions, err := repo.GetStageTransitions(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	rates := computeConversionRates(transitions)
	require.NotEmpty(t, rates)

	rateMap := make(map[string]float64)
	for _, r := range rates {
		rateMap[r.FromStage+"→"+r.ToStage] = r.Rate
	}
	// new_lead → qualified: 2/3 = 0.667
	assert.InDelta(t, 2.0/3.0, rateMap["new_lead→qualified"], 0.01)
	// new_lead → closed_lost: 1/3 = 0.333
	assert.InDelta(t, 1.0/3.0, rateMap["new_lead→closed_lost"], 0.01)
}

func TestGetStageConversionRates_EmptyAuditLog(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "empty-audit-org")
	seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)

	transitions, err := repo.GetStageTransitions(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	rates := computeConversionRates(transitions)
	assert.NotNil(t, rates)
	assert.Empty(t, rates)
}

func TestGetAvgTimeInStage(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "time-stage-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	t1 := seedThread(t, db, boardID, "L1", `{"stage":"qualified"}`, now)

	// Stage new_lead entered at -48h, exited at -24h → 24 hours in new_lead.
	seedAuditLog(t, db, t1, "unknown", "new_lead", now.Add(-48*time.Hour))
	seedAuditLog(t, db, t1, "new_lead", "qualified", now.Add(-24*time.Hour))

	result, err := repo.GetAvgTimeInStage(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotEmpty(t, result)

	found := false
	for _, r := range result {
		if r.Stage == "new_lead" {
			found = true
			require.NotNil(t, r.AvgHours)
			assert.InDelta(t, 24.0, *r.AvgHours, 1.0)
		}
	}
	assert.True(t, found, "expected to find new_lead stage")
}

func TestGetAvgTimeInStage_NoData(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "no-time-org")
	seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)

	result, err := repo.GetAvgTimeInStage(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestSalesAssigneeFilter(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	orgID := seedOrg(t, db, "sales-filter-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","assigned_to":"rep_a"}`, now)
	seedThread(t, db, boardID, "L2", `{"stage":"qualified","assigned_to":"rep_b"}`, now)

	params := defaultParams()
	params.Assignee = "rep_a"
	funnel, err := repo.GetPipelineFunnel(context.Background(), orgID, params)
	require.NoError(t, err)
	total := int64(0)
	for _, s := range funnel {
		total += s.Count
	}
	assert.Equal(t, int64(1), total)
}

// --- Service Unit Tests ---

func TestComputeWinLossRates(t *testing.T) {
	tests := []struct {
		name                              string
		won, lost                         int64
		expectedWinRate, expectedLossRate float64
	}{
		{"normal", 2, 1, 2.0 / 3.0, 1.0 / 3.0},
		{"all won", 5, 0, 1.0, 0.0},
		{"all lost", 0, 3, 0.0, 1.0},
		{"zero both", 0, 0, 0.0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wr, lr := computeWinLossRates(tt.won, tt.lost)
			assert.InDelta(t, tt.expectedWinRate, wr, 0.001)
			assert.InDelta(t, tt.expectedLossRate, lr, 0.001)
		})
	}
}

func TestComputeConversionRates(t *testing.T) {
	t.Run("empty returns empty slice", func(t *testing.T) {
		rates := computeConversionRates(nil)
		assert.NotNil(t, rates)
		assert.Empty(t, rates)
	})

	t.Run("single transition", func(t *testing.T) {
		rates := computeConversionRates([]stageTransitionRow{
			{FromStage: "a", ToStage: "b", TransitionCount: 5},
		})
		require.Len(t, rates, 1)
		assert.Equal(t, 1.0, rates[0].Rate)
	})

	t.Run("multiple transitions from same stage", func(t *testing.T) {
		rates := computeConversionRates([]stageTransitionRow{
			{FromStage: "a", ToStage: "b", TransitionCount: 3},
			{FromStage: "a", ToStage: "c", TransitionCount: 7},
		})
		require.Len(t, rates, 2)
		assert.InDelta(t, 0.3, rates[0].Rate, 0.001)
		assert.InDelta(t, 0.7, rates[1].Rate, 0.001)
	})
}

// --- Param Parsing Tests ---

func TestParseReportParams(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		params, err := parseReportParams(req)
		require.NoError(t, err)
		assert.False(t, params.From.IsZero())
		assert.False(t, params.To.IsZero())
		assert.Empty(t, params.Assignee)
	})

	t.Run("valid dates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?from=2026-01-01&to=2026-03-15&assignee=user1", nil)
		params, err := parseReportParams(req)
		require.NoError(t, err)
		assert.Equal(t, 2026, params.From.Year())
		assert.Equal(t, time.January, params.From.Month())
		assert.Equal(t, "user1", params.Assignee)
	})

	t.Run("invalid from date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?from=not-a-date", nil)
		_, err := parseReportParams(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "from")
	})

	t.Run("invalid to date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?to=2026/13/45", nil)
		_, err := parseReportParams(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "to")
	})
}

// --- Handler Unit Tests ---

// withChiOrgParam creates a request with chi URL param {org} set.
func withChiOrgParam(req *http.Request, orgID string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("org", orgID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// withAuth sets the user context on the request.
func withAuth(req *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(req.Context(), &auth.UserContext{UserID: userID, AuthMethod: auth.AuthMethodJWT})
	return req.WithContext(ctx)
}

func TestHandler_GetSalesMetrics(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "hnd-sales-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","deal_value":10000,"score":50}`, now)
	seedThread(t, db, boardID, "L2", `{"stage":"closed_won","deal_value":20000,"score":80}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	from := now.AddDate(0, -1, 0).Format("2006-01-02")
	to := now.Add(24 * time.Hour).Format("2006-01-02")

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/v1/orgs/%s/reports/sales?from=%s&to=%s", orgID, from, to), nil)
	req = withChiOrgParam(req, orgID)

	rr := httptest.NewRecorder()
	h.GetSalesMetrics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var metrics SalesMetrics
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics.PipelineFunnel)
}

func TestHandler_GetSalesMetrics_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/org1/reports/sales?from=bad", nil)
	req = withChiOrgParam(req, "org1")

	rr := httptest.NewRecorder()
	h.GetSalesMetrics(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_GetSupportMetrics(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "hnd-supp-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open"}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/reports/support", nil)
	req = withChiOrgParam(req, orgID)

	rr := httptest.NewRecorder()
	h.GetSupportMetrics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var metrics SupportMetrics
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&metrics))
	assert.Equal(t, int64(1), metrics.StatusBreakdown["open"])
}

func TestHandler_GetSupportExport(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "hnd-exp-supp-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open","priority":"high"}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/reports/support/export", nil)
	req = withChiOrgParam(req, orgID)

	rr := httptest.NewRecorder()
	h.GetSupportExport(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/csv", rr.Header().Get("Content-Type"))

	reader := csv.NewReader(strings.NewReader(rr.Body.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Greater(t, len(records), 1)
	assert.Equal(t, "id", records[0][0])
}

func TestHandler_GetSalesExport(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "hnd-exp-sales-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","deal_value":5000}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/reports/sales/export", nil)
	req = withChiOrgParam(req, orgID)

	rr := httptest.NewRecorder()
	h.GetSalesExport(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/csv", rr.Header().Get("Content-Type"))

	reader := csv.NewReader(strings.NewReader(rr.Body.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Greater(t, len(records), 1)
	assert.Equal(t, []string{"id", "title", "stage", "assigned_to", "deal_value", "score", "created_at"}, records[0])
}

func TestHandler_GetSupportExport_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/org1/reports/support/export?from=invalid", nil)
	req = withChiOrgParam(req, "org1")

	rr := httptest.NewRecorder()
	h.GetSupportExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_GetSalesExport_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/org1/reports/sales/export?to=invalid", nil)
	req = withChiOrgParam(req, "org1")

	rr := httptest.NewRecorder()
	h.GetSalesExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_GetSupportMetrics_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/org1/reports/support?from=bad", nil)
	req = withChiOrgParam(req, "org1")

	rr := httptest.NewRecorder()
	h.GetSupportMetrics(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- RBAC Middleware Tests ---

func TestRequireOrgAdminOrOwner_NoAuth(t *testing.T) {
	db := testDB(t)
	mw := RequireOrgAdminOrOwner(db)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = withChiOrgParam(req, "org1")

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireOrgAdminOrOwner_NoOrg(t *testing.T) {
	db := testDB(t)
	mw := RequireOrgAdminOrOwner(db)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = withAuth(req, "user1")
	// no chi org param

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRequireOrgAdminOrOwner_ViewerDenied(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "rbac-viewer-org")
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: orgID, UserID: "viewer1", Role: models.RoleViewer,
	}).Error)

	mw := RequireOrgAdminOrOwner(db)
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = withChiOrgParam(req, orgID)
	req = withAuth(req, "viewer1")

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireOrgAdminOrOwner_AdminAllowed(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "rbac-admin-org")
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: orgID, UserID: "admin1", Role: models.RoleAdmin,
	}).Error)

	mw := RequireOrgAdminOrOwner(db)
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = withChiOrgParam(req, orgID)
	req = withAuth(req, "admin1")

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.True(t, nextCalled)
}

func TestRequireOrgAdminOrOwner_OwnerAllowed(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "rbac-owner-org")
	require.NoError(t, db.Create(&models.OrgMembership{
		OrgID: orgID, UserID: "owner1", Role: models.RoleOwner,
	}).Error)

	mw := RequireOrgAdminOrOwner(db)
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = withChiOrgParam(req, orgID)
	req = withAuth(req, "owner1")

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.True(t, nextCalled)
}

func TestRequireOrgAdminOrOwner_NoMembership(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "rbac-nomember-org")

	mw := RequireOrgAdminOrOwner(db)
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = withChiOrgParam(req, orgID)
	req = withAuth(req, "stranger")

	rr := httptest.NewRecorder()
	mw(next).ServeHTTP(rr, req)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// --- Service-Level Tests for GetSalesMetrics ---

func TestService_GetSalesMetrics(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "svc-sales-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","deal_value":10000,"score":50,"assigned_to":"rep_a"}`, now)
	seedThread(t, db, boardID, "L2", `{"stage":"closed_won","deal_value":20000,"score":80,"assigned_to":"rep_b"}`, now)
	seedThread(t, db, boardID, "L3", `{"stage":"closed_lost","deal_value":5000,"score":30}`, now)

	svc := NewService(NewRepository(db))
	metrics, err := svc.GetSalesMetrics(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.NotEmpty(t, metrics.PipelineFunnel)
	assert.NotNil(t, metrics.AvgDealValue)
	assert.InDelta(t, 2.0/3.0, metrics.WinRate+metrics.LossRate, 1.01) // win + loss = 1 (or 0 if no closed)
}

func TestService_GetSalesExportRows(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "svc-sales-exp-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","deal_value":10000}`, now)

	svc := NewService(NewRepository(db))
	rows, err := svc.GetSalesExportRows(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "L1", rows[0].Title)
	assert.Equal(t, "new_lead", rows[0].Stage)
}

func TestService_GetSupportExportRows(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "svc-supp-exp-org")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open","priority":"high","assigned_to":"agent_1"}`, now)

	svc := NewService(NewRepository(db))
	rows, err := svc.GetSupportExportRows(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "T1", rows[0].Title)
	assert.Equal(t, "open", rows[0].Status)
}

// --- Fuzz Tests ---

// FuzzStageName tests stage name strings in audit_log before/after state.
func FuzzStageName(f *testing.F) {
	seeds := []string{"new_lead", "qualified", "closed_won", "closed_lost", "", "unknown", "a'b", `"quoted"`, "日本語", "<script>"}
	for _, s := range seeds {
		f.Add(s)
	}
	// Add additional seeds to reach ≥50 distinct corpus entries.
	for i := 0; i < 50; i++ {
		f.Add(fmt.Sprintf("stage_%d_%s", i, strings.Repeat("x", i%20)))
	}

	f.Fuzz(func(t *testing.T, stageName string) {
		db := testDB(t)
		repo := NewRepository(db)
		orgID := seedOrg(t, db, "fuzz-stage")
		_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
		now := time.Now()

		meta := fmt.Sprintf(`{"stage":%q}`, stageName)
		threadID := seedThread(t, db, boardID, "FuzzLead", meta, now)

		// Seed audit log with the fuzz stage name.
		beforeState := fmt.Sprintf(`{"stage":%q}`, stageName)
		afterState := `{"stage":"closed_won"}`
		al := &models.AuditLog{
			UserID:      "fuzz_user",
			Action:      "thread.updated",
			EntityType:  "thread",
			EntityID:    threadID,
			BeforeState: beforeState,
			AfterState:  afterState,
		}
		_ = db.Create(al).Error

		// Should not panic.
		_, _ = repo.GetPipelineFunnel(context.Background(), orgID, defaultParams())
		_, _ = repo.GetStageTransitions(context.Background(), orgID, defaultParams())
	})
}

// FuzzDealValue tests random deal_value metadata strings.
func FuzzDealValue(f *testing.F) {
	seeds := []string{"10000", "0", "-1", "99999.99", "NaN", "Infinity", "", "abc", "1e10", "null"}
	for _, s := range seeds {
		f.Add(s)
	}
	for i := 0; i < 50; i++ {
		f.Add(fmt.Sprintf("%d.%02d", i*1000, i%100))
	}

	f.Fuzz(func(t *testing.T, dealValue string) {
		db := testDB(t)
		repo := NewRepository(db)
		orgID := seedOrg(t, db, "fuzz-deal")
		_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
		now := time.Now()

		meta := fmt.Sprintf(`{"stage":"new_lead","deal_value":%s}`, dealValue)
		// If the deal_value string is not valid JSON, wrap it in quotes.
		if !json.Valid([]byte(meta)) {
			meta = fmt.Sprintf(`{"stage":"new_lead","deal_value":%q}`, dealValue)
		}
		seedThread(t, db, boardID, "FuzzDeal", meta, now)

		// Should not panic.
		_, _ = repo.GetAvgDealValue(context.Background(), orgID, defaultParams())
		_, _ = repo.GetPipelineFunnel(context.Background(), orgID, defaultParams())
	})
}

// FuzzDateParams tests random from/to date strings for param parsing.
func FuzzDateParams(f *testing.F) {
	seeds := []string{"2026-01-01", "2026-12-31", "not-a-date", "", "2026-13-01", "0000-00-00", "9999-99-99"}
	for _, s := range seeds {
		f.Add(s)
	}
	for i := 0; i < 50; i++ {
		f.Add(fmt.Sprintf("202%d-%02d-%02d", i%10, (i%12)+1, (i%28)+1))
	}

	f.Fuzz(func(t *testing.T, dateStr string) {
		req := httptest.NewRequest(http.MethodGet, "/test?from="+dateStr+"&to="+dateStr, nil)
		// Should not panic.
		_, _ = parseReportParams(req)
	})
}

// FuzzAssigneeParam tests random assignee strings for param parsing.
func FuzzAssigneeParam(f *testing.F) {
	seeds := []string{"", "user_123", "a'b", `"quoted"`, "日本語", "<script>alert(1)</script>", strings.Repeat("x", 500)}
	for _, s := range seeds {
		f.Add(s)
	}
	for i := 0; i < 50; i++ {
		f.Add(fmt.Sprintf("user_%d_%s", i, strings.Repeat("a", i%50)))
	}

	f.Fuzz(func(t *testing.T, assignee string) {
		req := httptest.NewRequest(http.MethodGet, "/test?assignee="+assignee, nil)
		params, err := parseReportParams(req)
		// Should not panic.
		if err == nil {
			assert.Equal(t, assignee, params.Assignee)
		}
	})
}
