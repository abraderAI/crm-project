package reporting_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
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
	"github.com/abraderAI/crm-project/api/internal/reporting"
	"github.com/abraderAI/crm-project/api/internal/server"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// --- Test helpers (duplicated for external test package) ---

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
	require.NoError(t, db.Exec("UPDATE threads SET created_at = ? WHERE id = ?", createdAt.Format("2006-01-02 15:04:05.999"), thread.ID).Error)
	require.NoError(t, db.First(thread, "id = ?", thread.ID).Error)
	return thread
}

func createMessage(t *testing.T, db *gorm.DB, threadID, authorID string, createdAt time.Time) {
	t.Helper()
	msg := &models.Message{
		ThreadID: threadID,
		Body:     "reply",
		AuthorID: authorID,
		Metadata: "{}",
	}
	require.NoError(t, db.Create(msg).Error)
	require.NoError(t, db.Exec("UPDATE messages SET created_at = ? WHERE id = ?", createdAt.Format("2006-01-02 15:04:05.999"), msg.ID).Error)
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

// seedSupportData creates a fully seeded support data set for testing.
func seedSupportData(t *testing.T, db *gorm.DB) string {
	t.Helper()

	org := createTestOrg(t, db, "report-org")
	space := createSupportSpace(t, db, org.ID)
	board := createBoard(t, db, space.ID)

	now := time.Now()
	day1 := now.Add(-60 * time.Hour) // 2.5 days ago — within 72h, not overdue
	day2 := now.Add(-36 * time.Hour) // 1.5 days ago
	day3 := now.Add(-12 * time.Hour) // 0.5 days ago

	t1 := createThread(t, db, board.ID, "Ticket 1", "author-1",
		`{"status":"open","priority":"high","assigned_to":"user-a"}`, day1)
	createMessage(t, db, t1.ID, "agent-1", day1.Add(2*time.Hour))

	createThread(t, db, board.ID, "Ticket 2", "author-2",
		`{"status":"open","priority":"medium","assigned_to":"user-a"}`, day1)

	t3 := createThread(t, db, board.ID, "Ticket 3", "author-3",
		`{"status":"in_progress","priority":"high","assigned_to":"user-b"}`, day2)
	createMessage(t, db, t3.ID, "agent-2", day2.Add(4*time.Hour))

	t4 := createThread(t, db, board.ID, "Ticket 4", "author-4",
		`{"status":"resolved","priority":"low","assigned_to":"user-a"}`, day2)
	require.NoError(t, db.Exec("UPDATE threads SET updated_at = datetime(created_at, '+10 hours') WHERE id = ?", t4.ID).Error)

	t5 := createThread(t, db, board.ID, "Ticket 5", "author-5",
		`{"status":"closed","priority":"medium","assigned_to":"user-b"}`, day3)
	require.NoError(t, db.Exec("UPDATE threads SET updated_at = datetime(created_at, '+6 hours') WHERE id = ?", t5.ID).Error)

	createThread(t, db, board.ID, "Ticket 6", "author-6",
		`{"status":"open"}`, now.AddDate(0, 0, -5))

	createThread(t, db, board.ID, "Ticket 7", "author-7",
		`{"status":"open","priority":"high","assigned_to":"user-a"}`, day3)

	createThread(t, db, board.ID, "Ticket 8", "author-8",
		`{"status":"in_progress","priority":"low","assigned_to":"user-b"}`, now.AddDate(0, 0, -10))

	t9 := createThread(t, db, board.ID, "Ticket 9", "author-9",
		`{"status":"resolved","priority":"high"}`, day3)
	require.NoError(t, db.Exec("UPDATE threads SET updated_at = datetime(created_at, '+8 hours') WHERE id = ?", t9.ID).Error)

	t10 := createThread(t, db, board.ID, "Ticket 10", "author-10",
		`{"status":"open","priority":"medium","assigned_to":"user-a"}`, day3)
	createMessage(t, db, t10.ID, "agent-3", day3.Add(1*time.Hour))

	return org.ID
}

// --- Live API tests ---

type liveAuthEnv struct {
	BaseURL   string
	IssuerURL string
	DB        *gorm.DB
	SignToken func(claims auth.JWTClaims) string
	Cleanup   func()
}

func liveAuthServer(t *testing.T) *liveAuthEnv {
	t.Helper()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubKey := &privKey.PublicKey
	kid := "reporting-test-kid"

	nB64 := base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes())
	eB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes())
	jwks := map[string]interface{}{
		"keys": []map[string]string{{
			"kid": kid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": nB64, "e": eB64,
		}},
	}
	jwksMux := http.NewServeMux()
	jwksMux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})
	jwksSrv := httptest.NewServer(jwksMux)

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

	testLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	router := server.NewRouter(server.Config{
		DB:          db,
		Logger:      testLogger,
		CORSOrigins: []string{"http://localhost:3000"},
		IssuerURL:   jwksSrv.URL,
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(listener) }()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())
	for i := 0; i < 50; i++ {
		if resp, err := http.Get(baseURL + "/healthz"); err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	signToken := func(claims auth.JWTClaims) string {
		header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
		headerJSON, _ := json.Marshal(header)
		claimsJSON, _ := json.Marshal(claims)
		hB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
		cB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
		input := hB64 + "." + cB64
		h := sha256.Sum256([]byte(input))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, h[:])
		return input + "." + base64.RawURLEncoding.EncodeToString(sig)
	}

	return &liveAuthEnv{
		BaseURL:   baseURL,
		IssuerURL: jwksSrv.URL,
		DB:        db,
		SignToken: signToken,
		Cleanup: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
			jwksSrv.Close()
			_, _ = sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
			_ = sqlDB.Close()
		},
	}
}

