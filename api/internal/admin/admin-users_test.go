package admin

import (
	"context"
	"encoding/json"
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

// --- User Shadow Service Tests ---

func TestService_SyncUserShadow(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "user1@example.com", "User One")

	var shadow models.UserShadow
	require.NoError(t, db.Where("clerk_user_id = ?", "user1").First(&shadow).Error)
	assert.Equal(t, "user1@example.com", shadow.Email)
	assert.Equal(t, "User One", shadow.DisplayName)
	assert.False(t, shadow.IsBanned)

	// Update on re-sync.
	svc.SyncUserShadow(ctx, "user1", "new@example.com", "New Name")
	require.NoError(t, db.Where("clerk_user_id = ?", "user1").First(&shadow).Error)
	assert.Equal(t, "new@example.com", shadow.Email)
}

func TestService_BanUnban(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "user1@example.com", "User One")

	// Ban.
	err := svc.BanUser(ctx, "user1", "spam", "admin1")
	require.NoError(t, err)

	banned, err := svc.IsUserBanned(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, banned)

	// Unban.
	err = svc.UnbanUser(ctx, "user1")
	require.NoError(t, err)

	banned, err = svc.IsUserBanned(ctx, "user1")
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestService_BanUser_CreatesIfNotExists(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.BanUser(ctx, "new_user", "abuse", "admin1")
	require.NoError(t, err)

	banned, err := svc.IsUserBanned(ctx, "new_user")
	require.NoError(t, err)
	assert.True(t, banned)
}

func TestService_UnbanUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	err := svc.UnbanUser(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_IsUserBanned_NoRecord(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	banned, err := svc.IsUserBanned(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, banned)
}

func TestService_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")
	svc.SyncUserShadow(ctx, "user2", "bob@example.com", "Bob")
	svc.SyncUserShadow(ctx, "user3", "charlie@example.com", "Charlie")

	// No filters.
	users, pageInfo, err := svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
	})
	require.NoError(t, err)
	assert.Len(t, users, 3)
	assert.False(t, pageInfo.HasMore)

	// Filter by email.
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
		Email:  "alice",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice@example.com", users[0].Email)

	// Filter by name.
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
		Name:   "Bob",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)

	// Filter by is_banned.
	_ = svc.BanUser(ctx, "user2", "spam", "admin1")
	trueVal := true
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params:   pagination.Params{Limit: 50},
		IsBanned: &trueVal,
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "user2", users[0].ClerkUserID)

	// Filter by user_id prefix.
	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
		UserID: "user1",
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestService_ListUsers_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.SyncUserShadow(ctx, strings.Repeat("a", 10)+string(rune('0'+i)), "", "")
	}

	users, pageInfo, err := svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 3},
	})
	require.NoError(t, err)
	assert.Len(t, users, 3)
	assert.True(t, pageInfo.HasMore)
	assert.NotEmpty(t, pageInfo.NextCursor)
}

func TestService_ListUsers_SeenDateRange(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "a@b.com", "A")

	now := time.Now()
	after := now.Add(-1 * time.Hour)
	before := now.Add(1 * time.Hour)

	users, _, err := svc.ListUsers(ctx, UserListParams{
		Params:    pagination.Params{Limit: 50},
		SeenAfter: &after,
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)

	users, _, err = svc.ListUsers(ctx, UserListParams{
		Params:     pagination.Params{Limit: 50},
		SeenBefore: &before,
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestService_GetUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")

	detail, err := svc.GetUser(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "alice@example.com", detail.Email)
	assert.Empty(t, detail.Memberships)

	// Not found.
	detail, err = svc.GetUser(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, detail)
}

func TestService_GetUser_WithMemberships(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")
	org := models.Org{Name: "Test Org", Slug: "test-org", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	m := models.OrgMembership{OrgID: org.ID, UserID: "user1", Role: models.RoleOwner}
	require.NoError(t, db.Create(&m).Error)

	detail, err := svc.GetUser(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Len(t, detail.Memberships, 1)
}

func TestService_GetUser_EnrichedOrgNames(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")
	org1 := models.Org{Name: "Acme Corp", Slug: "acme-corp", Metadata: "{}"}
	require.NoError(t, db.Create(&org1).Error)
	org2 := models.Org{Name: "Beta Inc", Slug: "beta-inc", Metadata: "{}"}
	require.NoError(t, db.Create(&org2).Error)

	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org1.ID, UserID: "user1", Role: models.RoleOwner}).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org2.ID, UserID: "user1", Role: models.RoleAdmin}).Error)

	detail, err := svc.GetUser(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Len(t, detail.Memberships, 2)

	// Verify org names are resolved.
	names := map[string]string{}
	for _, m := range detail.Memberships {
		names[m.OrgSlug] = m.OrgName
	}
	assert.Equal(t, "Acme Corp", names["acme-corp"])
	assert.Equal(t, "Beta Inc", names["beta-inc"])
}

