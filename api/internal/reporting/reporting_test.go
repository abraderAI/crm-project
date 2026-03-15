package reporting

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// --- Test helpers ---

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

func createTestOrg(t *testing.T, db *gorm.DB, slug string) *models.Org {
	t.Helper()
	org := &models.Org{Name: slug, Slug: slug, Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	return org
}

func createSupportSpace(t *testing.T, db *gorm.DB, orgID string) *models.Space {
	t.Helper()
	space := &models.Space{
		OrgID:    orgID,
		Name:     "Support",
		Slug:     "support-" + orgID[:8],
		Type:     models.SpaceTypeSupport,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(space).Error)
	return space
}

func createBoard(t *testing.T, db *gorm.DB, spaceID string) *models.Board {
	t.Helper()
	board := &models.Board{
		SpaceID:  spaceID,
		Name:     "Tickets",
		Slug:     "tickets-" + spaceID[:8],
		Metadata: "{}",
	}
	require.NoError(t, db.Create(board).Error)
	return board
}

func createThread(t *testing.T, db *gorm.DB, boardID, title, authorID string, metadata string, createdAt time.Time) *models.Thread {
	t.Helper()
	uid, err := uuid.NewV7()
	require.NoError(t, err)
	thread := &models.Thread{
		BoardID:  boardID,
		Title:    title,
		Slug:     "t-" + uid.String(),
		AuthorID: authorID,
		Metadata: metadata,
	}
	require.NoError(t, db.Create(thread).Error)
	// Override created_at.
	require.NoError(t, db.Exec("UPDATE threads SET created_at = ? WHERE id = ?", createdAt, thread.ID).Error)
	// Re-read to get updated timestamps.
	require.NoError(t, db.First(thread, "id = ?", thread.ID).Error)
	return thread
}

func createMessage(t *testing.T, db *gorm.DB, threadID, authorID string, createdAt time.Time) *models.Message {
	t.Helper()
	msg := &models.Message{
		ThreadID: threadID,
		Body:     "reply",
		AuthorID: authorID,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(msg).Error)
	require.NoError(t, db.Exec("UPDATE messages SET created_at = ? WHERE id = ?", createdAt, msg.ID).Error)
	return msg
}

func createAdminMembership(t *testing.T, db *gorm.DB, orgID, userID string) {
	t.Helper()
	m := &models.OrgMembership{OrgID: orgID, UserID: userID, Role: models.RoleAdmin}
	require.NoError(t, db.Create(m).Error)
}

func createViewerMembership(t *testing.T, db *gorm.DB, orgID, userID string) {
	t.Helper()
	m := &models.OrgMembership{OrgID: orgID, UserID: userID, Role: models.RoleViewer}
	require.NoError(t, db.Create(m).Error)
}

func withAuthCtx(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: userID})
	return r.WithContext(ctx)
}

func withChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withAuthAndChi(r *http.Request, userID string, params map[string]string) *http.Request {
	return withChiParams(withAuthCtx(r, userID), params)
}

func defaultParams() ReportParams {
	return ReportParams{
		From: time.Now().UTC().AddDate(0, 0, -90),
		To:   time.Now().UTC().Add(24 * time.Hour),
	}
}

