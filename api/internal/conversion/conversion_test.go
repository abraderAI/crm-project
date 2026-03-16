package conversion_test

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

	"github.com/abraderAI/crm-project/api/internal/audit"
	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/conversion"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/seed"
)

// testDB creates a fresh in-memory SQLite DB with migrations and seeds applied.
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
	require.NoError(t, seed.Run(db))
	return db
}

func withAuth(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{
		UserID:     userID,
		AuthMethod: auth.AuthMethodJWT,
	})
	return r.WithContext(ctx)
}

func newTestRouter(db *gorm.DB) *chi.Mux {
	svc := conversion.NewService(db)
	auditSvc := audit.NewService(db)
	h := conversion.NewHandler(svc, auditSvc)

	r := chi.NewRouter()
	r.Post("/me/upgrade", h.SelfServiceUpgrade)
	r.Post("/admin/leads/{lead_id}/convert", h.SalesConvert)
	r.Post("/admin/users/{user_id}/promote", h.AdminPromote)
	return r
}

// --- Service-level tests ---

func TestSelfServiceUpgrade_Success(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	shadow := &models.UserShadow{ClerkUserID: "user-upgrade", Email: "upgrade@example.com", DisplayName: "Upgrader"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.SelfServiceUpgrade(context.Background(), "user-upgrade", "My Org")
	require.NoError(t, err)
	assert.Equal(t, "converted", result.Status)
	assert.Equal(t, 3, result.Tier)
	assert.Equal(t, "My Org", result.Org.Name)
	assert.NotEmpty(t, result.Org.ID)

	// Verify org membership was created.
	var mem models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", result.Org.ID, "user-upgrade").First(&mem).Error)
	assert.Equal(t, models.RoleOwner, mem.Role)
}

func TestSelfServiceUpgrade_UpdatesLeadStatus(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	shadow := &models.UserShadow{ClerkUserID: "user-lead-up", Email: "lead@example.com", DisplayName: "LeadUser"}
	require.NoError(t, db.Create(shadow).Error)

	userID := "user-lead-up"
	lead := &models.Lead{Email: "lead@example.com", Status: models.LeadStatusRegistered, UserID: &userID}
	require.NoError(t, db.Create(lead).Error)

	_, err := svc.SelfServiceUpgrade(context.Background(), "user-lead-up", "Lead Org")
	require.NoError(t, err)

	// Verify lead was updated.
	var updated models.Lead
	require.NoError(t, db.First(&updated, "id = ?", lead.ID).Error)
	assert.Equal(t, models.LeadStatusConverted, updated.Status)
}

func TestSelfServiceUpgrade_EmptyUserID(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.SelfServiceUpgrade(context.Background(), "", "Org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user_id")
}

func TestSelfServiceUpgrade_EmptyOrgName(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.SelfServiceUpgrade(context.Background(), "user-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org_name")
}

func TestSelfServiceUpgrade_UserNotFound(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.SelfServiceUpgrade(context.Background(), "nonexistent", "Org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestSelfServiceUpgrade_AlreadyInOrg(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	shadow := &models.UserShadow{ClerkUserID: "user-has-org", Email: "hasorg@example.com", DisplayName: "HasOrg"}
	require.NoError(t, db.Create(shadow).Error)
	org := &models.Org{Name: "Existing", Slug: "existing-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	mem := &models.OrgMembership{OrgID: org.ID, UserID: "user-has-org", Role: models.RoleViewer}
	require.NoError(t, db.Create(mem).Error)

	_, err := svc.SelfServiceUpgrade(context.Background(), "user-has-org", "New Org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already belongs")
}

func TestSalesConvert_Success(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	userID := "user-sales-lead"
	shadow := &models.UserShadow{ClerkUserID: userID, Email: "salesl@example.com", DisplayName: "SalesLead"}
	require.NoError(t, db.Create(shadow).Error)

	lead := &models.Lead{Email: "salesl@example.com", Status: models.LeadStatusRegistered, UserID: &userID}
	require.NoError(t, db.Create(lead).Error)

	result, err := svc.SalesConvert(context.Background(), lead.ID, "Converted Corp", "deft-sales-user")
	require.NoError(t, err)
	assert.Equal(t, "converted", result.Status)
	assert.Equal(t, lead.ID, result.LeadID)
	assert.NotEmpty(t, result.Org.ID)

	// Verify user was added as owner.
	var mem models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", result.Org.ID, userID).First(&mem).Error)
	assert.Equal(t, models.RoleOwner, mem.Role)

	// Verify lead status updated.
	var updated models.Lead
	require.NoError(t, db.First(&updated, "id = ?", lead.ID).Error)
	assert.Equal(t, models.LeadStatusConverted, updated.Status)
}

func TestSalesConvert_LeadWithNoUser(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	lead := &models.Lead{Email: "anon@example.com", Status: models.LeadStatusAnonymous}
	require.NoError(t, db.Create(lead).Error)

	result, err := svc.SalesConvert(context.Background(), lead.ID, "Anon Corp", "deft-user")
	require.NoError(t, err)
	assert.Equal(t, "converted", result.Status)

	// No membership should be created (no user linked).
	var count int64
	db.Model(&models.OrgMembership{}).Where("org_id = ?", result.Org.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestSalesConvert_LeadNotFound(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.SalesConvert(context.Background(), "nonexistent-lead", "Org", "actor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lead not found")
}

func TestSalesConvert_AlreadyConverted(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	lead := &models.Lead{Email: "conv@example.com", Status: models.LeadStatusConverted}
	require.NoError(t, db.Create(lead).Error)

	_, err := svc.SalesConvert(context.Background(), lead.ID, "Org", "actor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already converted")
}

func TestSalesConvert_EmptyLeadID(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.SalesConvert(context.Background(), "", "Org", "actor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lead_id")
}

func TestSalesConvert_EmptyOrgName(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.SalesConvert(context.Background(), "some-id", "", "actor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org_name")
}

func TestAdminPromote_Success(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	shadow := &models.UserShadow{ClerkUserID: "user-promote", Email: "promote@example.com", DisplayName: "Promotee"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.AdminPromote(context.Background(), "user-promote", "Promoted Corp")
	require.NoError(t, err)
	assert.Equal(t, "promoted", result.Status)
	assert.Equal(t, 3, result.Tier)
	assert.Equal(t, "user-promote", result.UserID)
	assert.NotEmpty(t, result.Org.ID)

	// Verify membership.
	var mem models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", result.Org.ID, "user-promote").First(&mem).Error)
	assert.Equal(t, models.RoleOwner, mem.Role)
}

func TestAdminPromote_UserNotFound(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.AdminPromote(context.Background(), "ghost-user", "Org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAdminPromote_EmptyUserID(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.AdminPromote(context.Background(), "", "Org")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user_id")
}

func TestAdminPromote_EmptyOrgName(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	_, err := svc.AdminPromote(context.Background(), "user-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org_name")
}

func TestAdminPromote_UpdatesLeadStatus(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	shadow := &models.UserShadow{ClerkUserID: "user-prom-lead", Email: "promlead@example.com", DisplayName: "PromLead"}
	require.NoError(t, db.Create(shadow).Error)

	userID := "user-prom-lead"
	lead := &models.Lead{Email: "promlead@example.com", Status: models.LeadStatusRegistered, UserID: &userID}
	require.NoError(t, db.Create(lead).Error)

	_, err := svc.AdminPromote(context.Background(), "user-prom-lead", "Promoted Org")
	require.NoError(t, err)

	var updated models.Lead
	require.NoError(t, db.First(&updated, "id = ?", lead.ID).Error)
	assert.Equal(t, models.LeadStatusConverted, updated.Status)
}

func TestIsDeftOrgMember_True(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)

	mem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "deft-check-user", Role: models.RoleContributor}
	require.NoError(t, db.Create(mem).Error)

	ok, err := svc.IsDeftOrgMember(context.Background(), "deft-check-user")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestIsDeftOrgMember_False(t *testing.T) {
	db := testDB(t)
	svc := conversion.NewService(db)

	ok, err := svc.IsDeftOrgMember(context.Background(), "random-user")
	require.NoError(t, err)
	assert.False(t, ok)
}

// --- HTTP Handler tests ---

func TestHandler_SelfServiceUpgrade_Success(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	shadow := &models.UserShadow{ClerkUserID: "h-upgrade", Email: "hupgrade@example.com", DisplayName: "H"}
	require.NoError(t, db.Create(shadow).Error)

	body := `{"org_name":"Handler Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/me/upgrade", bytes.NewBufferString(body)), "h-upgrade")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result conversion.SelfServiceResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "converted", result.Status)
	assert.Equal(t, 3, result.Tier)
}

func TestHandler_SelfServiceUpgrade_Unauthenticated(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"org_name":"Org"}`
	req := httptest.NewRequest(http.MethodPost, "/me/upgrade", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SelfServiceUpgrade_EmptyOrgName(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	shadow := &models.UserShadow{ClerkUserID: "h-empty-org", Email: "empty@example.com", DisplayName: "E"}
	require.NoError(t, db.Create(shadow).Error)

	body := `{"org_name":""}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/me/upgrade", bytes.NewBufferString(body)), "h-empty-org")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SelfServiceUpgrade_AlreadyInOrg(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	shadow := &models.UserShadow{ClerkUserID: "h-has-org", Email: "hasorg@ex.com", DisplayName: "O"}
	require.NoError(t, db.Create(shadow).Error)
	org := &models.Org{Name: "Existing", Slug: "existing-h", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "h-has-org", Role: models.RoleViewer}).Error)

	body := `{"org_name":"New Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/me/upgrade", bytes.NewBufferString(body)), "h-has-org")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SalesConvert_Success(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	// Add actor as DEFT member.
	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: deftOrg.ID, UserID: "deft-sales", Role: models.RoleContributor}).Error)

	lead := &models.Lead{Email: "lead@example.com", Status: models.LeadStatusRegistered}
	require.NoError(t, db.Create(lead).Error)

	body := `{"org_name":"Converted Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/leads/"+lead.ID+"/convert", bytes.NewBufferString(body)), "deft-sales")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result conversion.SalesConvertResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "converted", result.Status)
}

func TestHandler_SalesConvert_NonDeftUserForbidden(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	lead := &models.Lead{Email: "forb@example.com", Status: models.LeadStatusRegistered}
	require.NoError(t, db.Create(lead).Error)

	body := `{"org_name":"Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/leads/"+lead.ID+"/convert", bytes.NewBufferString(body)), "random-user")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_SalesConvert_LeadNotFound(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: deftOrg.ID, UserID: "deft-sc-nf", Role: models.RoleContributor}).Error)

	body := `{"org_name":"Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/leads/nonexistent/convert", bytes.NewBufferString(body)), "deft-sc-nf")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_AdminPromote_Success(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	shadow := &models.UserShadow{ClerkUserID: "h-prom", Email: "hprom@example.com", DisplayName: "P"}
	require.NoError(t, db.Create(shadow).Error)

	body := `{"org_name":"Admin Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/users/h-prom/promote", bytes.NewBufferString(body)), "admin-actor")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result conversion.AdminPromoteResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, "promoted", result.Status)
	assert.Equal(t, 3, result.Tier)
	assert.Equal(t, "h-prom", result.UserID)
}