func TestService_ListUsers_WithPrimaryOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	svc.SyncUserShadow(ctx, "user1", "alice@example.com", "Alice")
	svc.SyncUserShadow(ctx, "user2", "bob@example.com", "Bob")

	org := models.Org{Name: "Acme Corp", Slug: "acme-corp", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user1", Role: models.RoleOwner}).Error)

	users, _, err := svc.ListUsers(ctx, UserListParams{
		Params: pagination.Params{Limit: 50},
	})
	require.NoError(t, err)
	require.Len(t, users, 2)

	// user1 should have primary org info.
	var user1, user2 UserShadowWithOrg
	for _, u := range users {
		if u.ClerkUserID == "user1" {
			user1 = u
		} else {
			user2 = u
		}
	}
	assert.Equal(t, "Acme Corp", user1.PrimaryOrgName)
	assert.Equal(t, "acme-corp", user1.PrimaryOrgSlug)

	// user2 has no org — fields should be empty.
	assert.Empty(t, user2.PrimaryOrgName)
	assert.Empty(t, user2.PrimaryOrgSlug)
}

// --- User Handler Tests ---

func TestHandler_ListUsers(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})
	db.Create(&models.UserShadow{ClerkUserID: "u2", Email: "b@test.com", DisplayName: "B", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHandler_ListUsers_WithFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "alice@test.com", DisplayName: "Alice", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users?email=alice&is_banned=false", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var body struct{ Data []json.RawMessage }
	_ = json.NewDecoder(w.Body).Decode(&body)
	assert.Len(t, body.Data, 1)
}

func TestHandler_GetUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "user@test.com", DisplayName: "User", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users/u1", nil)
	r = chiCtx(r, "user_id", "u1")
	h.GetUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user@test.com")
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users/nonexistent", nil)
	r = chiCtx(r, "user_id", "nonexistent")
	h.GetUser(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users/", nil)
	r = chiCtx(r, "user_id", "")
	h.GetUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BanUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "ban_u", Email: "ban@test.com", DisplayName: "Ban", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/ban_u/ban", strings.NewReader(`{"reason":"spam"}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "ban_u")
	r = r.WithContext(auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin1", AuthMethod: auth.AuthMethodJWT}))
	h.BanUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "banned")
}

func TestHandler_BanUser_InvalidBody(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/ban", strings.NewReader("invalid"))
	r = chiCtx(r, "user_id", "u1")
	h.BanUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_BanUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users//ban", strings.NewReader(`{}`))
	r = chiCtx(r, "user_id", "")
	h.BanUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UnbanUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "test@test.com", DisplayName: "T", IsBanned: true, LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/u1/unban", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "u1")
	h.UnbanUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "unbanned")
}

func TestHandler_UnbanUser_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users/nope/unban", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "user_id", "nope")
	h.UnbanUser(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UnbanUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/users//unban", strings.NewReader(`{}`))
	r = chiCtx(r, "user_id", "")
	h.UnbanUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PurgeUser(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "purge_u", Email: "p@test.com", DisplayName: "P", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/users/purge_u/purge", nil)
	r = chiCtx(r, "user_id", "purge_u")
	h.PurgeUser(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "purged")
}

func TestHandler_PurgeUser_EmptyID(t *testing.T) {
	h, _ := setupTestHandler(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/admin/users//purge", nil)
	r = chiCtx(r, "user_id", "")
	h.PurgeUser(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListUsers_DateFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	now := time.Now()
	db.Create(&models.UserShadow{ClerkUserID: "u1", Email: "a@test.com", DisplayName: "A", LastSeenAt: now, SyncedAt: now})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users?seen_after="+now.Add(-time.Hour).Format(time.RFC3339)+"&seen_before="+now.Add(time.Hour).Format(time.RFC3339)+"&is_banned=true", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListUsers_Error(t *testing.T) {
	h, db := setupTestHandler(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	h.ListUsers(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