// seedSupportData creates a fully seeded support data set for testing.
// Returns orgID, boardID, and the number of threads created.
func seedSupportData(t *testing.T, db *gorm.DB) (string, string) {
	t.Helper()

	org := createTestOrg(t, db, "report-org")
	space := createSupportSpace(t, db, org.ID)
	board := createBoard(t, db, space.ID)

	now := time.Now().UTC()
	day1 := now.Add(-60 * time.Hour) // 2.5 days ago — within 72h, not overdue
	day2 := now.Add(-36 * time.Hour) // 1.5 days ago
	day3 := now.Add(-12 * time.Hour) // 0.5 days ago

	// Thread 1: open, high, assigned to user-a, day1
	t1 := createThread(t, db, board.ID, "Ticket 1", "author-1",
		`{"status":"open","priority":"high","assigned_to":"user-a"}`, day1)
	// Reply by someone else 2 hours later.
	createMessage(t, db, t1.ID, "agent-1", day1.Add(2*time.Hour))

	// Thread 2: open, medium, assigned to user-a, day1
	createThread(t, db, board.ID, "Ticket 2", "author-2",
		`{"status":"open","priority":"medium","assigned_to":"user-a"}`, day1)

	// Thread 3: in_progress, high, assigned to user-b, day2
	t3 := createThread(t, db, board.ID, "Ticket 3", "author-3",
		`{"status":"in_progress","priority":"high","assigned_to":"user-b"}`, day2)
	// Reply 4 hours later.
	createMessage(t, db, t3.ID, "agent-2", day2.Add(4*time.Hour))

	// Thread 4: resolved, low, assigned to user-a, day2
	t4 := createThread(t, db, board.ID, "Ticket 4", "author-4",
		`{"status":"resolved","priority":"low","assigned_to":"user-a"}`, day2)
	// Set updated_at to 10 hours after created for resolution time calculation.
	require.NoError(t, db.Exec("UPDATE threads SET updated_at = datetime(created_at, '+10 hours') WHERE id = ?", t4.ID).Error)

	// Thread 5: closed, medium, assigned to user-b, day3
	t5 := createThread(t, db, board.ID, "Ticket 5", "author-5",
		`{"status":"closed","priority":"medium","assigned_to":"user-b"}`, day3)
	require.NoError(t, db.Exec("UPDATE threads SET updated_at = datetime(created_at, '+6 hours') WHERE id = ?", t5.ID).Error)

	// Thread 6: open, no priority, unassigned, day1 (overdue — > 72h old)
	createThread(t, db, board.ID, "Ticket 6", "author-6",
		`{"status":"open"}`, now.AddDate(0, 0, -5))

	// Thread 7: open, high, assigned to user-a, day3
	createThread(t, db, board.ID, "Ticket 7", "author-7",
		`{"status":"open","priority":"high","assigned_to":"user-a"}`, day3)

	// Thread 8: in_progress, low, assigned to user-b, old (overdue)
	createThread(t, db, board.ID, "Ticket 8", "author-8",
		`{"status":"in_progress","priority":"low","assigned_to":"user-b"}`, now.AddDate(0, 0, -10))

	// Thread 9: resolved, high, no assignee, day3
	t9 := createThread(t, db, board.ID, "Ticket 9", "author-9",
		`{"status":"resolved","priority":"high"}`, day3)
	require.NoError(t, db.Exec("UPDATE threads SET updated_at = datetime(created_at, '+8 hours') WHERE id = ?", t9.ID).Error)

	// Thread 10: open, medium, assigned to user-a, day3
	t10 := createThread(t, db, board.ID, "Ticket 10", "author-10",
		`{"status":"open","priority":"medium","assigned_to":"user-a"}`, day3)
	// Reply by someone else 1 hour later.
	createMessage(t, db, t10.ID, "agent-3", day3.Add(1*time.Hour))

	return org.ID, board.ID
}

// --- Unit tests (Repository) ---

func TestGetStatusBreakdown(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetStatusBreakdown(context.Background(), orgID, defaultParams())
	require.NoError(t, err)

	// We have: 5 open, 2 in_progress, 2 resolved, 1 closed
	total := int64(0)
	for _, v := range result {
		total += v
	}
	assert.Equal(t, int64(10), total)
	assert.True(t, result["open"] > 0)
	assert.True(t, result["in_progress"] > 0)
	assert.True(t, result["resolved"] > 0)
	assert.True(t, result["closed"] > 0)
}

func TestGetVolumeOverTime(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetVolumeOverTime(context.Background(), orgID, defaultParams())
	require.NoError(t, err)

	// Should have entries across multiple days.
	assert.GreaterOrEqual(t, len(result), 2)

	total := int64(0)
	for _, dc := range result {
		total += dc.Count
		assert.NotEmpty(t, dc.Date)
	}
	assert.Equal(t, int64(10), total)
}

func TestGetAvgResolutionTime(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetAvgResolutionHours(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotNil(t, result, "expected non-nil avg for resolved/closed threads")

	// We set resolution times of 10h, 6h, 8h → avg = 8h.
	assert.InDelta(t, 8.0, *result, 0.5)
}

func TestGetTicketsByAssignee(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetTicketsByAssignee(context.Background(), orgID, defaultParams())
	require.NoError(t, err)

	// Should have user-a and user-b with open/in_progress tickets.
	assert.GreaterOrEqual(t, len(result), 2)
	found := map[string]int64{}
	for _, ac := range result {
		found[ac.UserID] = ac.Count
	}
	assert.True(t, found["user-a"] > 0)
	assert.True(t, found["user-b"] > 0)
}

func TestGetTicketsByPriority(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetTicketsByPriority(context.Background(), orgID, defaultParams())
	require.NoError(t, err)

	total := int64(0)
	for _, v := range result {
		total += v
	}
	assert.Equal(t, int64(10), total)
	assert.True(t, result["high"] > 0)
	assert.True(t, result["medium"] > 0)
	assert.True(t, result["low"] > 0)
}

func TestGetAvgFirstResponseTime(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetAvgFirstResponseHours(context.Background(), orgID, defaultParams())
	require.NoError(t, err)
	require.NotNil(t, result, "expected non-nil avg for threads with replies")

	// We have replies at 2h, 4h, 1h → avg ≈ 2.33h.
	assert.InDelta(t, 2.33, *result, 0.5)
}

func TestGetOverdueCount(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	result, err := repo.GetOverdueCount(context.Background(), orgID, defaultParams())
	require.NoError(t, err)

	// Threads older than 72h with open/in_progress status:
	// Thread 6 (5 days old, open), Thread 8 (10 days old, in_progress).
	assert.Equal(t, int64(2), result)
}

func TestAssigneeFilterApplied(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	params := defaultParams()
	params.Assignee = "user-a"

	breakdown, err := repo.GetStatusBreakdown(context.Background(), orgID, params)
	require.NoError(t, err)

	total := int64(0)
	for _, v := range breakdown {
		total += v
	}
	// user-a has: Ticket 1 (open), Ticket 2 (open), Ticket 4 (resolved), Ticket 7 (open), Ticket 10 (open) = 5
	assert.Equal(t, int64(5), total)
}

func TestEmptyOrgReturnsZeros(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "empty-org")
	repo := NewRepository(db)
	svc := NewService(repo)

	metrics, err := svc.GetSupportMetrics(context.Background(), org.ID, defaultParams())
	require.NoError(t, err)

	assert.Empty(t, metrics.StatusBreakdown)
	assert.Empty(t, metrics.VolumeOverTime)
	assert.Nil(t, metrics.AvgResolutionHours)
	assert.Empty(t, metrics.TicketsByAssignee)
	assert.Empty(t, metrics.TicketsByPriority)
	assert.Nil(t, metrics.AvgFirstResponseHours)
	assert.Equal(t, int64(0), metrics.OverdueCount)
}

