package upload

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/abraderAI/crm-project/api/internal/database"
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
	sqlDB, _ := db.DB()
	_, _ = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, database.Migrate(db))
	return db
}

func TestNewLocalStorage(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)
	assert.NotNil(t, storage)
}

func TestLocalStorage_StoreGetDelete(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)

	content := "Hello, World!"
	path, err := storage.Store("test.txt", strings.NewReader(content))
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// Get.
	reader, err := storage.Get(path)
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	require.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Delete.
	err = storage.Delete(path)
	require.NoError(t, err)

	// Get after delete should fail.
	_, err = storage.Get(path)
	assert.Error(t, err)
}

func TestLocalStorage_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)

	_, err = storage.Get("../../etc/passwd")
	assert.Error(t, err)

	err = storage.Delete("../../etc/passwd")
	assert.Error(t, err)
}

func TestValidateContentType(t *testing.T) {
	assert.True(t, ValidateContentType("image/png"))
	assert.True(t, ValidateContentType("application/json"))
	assert.True(t, ValidateContentType("text/plain; charset=utf-8"))
	assert.False(t, ValidateContentType("application/x-executable"))
	assert.False(t, ValidateContentType("video/mp4"))
}

func TestIsSubpath(t *testing.T) {
	assert.True(t, isSubpath("/a/b", "/a/b/c"))
	assert.False(t, isSubpath("/a/b", "/a/c"))
	assert.False(t, isSubpath("/a/b", "/a/b/../c"))
}

