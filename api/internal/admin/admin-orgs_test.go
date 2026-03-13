package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/pagination"
)

// --- Org Management Service Tests ---

func TestService_ListOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		org := models.Org{Name: "Org " + string(rune('A'+i)), Slug: "org-" + string(rune('a'+i)), Metadata: "{}"}
		require.NoError(t, db.Create(&org).Error)
	}

	orgs, pageInfo, err := svc.ListOrgs(ctx, OrgListParams{Params: pagination.Params{Limit: 50}})
	require.NoError(t, err)
	assert.Len(t, orgs, 3)
	assert.False(t, pageInfo.HasMore)
}

func TestService_ListOrgs_Filters(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&models.Org{Name: "Alpha", Slug: "alpha", Metadata: "{}"}).Error)
	require.NoError(t, db.Create(&models.Org{Name: "Beta", Slug: "beta", Metadata: "{}"}).Error)

	orgs, _, err := svc.ListOrgs(ctx, OrgListParams{
		Params: pagination.Params{Limit: 50},
		Slug:   "alpha",
	})
	require.NoError(t, err)
	assert.Len(t, orgs, 1)

	orgs, _, err = svc.ListOrgs(ctx, OrgListParams{
		Params: pagination.Params{Limit: 50},
		Name:   "Beta",
	})
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
}

func TestService_GetOrgDetail(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	detail, err := svc.GetOrgDetail(ctx, org.Slug)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "Test Org", detail.Name)

	// Not found.
	detail, err = svc.GetOrgDetail(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, detail)
}

func TestService_GetOrgDetail_ByID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	detail, err := svc.GetOrgDetail(ctx, org.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "Test Org", detail.Name)
}

func TestService_SuspendUnsuspendOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	// Suspend.
	err := svc.SuspendOrg(ctx, org.Slug, "violation", "admin1")
	require.NoError(t, err)

	suspended, err := svc.IsOrgSuspended(ctx, org.Slug)
	require.NoError(t, err)
	assert.True(t, suspended)

	// Unsuspend.
	err = svc.UnsuspendOrg(ctx, org.Slug)
	require.NoError(t, err)

	suspended, err = svc.IsOrgSuspended(ctx, org.Slug)
	require.NoError(t, err)
	assert.False(t, suspended)
}

func TestService_SuspendOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.SuspendOrg(ctx, "nonexistent", "reason", "admin1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_UnsuspendOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.UnsuspendOrg(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_IsOrgSuspended_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	suspended, err := svc.IsOrgSuspended(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, suspended)
}

func TestService_TransferOrgOwnership(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Transfer Org", Slug: "transfer-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	// Create an owner.
	m := models.OrgMembership{OrgID: org.ID, UserID: "old_owner", Role: models.RoleOwner}
	require.NoError(t, db.Create(&m).Error)

	// Transfer ownership.
	err := svc.TransferOrgOwnership(ctx, org.Slug, "new_owner")
	require.NoError(t, err)

	// Verify old owner is demoted to admin.
	var oldMember models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", org.ID, "old_owner").First(&oldMember).Error)
	assert.Equal(t, models.RoleAdmin, oldMember.Role)

	// Verify new owner is owner.
	var newMember models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", org.ID, "new_owner").First(&newMember).Error)
	assert.Equal(t, models.RoleOwner, newMember.Role)
}

func TestService_TransferOrgOwnership_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.TransferOrgOwnership(ctx, "nonexistent", "user1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_TransferOrgOwnership_ExistingMember(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := models.Org{Name: "Org", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)

	// Create owner and an existing member.
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "owner", Role: models.RoleOwner}).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "member", Role: models.RoleViewer}).Error)

	err := svc.TransferOrgOwnership(ctx, org.Slug, "member")
	require.NoError(t, err)

	var m models.OrgMembership
	require.NoError(t, db.Where("org_id = ? AND user_id = ?", org.ID, "member").First(&m).Error)
	assert.Equal(t, models.RoleOwner, m.Role)
}

func TestHandler_ListOrgs(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Org A", Slug: "org-a", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs", nil)
	h.ListOrgs(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_GetOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Detail", Slug: "detail", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs/detail", nil)
	r = chiCtx(r, "org", "detail")
	h.GetOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Detail")
}

func TestHandler_GetOrg_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs/nope", nil)
	r = chiCtx(r, "org", "nope")
	h.GetOrg(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs/", nil)
	r = chiCtx(r, "org", "")
	h.GetOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SuspendOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Susp", Slug: "susp", Metadata: "{}"})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/susp/suspend", strings.NewReader(`{"reason":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "susp")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "suspended")
}

func TestHandler_SuspendOrg_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/nope/suspend", strings.NewReader(`{"reason":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "nope")
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SuspendOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs//suspend", strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SuspendOrg_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/s/suspend", strings.NewReader("invalid"))
	r = chiCtx(r, "org", "s")
	h.SuspendOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UnsuspendOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.Org{Name: "Susp", Slug: "susp", Metadata: "{}", SuspendedAt: &now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/susp/unsuspend", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "susp")
	h.UnsuspendOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "unsuspended")
}

func TestHandler_UnsuspendOrg_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/nope/unsuspend", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "nope")
	h.UnsuspendOrg(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UnsuspendOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs//unsuspend", strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.UnsuspendOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferOwnership(t *testing.T) {
	h, db := setupTestHandler(t)
	org := models.Org{Name: "Xfer", Slug: "xfer", Metadata: "{}"}
	db.Create(&org)
	db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "old", Role: models.RoleOwner})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/xfer/transfer-ownership",
		strings.NewReader(`{"new_owner_user_id":"new"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "xfer")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "transferred")
}

func TestHandler_TransferOwnership_EmptyNewOwner(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/x/transfer-ownership",
		strings.NewReader(`{"new_owner_user_id":""}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", "x")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferOwnership_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs//transfer-ownership",
		strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferOwnership_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/x/transfer-ownership",
		strings.NewReader("invalid"))
	r = chiCtx(r, "org", "x")
	h.TransferOwnership(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg(t *testing.T) {
	h, db := setupTestHandler(t)
	org := models.Org{Name: "POrg", Slug: "porg", Metadata: "{}"}
	db.Create(&org)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/"+org.ID+"/purge",
		strings.NewReader(`{"confirm":"purge `+org.ID+`"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", org.ID)
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "purged")
}

func TestHandler_PurgeOrg_BadConfirm(t *testing.T) {
	h, db := setupTestHandler(t)
	org := models.Org{Name: "POrg", Slug: "porg2", Metadata: "{}"}
	db.Create(&org)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/"+org.ID+"/purge",
		strings.NewReader(`{"confirm":"wrong"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "org", org.ID)
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs//purge", strings.NewReader(`{}`))
	r = chiCtx(r, "org", "")
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeOrg_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/orgs/x/purge", strings.NewReader("invalid"))
	r = chiCtx(r, "org", "x")
	h.PurgeOrg(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListOrgs_DateFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	db.Create(&models.Org{Name: "Org1", Slug: "org1", Metadata: "{}"})

	w := httptest.NewRecorder()
	now := time.Now()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs?created_after="+now.Add(-24*time.Hour).Format(time.RFC3339)+"&created_before="+now.Add(time.Hour).Format(time.RFC3339), nil)
	h.ListOrgs(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListOrgs_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/orgs", nil)
	h.ListOrgs(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
