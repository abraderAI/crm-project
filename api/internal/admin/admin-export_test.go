package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Mock Storage Provider ---

type mockStorage struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{files: make(map[string][]byte)}
}

func (m *mockStorage) Store(filename string, content io.Reader) (string, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.files[filename] = data
	m.mu.Unlock()
	return filename, nil
}

func (m *mockStorage) Get(storagePath string) (io.ReadCloser, error) {
	m.mu.Lock()
	data, ok := m.files[storagePath]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("file not found: %s", storagePath)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStorage) Delete(storagePath string) error {
	m.mu.Lock()
	delete(m.files, storagePath)
	m.mu.Unlock()
	return nil
}

// --- Export Service Tests ---

func TestExport_Lifecycle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	// Create test users.
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u2", Email: "b@test.com", DisplayName: "B", LastSeenAt: now, SyncedAt: now})

	// Create export (async processing).
	export, err := svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
	require.NoError(t, err)

	// Wait for async processing.
	time.Sleep(500 * time.Millisecond)

	// Re-read from DB to get updated status.
	fetched, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "completed", fetched.Status)
	assert.NotEmpty(t, fetched.FilePath)

	// Verify CSV content.
	reader, err := storage.Get(fetched.FilePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)
	csvStr := string(data)
	assert.Contains(t, csvStr, "clerk_user_id")
	assert.Contains(t, csvStr, "a@test.com")
	assert.Contains(t, csvStr, "b@test.com")
}

func TestExport_JSONFormat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})

	export, err := svc.CreateExport(ctx, "users", "json", "{}", "admin1", storage)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	fetched, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", fetched.Status)

	reader, err := storage.Get(fetched.FilePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)

	var records []map[string]any
	require.NoError(t, json.Unmarshal(data, &records))
	assert.Len(t, records, 1)
	assert.Equal(t, "a@test.com", records[0]["email"])
}

func TestExport_OrgType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.Org{Name: "Org1", Slug: "org1", Metadata: "{}"})
	db.Create(&models.Org{Name: "Org2", Slug: "org2", Metadata: "{}"})

	export, err := svc.CreateExport(ctx, "orgs", "csv", "{}", "admin1", storage)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	fetched, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", fetched.Status)

	reader, err := storage.Get(fetched.FilePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)
	assert.Contains(t, string(data), "org1")
	assert.Contains(t, string(data), "org2")
}

func TestExport_AuditType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.AuditLog{UserID: "admin1", Action: "ban", EntityType: "user", EntityID: "u1"})

	export, err := svc.CreateExport(ctx, "audit", "csv", "{}", "admin1", storage)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	fetched, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", fetched.Status)

	reader, err := storage.Get(fetched.FilePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)
	assert.Contains(t, string(data), "ban")
}

func TestExport_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	storage := newMockStorage()

	_, err := svc.CreateExport(context.Background(), "invalid", "csv", "{}", "admin1", storage)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid export type")
}

func TestExport_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	storage := newMockStorage()

	_, err := svc.CreateExport(context.Background(), "users", "xml", "{}", "admin1", storage)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid export format")
}

func TestExport_InvalidFilters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	storage := newMockStorage()

	_, err := svc.CreateExport(context.Background(), "users", "csv", "{invalid", "admin1", storage)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filters JSON")
}

func TestExport_EmptyRequestedBy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	storage := newMockStorage()

	_, err := svc.CreateExport(context.Background(), "users", "csv", "{}", "", storage)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requested_by is required")
}

func TestExport_EmptyFiltersDefaultsToEmptyJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	storage := newMockStorage()

	export, err := svc.CreateExport(context.Background(), "users", "csv", "", "admin1", storage)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	fetched, err := svc.GetExport(context.Background(), export.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", fetched.Status)
}