func TestService_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)

	// Create an org for FK.
	org := &models.Org{Name: "Upload Org", Slug: "upload-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db, storage, 104857600)

	// Write a temp file to simulate upload.
	tmpFile := filepath.Join(dir, "test-upload.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test file content"), 0o644))
	f, err := os.Open(tmpFile)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	upload, err := svc.Create(context.Background(), org.ID, "thread", "t1", "user1", "test-upload.txt", 17, f)
	require.NoError(t, err)
	assert.NotEmpty(t, upload.ID)
	assert.Equal(t, "test-upload.txt", upload.Filename)
	assert.Equal(t, int64(17), upload.Size)

	// Get.
	got, err := svc.Get(context.Background(), upload.ID)
	require.NoError(t, err)
	assert.Equal(t, upload.ID, got.ID)

	// Not found.
	got, err = svc.Get(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestService_Delete(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)

	org := &models.Org{Name: "Del Org", Slug: "del-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db, storage, 104857600)

	tmpFile := filepath.Join(dir, "del.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("delete me"), 0o644))
	f, _ := os.Open(tmpFile)
	defer func() { _ = f.Close() }()

	upload, err := svc.Create(context.Background(), org.ID, "thread", "t1", "user1", "del.txt", 9, f)
	require.NoError(t, err)

	err = svc.Delete(context.Background(), upload.ID)
	require.NoError(t, err)

	// Verify soft deleted.
	got, err := svc.Get(context.Background(), upload.ID)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Delete not found.
	err = svc.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestService_FileSizeLimit(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)

	org := &models.Org{Name: "Limit Org", Slug: "limit-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db, storage, 10) // 10 bytes max

	tmpFile := filepath.Join(dir, "big.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("this is more than ten bytes"), 0o644))
	f, _ := os.Open(tmpFile)
	defer func() { _ = f.Close() }()

	_, err = svc.Create(context.Background(), org.ID, "thread", "t1", "user1", "big.txt", 27, f)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestService_EmptyOrgID(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)

	tmpFile := filepath.Join(dir, "empty.txt")
	_ = os.WriteFile(tmpFile, []byte("x"), 0o644)
	f, _ := os.Open(tmpFile)
	defer func() { _ = f.Close() }()

	_, err := svc.Create(context.Background(), "", "thread", "t1", "user1", "empty.txt", 1, f)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org_id is required")
}

func TestService_EmptyFilename(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)

	tmpFile := filepath.Join(dir, "noname.txt")
	_ = os.WriteFile(tmpFile, []byte("x"), 0o644)
	f, _ := os.Open(tmpFile)
	defer func() { _ = f.Close() }()

	_, err := svc.Create(context.Background(), "org1", "thread", "t1", "user1", "", 1, f)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "filename is required")
}

func TestService_GetFile(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, err := NewLocalStorage(filepath.Join(dir, "uploads"))
	require.NoError(t, err)

	org := &models.Org{Name: "GF Org", Slug: "gf-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	svc := NewService(db, storage, 104857600)

	tmpFile := filepath.Join(dir, "getfile.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("file content"), 0o644))
	f, _ := os.Open(tmpFile)
	defer func() { _ = f.Close() }()

	upload, err := svc.Create(context.Background(), org.ID, "org", org.ID, "user1", "getfile.txt", 12, f)
	require.NoError(t, err)

	reader, err := svc.GetFile(upload.StoragePath)
	require.NoError(t, err)
	data, _ := io.ReadAll(reader)
	_ = reader.Close()
	assert.Equal(t, "file content", string(data))
}

func TestHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)
	h := NewHandler(svc, 104857600)

	org := &models.Org{Name: "H Org", Slug: "h-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	// Build multipart form.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("org_id", org.ID)
	_ = writer.WriteField("entity_type", "org")
	_ = writer.WriteField("entity_id", org.ID)
	part, _ := writer.CreateFormFile("file", "handler-test.txt")
	_, _ = part.Write([]byte("handler test content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/uploads", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_Create_MissingFile(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)
	h := NewHandler(svc, 104857600)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("org_id", "org1")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/uploads", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	h.Create(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)
	h := NewHandler(svc, 104857600)

	org := &models.Org{Name: "HG Org", Slug: "hg-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	tmpFile := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(tmpFile, []byte("test"), 0o644)
	f, _ := os.Open(tmpFile)
	up, _ := svc.Create(context.Background(), org.ID, "org", org.ID, "u1", "test.txt", 4, f)
	_ = f.Close()

	// Use chi context for URL params.
	req := httptest.NewRequest(http.MethodGet, "/v1/uploads/"+up.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", up.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)
	h := NewHandler(svc, 104857600)

	req := httptest.NewRequest(http.MethodGet, "/v1/uploads/nonexistent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Get(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Download(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)
	h := NewHandler(svc, 104857600)

	org := &models.Org{Name: "DL Org", Slug: "dl-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	tmpFile := filepath.Join(dir, "dl.txt")
	_ = os.WriteFile(tmpFile, []byte("download me"), 0o644)
	f, _ := os.Open(tmpFile)
	up, _ := svc.Create(context.Background(), org.ID, "org", org.ID, "u1", "dl.txt", 11, f)
	_ = f.Close()

	req := httptest.NewRequest(http.MethodGet, "/v1/uploads/"+up.ID+"/download", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", up.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Download(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "dl.txt")
	assert.Equal(t, "download me", w.Body.String())
}

func TestHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	storage, _ := NewLocalStorage(filepath.Join(dir, "uploads"))
	svc := NewService(db, storage, 104857600)
	h := NewHandler(svc, 104857600)

	org := &models.Org{Name: "HDel Org", Slug: "hdel-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	tmpFile := filepath.Join(dir, "hdel.txt")
	_ = os.WriteFile(tmpFile, []byte("delete"), 0o644)
	f, _ := os.Open(tmpFile)
	up, _ := svc.Create(context.Background(), org.ID, "org", org.ID, "u1", "hdel.txt", 6, f)
	_ = f.Close()

	req := httptest.NewRequest(http.MethodDelete, "/v1/uploads/"+up.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", up.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Delete not found.
	req = httptest.NewRequest(http.MethodDelete, "/v1/uploads/nonexistent", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("id", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w = httptest.NewRecorder()
	h.Delete(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