func TestHandler_AdminPromote_UserNotFound(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"org_name":"Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/users/ghost/promote", bytes.NewBufferString(body)), "admin")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_AdminPromote_Unauthenticated(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"org_name":"Org"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/users/someone/promote", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AdminPromote_EmptyOrgName(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"org_name":""}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/users/someone/promote", bytes.NewBufferString(body)), "admin")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SelfServiceUpgrade_InvalidJSON(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	req := withAuth(httptest.NewRequest(http.MethodPost, "/me/upgrade", bytes.NewBufferString("{bad}")), "user-1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SalesConvert_InvalidJSON(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: deftOrg.ID, UserID: "deft-bad-json", Role: models.RoleContributor}).Error)

	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/leads/some-id/convert", bytes.NewBufferString("{bad}")), "deft-bad-json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AdminPromote_InvalidJSON(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/users/u/promote", bytes.NewBufferString("{bad}")), "admin")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SalesConvert_Unauthenticated(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"org_name":"Org"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/leads/some/convert", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_SalesConvert_EmptyOrgName(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: deftOrg.ID, UserID: "deft-empty-org", Role: models.RoleContributor}).Error)

	body := `{"org_name":""}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/leads/some-id/convert", bytes.NewBufferString(body)), "deft-empty-org")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SalesConvert_AlreadyConverted(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: deftOrg.ID, UserID: "deft-dup-conv", Role: models.RoleContributor}).Error)

	lead := &models.Lead{Email: "dup@example.com", Status: models.LeadStatusConverted}
	require.NoError(t, db.Create(lead).Error)

	body := `{"org_name":"Org"}`
	req := withAuth(httptest.NewRequest(http.MethodPost, "/admin/leads/"+lead.ID+"/convert", bytes.NewBufferString(body)), "deft-dup-conv")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