func TestExport_ListExports(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	_, _ = svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
	_, _ = svc.CreateExport(ctx, "orgs", "csv", "{}", "admin1", storage)
	_, _ = svc.CreateExport(ctx, "audit", "json", "{}", "admin1", storage)

	exports, pageInfo, err := svc.ListExports(ctx, "admin1", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, exports, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestExport_ListExports_FiltersRequestedBy(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	_, _ = svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
	_, _ = svc.CreateExport(ctx, "orgs", "csv", "{}", "admin2", storage)

	exports, _, err := svc.ListExports(ctx, "admin1", pagination.Params{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, exports, 1)
}

func TestExport_ListExports_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	for i := 0; i < 5; i++ {
		_, _ = svc.CreateExport(ctx, "users", "csv", "{}", "admin1", storage)
	}

	exports, pageInfo, err := svc.ListExports(ctx, "admin1", pagination.Params{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, exports, 3)
	assert.True(t, pageInfo.HasMore)
}

func TestExport_AuditCSV(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.AuditLog{UserID: "admin1", Action: "suspend", EntityType: "org", EntityID: "org-123"})
	db.Create(&models.AuditLog{UserID: "admin2", Action: "ban", EntityType: "user", EntityID: "user-456"})

	export, err := svc.CreateExport(ctx, "audit", "csv", "{}", "admin1", storage)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	fetched, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)

	reader, err := storage.Get(fetched.FilePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)
	csvStr := string(data)
	assert.Contains(t, csvStr, "suspend")
	assert.Contains(t, csvStr, "ban")
	assert.Contains(t, csvStr, "org-123")
	assert.Contains(t, csvStr, "user-456")
}

func TestExport_OrgsJSON(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()
	storage := newMockStorage()

	db.Create(&models.Org{Name: "Org1", Slug: "org1", Metadata: "{}"})
	db.Create(&models.Org{Name: "Org2", Slug: "org2", Metadata: "{}"})

	export, err := svc.CreateExport(ctx, "orgs", "json", "{}", "admin1", storage)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	fetched, err := svc.GetExport(ctx, export.ID)
	require.NoError(t, err)

	reader, err := storage.Get(fetched.FilePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)

	var records []map[string]any
	require.NoError(t, json.Unmarshal(data, &records))
	assert.Len(t, records, 2)
}

func TestExport_GetExport_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	export, err := svc.GetExport(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, export)
}

func TestExport_NilStorage(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	export, err := svc.CreateExport(context.Background(), "users", "csv", "{}", "admin1", nil)
	require.NoError(t, err)
	// Without storage, export is created as pending (no file stored).
	assert.Equal(t, "pending", export.Status)
	assert.Empty(t, export.FilePath)
}

// --- Export Handler Tests ---

func TestHandler_CreateExport(t *testing.T) {
	h, _ := setupTestHandler(t)
	h.SetStorage(newMockStorage())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"users","format":"csv"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.CreateExportHandler(w, r)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp["export_id"])
	assert.Equal(t, "pending", resp["status"])
}

func TestHandler_CreateExport_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports", strings.NewReader("invalid"))
	h.CreateExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateExport_MissingFields(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"users"}`))
	r.Header.Set("Content-Type", "application/json")
	h.CreateExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateExport_InvalidType(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"bad","format":"csv"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.CreateExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListExports(t *testing.T) {
	h, _ := setupTestHandler(t)
	h.SetStorage(newMockStorage())

	// Create an export first.
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodPost, "/v1/admin/exports",
		strings.NewReader(`{"type":"users","format":"csv"}`))
	r1.Header.Set("Content-Type", "application/json")
	r1 = r1.WithContext(auth.SetUserContext(r1.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.CreateExportHandler(w1, r1)

	// List exports.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports", nil)
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.ListExportsHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_GetExport(t *testing.T) {
	h, db := setupTestHandler(t)
	export := models.AdminExport{Type: "users", Format: "csv", Status: "completed", RequestedBy: "admin1"}
	db.Create(&export)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports/"+export.ID, nil)
	r = chiCtx(r, "id", export.ID)
	h.GetExportHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "completed")
}

func TestHandler_GetExport_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports/nonexistent", nil)
	r = chiCtx(r, "id", "nonexistent")
	h.GetExportHandler(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetExport_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/exports/", nil)
	r = chiCtx(r, "id", "")
	h.GetExportHandler(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- isExportValidationErr Tests ---

func TestIsExportValidationErr(t *testing.T) {
	assert.True(t, isExportValidationErr(fmt.Errorf("invalid export type: foo")))
	assert.True(t, isExportValidationErr(fmt.Errorf("invalid export format: xml")))
	assert.True(t, isExportValidationErr(fmt.Errorf("requested_by is required")))
	assert.True(t, isExportValidationErr(fmt.Errorf("invalid filters JSON")))
	assert.False(t, isExportValidationErr(fmt.Errorf("database error")))
}