// --- Handler unit tests ---

func TestHandler_GetSupportMetrics_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/test-org/reports/support", nil)
	req = withChiParams(req, map[string]string{"org": "test-org"})
	w := httptest.NewRecorder()

	handler.GetSupportMetrics(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetSupportMetrics_Forbidden(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "forbidden-org")
	createViewerMembership(t, db, org.ID, "viewer-user")
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/reports/support", nil)
	req = withAuthAndChi(req, "viewer-user", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	handler.GetSupportMetrics(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_GetSupportMetrics_InvalidDate(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "date-org")
	createAdminMembership(t, db, org.ID, "admin-user")
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/reports/support?from=invalid", nil)
	req = withAuthAndChi(req, "admin-user", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	handler.GetSupportMetrics(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(w.Body).Decode(&problem))
	assert.Equal(t, 400, problem.Status)
	assert.Contains(t, problem.Detail, "from")
}

func TestHandler_GetSupportMetrics_InvalidToDate(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "to-date-org")
	createAdminMembership(t, db, org.ID, "admin-user")
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/reports/support?to=bad-date", nil)
	req = withAuthAndChi(req, "admin-user", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	handler.GetSupportMetrics(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSupportMetrics_OK(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	createAdminMembership(t, db, orgID, "admin-user")
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/reports/support", nil)
	req = withAuthAndChi(req, "admin-user", map[string]string{"org": orgID})
	w := httptest.NewRecorder()

	handler.GetSupportMetrics(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var metrics SupportMetrics
	require.NoError(t, json.NewDecoder(w.Body).Decode(&metrics))
	assert.NotEmpty(t, metrics.StatusBreakdown)
	assert.NotEmpty(t, metrics.VolumeOverTime)
	assert.NotNil(t, metrics.AvgResolutionHours)
	assert.NotEmpty(t, metrics.TicketsByAssignee)
	assert.NotEmpty(t, metrics.TicketsByPriority)
	assert.NotNil(t, metrics.AvgFirstResponseHours)
	assert.True(t, metrics.OverdueCount > 0)
}

func TestHandler_GetSupportExport_CSV(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	createAdminMembership(t, db, orgID, "admin-user")
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+orgID+"/reports/support/export", nil)
	req = withAuthAndChi(req, "admin-user", map[string]string{"org": orgID})
	w := httptest.NewRecorder()

	handler.GetSupportExport(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/csv", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "support-report.csv")

	// Verify valid CSV.
	reader := csv.NewReader(strings.NewReader(w.Body.String()))
	records, err := reader.ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(records), 2) // header + at least 1 data row

	// Check header.
	assert.Equal(t, []string{"id", "title", "status", "priority", "assigned_to", "created_at", "updated_at"}, records[0])

	// Check at least 10 data rows.
	assert.Equal(t, 11, len(records)) // header + 10 threads
}

func TestHandler_GetSupportExport_Forbidden(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "export-forbidden-org")
	createViewerMembership(t, db, org.ID, "viewer-user")
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/reports/support/export", nil)
	req = withAuthAndChi(req, "viewer-user", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	handler.GetSupportExport(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_OwnerRoleAllowed(t *testing.T) {
	db := setupTestDB(t)
	org := createTestOrg(t, db, "owner-org")
	m := &models.OrgMembership{OrgID: org.ID, UserID: "owner-user", Role: models.RoleOwner}
	require.NoError(t, db.Create(m).Error)
	handler := NewHandler(NewService(NewRepository(db)))

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/reports/support", nil)
	req = withAuthAndChi(req, "owner-user", map[string]string{"org": org.ID})
	w := httptest.NewRecorder()

	handler.GetSupportMetrics(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestScanExportRows_Streaming(t *testing.T) {
	db := setupTestDB(t)
	orgID, _ := seedSupportData(t, db)
	repo := NewRepository(db)

	var rowCount int
	err := repo.ScanExportRows(context.Background(), orgID, defaultParams(), func(row ExportRow) error {
		rowCount++
		assert.NotEmpty(t, row.ID)
		assert.NotEmpty(t, row.Title)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 10, rowCount)
}
