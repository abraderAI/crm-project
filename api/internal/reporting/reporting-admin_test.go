package reporting

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Platform-Wide Support Metrics ---

func TestGetPlatformSupportMetrics(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)

	// Seed 2 orgs with support threads.
	org1 := seedOrg(t, db, "plat-supp-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeSupport)

	org2 := seedOrg(t, db, "plat-supp-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeSupport)

	now := time.Now()
	seedThread(t, db, board1, "T1", `{"status":"open","priority":"high"}`, now)
	seedThread(t, db, board1, "T2", `{"status":"resolved","priority":"low"}`, now.Add(-48*time.Hour))
	seedThread(t, db, board2, "T3", `{"status":"open","priority":"medium"}`, now)
	seedThread(t, db, board2, "T4", `{"status":"in_progress","priority":"high"}`, now)

	r := repo.(*repository)
	metrics, err := r.GetPlatformSupportMetrics(context.Background(), defaultParams())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Total status counts should be sum of per-org.
	totalStatus := int64(0)
	for _, v := range metrics.StatusBreakdown {
		totalStatus += v
	}
	assert.Equal(t, int64(4), totalStatus)
	assert.Equal(t, int64(2), metrics.StatusBreakdown["open"])
	assert.Equal(t, int64(1), metrics.StatusBreakdown["in_progress"])

	// Platform totals = sum of per-org totals.
	org1Metrics, err := repo.GetStatusBreakdown(context.Background(), org1, defaultParams())
	require.NoError(t, err)
	org2Metrics, err := repo.GetStatusBreakdown(context.Background(), org2, defaultParams())
	require.NoError(t, err)

	for status, count := range metrics.StatusBreakdown {
		assert.Equal(t, count, org1Metrics[status]+org2Metrics[status],
			"platform total for status %q should equal sum of per-org totals", status)
	}
}

// --- Platform-Wide Sales Metrics ---

func TestGetPlatformSalesMetrics(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)

	org1 := seedOrg(t, db, "plat-sales-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeCRM)

	org2 := seedOrg(t, db, "plat-sales-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeCRM)

	now := time.Now()
	seedThread(t, db, board1, "L1", `{"stage":"new_lead","deal_value":10000,"score":50}`, now)
	seedThread(t, db, board1, "L2", `{"stage":"closed_won","deal_value":20000,"score":80}`, now)
	seedThread(t, db, board2, "L3", `{"stage":"closed_lost","deal_value":5000,"score":30}`, now)
	seedThread(t, db, board2, "L4", `{"stage":"qualified","deal_value":15000,"score":60}`, now)

	r := repo.(*repository)
	metrics, err := r.GetPlatformSalesMetrics(context.Background(), defaultParams())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	// Funnel should include all 4 leads across both orgs.
	totalFunnel := int64(0)
	for _, s := range metrics.PipelineFunnel {
		totalFunnel += s.Count
	}
	assert.Equal(t, int64(4), totalFunnel)

	// Win/loss rates: 1 won, 1 lost → 50/50.
	assert.InDelta(t, 0.5, metrics.WinRate, 0.01)
	assert.InDelta(t, 0.5, metrics.LossRate, 0.01)

	// Avg deal value across all leads.
	require.NotNil(t, metrics.AvgDealValue)
	assert.InDelta(t, 12500.0, *metrics.AvgDealValue, 0.01)
}

// --- Per-Org Support Breakdown ---

func TestGetOrgSupportBreakdown(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db).(*repository)

	org1 := seedOrg(t, db, "bd-supp-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeSupport)

	org2 := seedOrg(t, db, "bd-supp-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeSupport)

	now := time.Now()
	// Org1: 5 threads (3 open, 1 resolved, 1 closed).
	seedThread(t, db, board1, "T1", `{"status":"open"}`, now)
	seedThread(t, db, board1, "T2", `{"status":"open"}`, now)
	seedThread(t, db, board1, "T3", `{"status":"open"}`, now.Add(-100*time.Hour)) // overdue
	id := seedThread(t, db, board1, "T4", `{"status":"resolved"}`, now.Add(-24*time.Hour))
	db.Exec("UPDATE threads SET updated_at = ? WHERE id = ?", now.Add(-12*time.Hour), id)
	seedThread(t, db, board1, "T5", `{"status":"closed"}`, now)

	// Org2: 2 threads (1 open, 1 in_progress).
	seedThread(t, db, board2, "T6", `{"status":"open"}`, now)
	seedThread(t, db, board2, "T7", `{"status":"in_progress"}`, now)

	breakdown, err := repo.GetOrgSupportBreakdown(context.Background(), defaultParams())
	require.NoError(t, err)
	require.Len(t, breakdown, 2)

	// Ordered by total_in_range DESC — org1 has 5, org2 has 2.
	assert.Equal(t, org1, breakdown[0].OrgID)
	assert.Equal(t, int64(5), breakdown[0].TotalInRange)
	assert.Equal(t, int64(3), breakdown[0].OpenCount)
	assert.Equal(t, int64(1), breakdown[0].OverdueCount)
	require.NotNil(t, breakdown[0].AvgResolutionHours)

	assert.Equal(t, org2, breakdown[1].OrgID)
	assert.Equal(t, int64(2), breakdown[1].TotalInRange)
	assert.Equal(t, int64(2), breakdown[1].OpenCount)
}

func TestGetOrgSupportBreakdown_FirstResponseJoin(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db).(*repository)

	orgID := seedOrg(t, db, "bd-frt")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	threadID := seedThread(t, db, boardID, "T1", `{"status":"open"}`, now.Add(-24*time.Hour))
	msg := &models.Message{
		ThreadID: threadID,
		Body:     "Reply",
		AuthorID: "agent1",
		Type:     models.MessageTypeComment,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(msg).Error)
	db.Exec("UPDATE messages SET created_at = ? WHERE id = ?", now.Add(-22*time.Hour), msg.ID)

	frMap, err := repo.GetOrgFirstResponseBreakdown(context.Background(), defaultParams())
	require.NoError(t, err)
	require.Contains(t, frMap, orgID)
	require.NotNil(t, frMap[orgID])
	assert.InDelta(t, 2.0, *frMap[orgID], 1.0)
}

// --- Per-Org Sales Breakdown ---

func TestGetOrgSalesBreakdown(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db).(*repository)

	org1 := seedOrg(t, db, "bd-sales-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeCRM)

	org2 := seedOrg(t, db, "bd-sales-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeCRM)

	now := time.Now()
	// Org1: 3 leads (1 won, 1 lost, 1 open).
	seedThread(t, db, board1, "L1", `{"stage":"closed_won","deal_value":10000}`, now)
	seedThread(t, db, board1, "L2", `{"stage":"closed_lost","deal_value":5000}`, now)
	seedThread(t, db, board1, "L3", `{"stage":"new_lead","deal_value":15000}`, now)

	// Org2: 2 leads (0 won, 0 lost).
	seedThread(t, db, board2, "L4", `{"stage":"new_lead","deal_value":20000}`, now)
	seedThread(t, db, board2, "L5", `{"stage":"qualified","deal_value":30000}`, now)

	breakdown, err := repo.GetOrgSalesBreakdown(context.Background(), defaultParams())
	require.NoError(t, err)
	require.Len(t, breakdown, 2)

	// Ordered by total_leads DESC — org1 has 3, org2 has 2.
	assert.Equal(t, org1, breakdown[0].OrgID)
	assert.Equal(t, int64(3), breakdown[0].TotalLeads)
	assert.InDelta(t, 0.5, breakdown[0].WinRate, 0.01) // 1/(1+1) = 0.5
	assert.Equal(t, int64(1), breakdown[0].OpenPipelineCount)

	assert.Equal(t, org2, breakdown[1].OrgID)
	assert.Equal(t, int64(2), breakdown[1].TotalLeads)
	assert.Equal(t, 0.0, breakdown[1].WinRate) // 0 won, 0 lost → zero-division safe
	assert.Equal(t, int64(2), breakdown[1].OpenPipelineCount)
}

func TestGetOrgSalesBreakdown_ZeroDivisionSafe(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db).(*repository)

	orgID := seedOrg(t, db, "bd-zero")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	// All open — no won or lost.
	seedThread(t, db, boardID, "L1", `{"stage":"new_lead"}`, now)
	seedThread(t, db, boardID, "L2", `{"stage":"qualified"}`, now)

	breakdown, err := repo.GetOrgSalesBreakdown(context.Background(), defaultParams())
	require.NoError(t, err)
	require.Len(t, breakdown, 1)
	assert.Equal(t, 0.0, breakdown[0].WinRate)
}

// --- Concurrent Aggregation ---

func TestConcurrentAggregation(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	org1 := seedOrg(t, db, "conc-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeSupport)
	_, board1c := seedSpaceAndBoard(t, db, org1, models.SpaceTypeCRM)

	org2 := seedOrg(t, db, "conc-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeSupport)
	_, board2c := seedSpaceAndBoard(t, db, org2, models.SpaceTypeCRM)

	now := time.Now()
	seedThread(t, db, board1, "ST1", `{"status":"open","priority":"high"}`, now)
	seedThread(t, db, board2, "ST2", `{"status":"resolved","priority":"low"}`, now.Add(-48*time.Hour))
	seedThread(t, db, board1c, "SL1", `{"stage":"new_lead","deal_value":10000}`, now)
	seedThread(t, db, board2c, "SL2", `{"stage":"closed_won","deal_value":20000}`, now)

	params := defaultParams()

	// Run concurrent calls to detect races.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := svc.GetAdminSupportMetrics(context.Background(), params)
			assert.NoError(t, err)
		}()
		go func() {
			defer wg.Done()
			_, err := svc.GetAdminSalesMetrics(context.Background(), params)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

// --- Admin Service Tests ---

func TestAdminSupportMetrics_MergesBreakdown(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	org1 := seedOrg(t, db, "admin-supp-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeSupport)

	org2 := seedOrg(t, db, "admin-supp-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeSupport)

	now := time.Now()
	seedThread(t, db, board1, "T1", `{"status":"open"}`, now)
	seedThread(t, db, board1, "T2", `{"status":"resolved"}`, now)
	seedThread(t, db, board2, "T3", `{"status":"open"}`, now)

	metrics, err := svc.GetAdminSupportMetrics(context.Background(), defaultParams())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.Len(t, metrics.OrgBreakdown, 2)

	// Platform totals.
	totalStatus := int64(0)
	for _, v := range metrics.StatusBreakdown {
		totalStatus += v
	}
	assert.Equal(t, int64(3), totalStatus)

	// Breakdown sums should match platform totals.
	breakdownTotal := int64(0)
	for _, b := range metrics.OrgBreakdown {
		breakdownTotal += b.TotalInRange
	}
	assert.Equal(t, totalStatus, breakdownTotal)
}

func TestAdminSalesMetrics_MergesBreakdown(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	org1 := seedOrg(t, db, "admin-sales-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeCRM)

	org2 := seedOrg(t, db, "admin-sales-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeCRM)

	now := time.Now()
	seedThread(t, db, board1, "L1", `{"stage":"new_lead","deal_value":10000}`, now)
	seedThread(t, db, board2, "L2", `{"stage":"closed_won","deal_value":20000}`, now)

	metrics, err := svc.GetAdminSalesMetrics(context.Background(), defaultParams())
	require.NoError(t, err)
	require.NotNil(t, metrics)

	assert.Len(t, metrics.OrgBreakdown, 2)

	// Platform funnel total should match sum of breakdown.
	platformTotal := int64(0)
	for _, s := range metrics.PipelineFunnel {
		platformTotal += s.Count
	}
	breakdownTotal := int64(0)
	for _, b := range metrics.OrgBreakdown {
		breakdownTotal += b.TotalLeads
	}
	assert.Equal(t, platformTotal, breakdownTotal)
}

// --- Admin Handler Tests ---

func TestHandler_GetAdminSupportMetrics(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "hnd-admin-supp")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeSupport)
	now := time.Now()

	seedThread(t, db, boardID, "T1", `{"status":"open","priority":"high"}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/support", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSupportMetrics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var metrics AdminSupportMetrics
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics.StatusBreakdown)
	assert.NotEmpty(t, metrics.OrgBreakdown)
}

func TestHandler_GetAdminSupportMetrics_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/support?from=bad", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSupportMetrics(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_GetAdminSalesMetrics(t *testing.T) {
	db := testDB(t)
	orgID := seedOrg(t, db, "hnd-admin-sales")
	_, boardID := seedSpaceAndBoard(t, db, orgID, models.SpaceTypeCRM)
	now := time.Now()

	seedThread(t, db, boardID, "L1", `{"stage":"new_lead","deal_value":10000}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/sales", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSalesMetrics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var metrics AdminSalesMetrics
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics.PipelineFunnel)
	assert.NotEmpty(t, metrics.OrgBreakdown)
}

func TestHandler_GetAdminSalesMetrics_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/sales?to=invalid", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSalesMetrics(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_GetAdminSupportExport(t *testing.T) {
	db := testDB(t)

	org1 := seedOrg(t, db, "hnd-adm-exp-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeSupport)

	org2 := seedOrg(t, db, "hnd-adm-exp-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeSupport)

	now := time.Now()
	seedThread(t, db, board1, "T1", `{"status":"open","priority":"high"}`, now)
	seedThread(t, db, board2, "T2", `{"status":"resolved","priority":"low"}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/support/export", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSupportExport(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/csv", rr.Header().Get("Content-Type"))

	reader := csv.NewReader(strings.NewReader(rr.Body.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Greater(t, len(records), 1)
	assert.Equal(t, "org_id", records[0][0])
	assert.Equal(t, "org_slug", records[0][1])
	assert.Equal(t, "id", records[0][2])
}

func TestHandler_GetAdminSupportExport_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/support/export?from=invalid", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSupportExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandler_GetAdminSalesExport(t *testing.T) {
	db := testDB(t)

	org1 := seedOrg(t, db, "hnd-adm-sexp-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeCRM)

	now := time.Now()
	seedThread(t, db, board1, "L1", `{"stage":"new_lead","deal_value":5000}`, now)

	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/sales/export", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSalesExport(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/csv", rr.Header().Get("Content-Type"))

	reader := csv.NewReader(strings.NewReader(rr.Body.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	assert.Greater(t, len(records), 1)
	assert.Equal(t, []string{"org_id", "org_slug", "id", "title", "stage", "assigned_to", "deal_value", "score", "created_at"}, records[0])
}

func TestHandler_GetAdminSalesExport_BadDate(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc, db)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/reports/sales/export?to=invalid", nil)
	rr := httptest.NewRecorder()
	h.GetAdminSalesExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// --- Admin Export Row Tests ---

func TestGetAdminSupportExportRows_MultiOrg(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db).(*repository)

	org1 := seedOrg(t, db, "aexp-supp-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeSupport)

	org2 := seedOrg(t, db, "aexp-supp-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeSupport)

	now := time.Now()
	seedThread(t, db, board1, "T1", `{"status":"open"}`, now)
	seedThread(t, db, board2, "T2", `{"status":"resolved"}`, now)

	rows, err := repo.GetAdminSupportExportRows(context.Background(), defaultParams())
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	// Verify org context is present.
	orgIDs := make(map[string]bool)
	for _, r := range rows {
		assert.NotEmpty(t, r.OrgID)
		assert.NotEmpty(t, r.OrgSlug)
		orgIDs[r.OrgID] = true
	}
	assert.True(t, orgIDs[org1])
	assert.True(t, orgIDs[org2])
}

func TestGetAdminSalesExportRows_MultiOrg(t *testing.T) {
	db := testDB(t)
	repo := NewRepository(db).(*repository)

	org1 := seedOrg(t, db, "aexp-sales-1")
	_, board1 := seedSpaceAndBoard(t, db, org1, models.SpaceTypeCRM)

	org2 := seedOrg(t, db, "aexp-sales-2")
	_, board2 := seedSpaceAndBoard(t, db, org2, models.SpaceTypeCRM)

	now := time.Now()
	seedThread(t, db, board1, "L1", `{"stage":"new_lead","deal_value":10000}`, now)
	seedThread(t, db, board2, "L2", `{"stage":"closed_won","deal_value":20000}`, now)

	rows, err := repo.GetAdminSalesExportRows(context.Background(), defaultParams())
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	for _, r := range rows {
		assert.NotEmpty(t, r.OrgID)
		assert.NotEmpty(t, r.OrgSlug)
	}
}

// --- Fuzz Tests ---

// FuzzAdminDateParams reuses the same fuzz strategy as Phase 1 but targets admin handlers.
func FuzzAdminDateParams(f *testing.F) {
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