func TestLive_SupportMetrics_OK(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID := seedSupportData(t, env.DB)
	createAdminMembership(t, env.DB, orgID, "live-admin")

	token := env.SignToken(auth.JWTClaims{
		Subject:   "live-admin",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	to := time.Now().Format("2006-01-02")

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/orgs/"+orgID+"/reports/support?from="+from+"&to="+to, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var metrics reporting.SupportMetrics
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))

	// All 7 fields present.
	assert.NotEmpty(t, metrics.StatusBreakdown)
	assert.NotEmpty(t, metrics.VolumeOverTime)
	assert.NotNil(t, metrics.AvgResolutionHours)
	assert.NotEmpty(t, metrics.TicketsByAssignee)
	assert.NotEmpty(t, metrics.TicketsByPriority)
	assert.NotNil(t, metrics.AvgFirstResponseHours)

	// Status breakdown sums match total seeded count (10).
	total := int64(0)
	for _, v := range metrics.StatusBreakdown {
		total += v
	}
	assert.Equal(t, int64(10), total)
}

func TestLive_SupportExport_CSV(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	orgID := seedSupportData(t, env.DB)
	createAdminMembership(t, env.DB, orgID, "live-admin")

	token := env.SignToken(auth.JWTClaims{
		Subject:   "live-admin",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/orgs/"+orgID+"/reports/support/export", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// header + 10 rows
	assert.Equal(t, 11, len(records))
	assert.Equal(t, "id", records[0][0])
}

func TestLive_SupportMetrics_Forbidden_ViewerRole(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := createTestOrg(t, env.DB, "live-viewer-org")
	createViewerMembership(t, env.DB, org.ID, "live-viewer")

	token := env.SignToken(auth.JWTClaims{
		Subject:   "live-viewer",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/orgs/"+org.ID+"/reports/support", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestLive_SupportMetrics_BadDate_RFC7807(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := createTestOrg(t, env.DB, "live-bad-date")
	createAdminMembership(t, env.DB, org.ID, "live-admin")

	token := env.SignToken(auth.JWTClaims{
		Subject:   "live-admin",
		Issuer:    env.IssuerURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})

	req, err := http.NewRequest(http.MethodGet,
		env.BaseURL+"/v1/orgs/"+org.ID+"/reports/support?from=invalid", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/problem+json", resp.Header.Get("Content-Type"))

	var problem apierrors.ProblemDetail
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&problem))
	assert.Equal(t, 400, problem.Status)
}

func TestLive_SupportMetrics_Unauthenticated(t *testing.T) {
	env := liveAuthServer(t)
	defer env.Cleanup()

	org := createTestOrg(t, env.DB, "live-unauth")

	resp, err := http.Get(env.BaseURL + "/v1/orgs/" + org.ID + "/reports/support")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// --- Fuzz tests ---

func FuzzDateParams(f *testing.F) {
	f.Add("2026-03-01", "2026-03-15")
	f.Add("", "")
	f.Add("not-a-date", "2026-01-01")
	f.Add("2026-01-01", "garbage")
	f.Add("99999", "00-00-00")
	for i := 0; i < 50; i++ {
		f.Add(fmt.Sprintf("seed-%d", i), fmt.Sprintf("seed-%d", i+100))
	}

	f.Fuzz(func(t *testing.T, fromStr, toStr string) {
		db := setupTestDB(t)
		org := createTestOrg(t, db, "fuzz-org")
		createAdminMembership(t, db, org.ID, "fuzz-user")
		handler := reporting.NewHandler(reporting.NewService(reporting.NewRepository(db)), db)

		baseURL := "/v1/orgs/" + org.ID + "/reports/support"
		req := httptest.NewRequest(http.MethodGet, baseURL, nil)
		q := req.URL.Query()
		if fromStr != "" {
			q.Set("from", fromStr)
		}
		if toStr != "" {
			q.Set("to", toStr)
		}
		req.URL.RawQuery = q.Encode()
		req = withAuthAndChi(req, "fuzz-user", map[string]string{"org": org.ID})
		w := httptest.NewRecorder()

		handler.GetSupportMetrics(w, req)

		// Should not panic; response must be 200 or 400.
		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, w.Code)
	})
}

func FuzzAssigneeParam(f *testing.F) {
	f.Add("")
	f.Add("user-123")
	f.Add("user_with_special_chars")
	f.Add("user-sql-injection")
	f.Add("user-xss-attempt")
	for i := 0; i < 50; i++ {
		f.Add(fmt.Sprintf("fuzz-user-%d", i))
	}

	f.Fuzz(func(t *testing.T, assignee string) {
		db := setupTestDB(t)
		org := createTestOrg(t, db, "fuzz-assignee-org")
		createAdminMembership(t, db, org.ID, "fuzz-user")
		handler := reporting.NewHandler(reporting.NewService(reporting.NewRepository(db)), db)

		// Build URL safely using net/url to avoid malformed request panics.
		baseURL := "/v1/orgs/" + org.ID + "/reports/support"
		req := httptest.NewRequest(http.MethodGet, baseURL, nil)
		if assignee != "" {
			q := req.URL.Query()
			q.Set("assignee", assignee)
			req.URL.RawQuery = q.Encode()
		}
		req = withAuthAndChi(req, "fuzz-user", map[string]string{"org": org.ID})
		w := httptest.NewRecorder()

		handler.GetSupportMetrics(w, req)

		// Should not panic; response must be 200.
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
