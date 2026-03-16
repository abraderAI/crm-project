package tier_test

import (
	"bytes"
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

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/database"
	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/internal/seed"
	"github.com/abraderAI/crm-project/api/internal/tier"
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

// --- Tier enum tests ---

func TestTier_String(t *testing.T) {
	tests := []struct {
		tier tier.Tier
		want string
	}{
		{tier.TierAnonymous, "anonymous"},
		{tier.TierRegistered, "registered"},
		{tier.TierCustomer, "customer"},
		{tier.TierDeftEmployee, "deft_employee"},
		{tier.TierCustomerAdmin, "customer_admin"},
		{tier.TierPlatformAdmin, "platform_admin"},
		{tier.Tier(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tier.String())
		})
	}
}

func TestTier_IsValid(t *testing.T) {
	assert.True(t, tier.TierAnonymous.IsValid())
	assert.True(t, tier.TierPlatformAdmin.IsValid())
	assert.False(t, tier.Tier(0).IsValid())
	assert.False(t, tier.Tier(7).IsValid())
	assert.False(t, tier.Tier(-1).IsValid())
}

// --- ResolveTier tests (all 6 paths) ---

func TestResolveTier_Anonymous_EmptyUserID(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	result, err := svc.ResolveTier("")
	require.NoError(t, err)
	assert.Equal(t, tier.TierAnonymous, result.Tier)
}

func TestResolveTier_Anonymous_UnknownUserID(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	result, err := svc.ResolveTier("unknown-user")
	require.NoError(t, err)
	assert.Equal(t, tier.TierAnonymous, result.Tier)
}

func TestResolveTier_Registered(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// Create a user shadow record (registered user with no org).
	shadow := &models.UserShadow{ClerkUserID: "user-registered", Email: "dev@example.com", DisplayName: "Dev User"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-registered")
	require.NoError(t, err)
	assert.Equal(t, tier.TierRegistered, result.Tier)
	assert.Empty(t, result.OrgID)
}

func TestResolveTier_CustomerMember(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// Create customer org and member.
	org := &models.Org{Name: "Customer Corp", Slug: "customer-corp", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "user-customer", Role: models.RoleViewer}
	require.NoError(t, db.Create(membership).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-customer", Email: "cust@example.com", DisplayName: "Customer"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-customer")
	require.NoError(t, err)
	assert.Equal(t, tier.TierCustomer, result.Tier)
	assert.Equal(t, org.ID, result.OrgID)
}

func TestResolveTier_CustomerAdmin(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	org := &models.Org{Name: "Admin Corp", Slug: "admin-corp", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "user-admin", Role: models.RoleAdmin}
	require.NoError(t, db.Create(membership).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-admin", Email: "admin@example.com", DisplayName: "Admin"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-admin")
	require.NoError(t, err)
	assert.Equal(t, tier.TierCustomerAdmin, result.Tier)
	assert.Equal(t, org.ID, result.OrgID)
}

func TestResolveTier_CustomerOwner(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	org := &models.Org{Name: "Owner Corp", Slug: "owner-corp", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	membership := &models.OrgMembership{OrgID: org.ID, UserID: "user-owner", Role: models.RoleOwner}
	require.NoError(t, db.Create(membership).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-owner", Email: "owner@example.com", DisplayName: "Owner"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-owner")
	require.NoError(t, err)
	assert.Equal(t, tier.TierCustomerAdmin, result.Tier)
	assert.Equal(t, tier.SubTypeOrgOwner, result.SubType)
	assert.Equal(t, org.ID, result.OrgID)
}

func TestResolveTier_DeftEmployee_Sales(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// Look up deft org.
	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)

	// Add user as deft org member.
	orgMem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "user-deft-sales", Role: models.RoleContributor}
	require.NoError(t, db.Create(orgMem).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-deft-sales", Email: "sales@deft.com", DisplayName: "Sales Rep"}
	require.NoError(t, db.Create(shadow).Error)

	// Add user to deft-sales space.
	var salesSpace models.Space
	require.NoError(t, db.Where("slug = ?", "deft-sales").First(&salesSpace).Error)
	spaceMem := &models.SpaceMembership{SpaceID: salesSpace.ID, UserID: "user-deft-sales", Role: models.RoleContributor}
	require.NoError(t, db.Create(spaceMem).Error)

	result, err := svc.ResolveTier("user-deft-sales")
	require.NoError(t, err)
	assert.Equal(t, tier.TierDeftEmployee, result.Tier)
	assert.Equal(t, tier.SubTypeDeftSales, result.SubType)
	assert.Equal(t, "sales", result.DeftDepartment)
	assert.Equal(t, deftOrg.ID, result.OrgID)
}

func TestResolveTier_DeftEmployee_Support(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)

	orgMem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "user-deft-support", Role: models.RoleContributor}
	require.NoError(t, db.Create(orgMem).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-deft-support", Email: "support@deft.com", DisplayName: "Support Agent"}
	require.NoError(t, db.Create(shadow).Error)

	var supportSpace models.Space
	require.NoError(t, db.Where("slug = ?", "deft-support").First(&supportSpace).Error)
	spaceMem := &models.SpaceMembership{SpaceID: supportSpace.ID, UserID: "user-deft-support", Role: models.RoleContributor}
	require.NoError(t, db.Create(spaceMem).Error)

	result, err := svc.ResolveTier("user-deft-support")
	require.NoError(t, err)
	assert.Equal(t, tier.TierDeftEmployee, result.Tier)
	assert.Equal(t, tier.SubTypeDeftSupport, result.SubType)
	assert.Equal(t, "support", result.DeftDepartment)
}

func TestResolveTier_DeftEmployee_Finance(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)

	orgMem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "user-deft-finance", Role: models.RoleContributor}
	require.NoError(t, db.Create(orgMem).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-deft-finance", Email: "finance@deft.com", DisplayName: "Finance Manager"}
	require.NoError(t, db.Create(shadow).Error)

	var financeSpace models.Space
	require.NoError(t, db.Where("slug = ?", "deft-finance").First(&financeSpace).Error)
	spaceMem := &models.SpaceMembership{SpaceID: financeSpace.ID, UserID: "user-deft-finance", Role: models.RoleContributor}
	require.NoError(t, db.Create(spaceMem).Error)

	result, err := svc.ResolveTier("user-deft-finance")
	require.NoError(t, err)
	assert.Equal(t, tier.TierDeftEmployee, result.Tier)
	assert.Equal(t, tier.SubTypeDeftFinance, result.SubType)
	assert.Equal(t, "finance", result.DeftDepartment)
}

func TestResolveTier_DeftEmployee_NoDepartment(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)

	// Deft member with no space membership.
	orgMem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "user-deft-general", Role: models.RoleContributor}
	require.NoError(t, db.Create(orgMem).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-deft-general", Email: "general@deft.com", DisplayName: "General"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-deft-general")
	require.NoError(t, err)
	assert.Equal(t, tier.TierDeftEmployee, result.Tier)
	assert.Equal(t, tier.SubTypeNone, result.SubType)
	assert.Empty(t, result.DeftDepartment)
}

func TestResolveTier_PlatformAdmin(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// Create platform admin.
	admin := &models.PlatformAdmin{UserID: "user-platform-admin", IsActive: true, GrantedBy: "system"}
	require.NoError(t, db.Create(admin).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-platform-admin", Email: "god@deft.com", DisplayName: "Platform Admin"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-platform-admin")
	require.NoError(t, err)
	assert.Equal(t, tier.TierPlatformAdmin, result.Tier)
}

func TestResolveTier_PlatformAdminTakesPrecedence(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// User is both platform admin and deft org member.
	admin := &models.PlatformAdmin{UserID: "user-multi", IsActive: true, GrantedBy: "system"}
	require.NoError(t, db.Create(admin).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-multi", Email: "multi@deft.com", DisplayName: "Multi User"}
	require.NoError(t, db.Create(shadow).Error)

	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	orgMem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "user-multi", Role: models.RoleAdmin}
	require.NoError(t, db.Create(orgMem).Error)

	result, err := svc.ResolveTier("user-multi")
	require.NoError(t, err)
	assert.Equal(t, tier.TierPlatformAdmin, result.Tier)
}

func TestResolveTier_InactivePlatformAdminIsNotTier6(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// Create active admin first, then deactivate (GORM ignores false for default:true booleans).
	admin := &models.PlatformAdmin{UserID: "user-inactive-admin", IsActive: true, GrantedBy: "system"}
	require.NoError(t, db.Create(admin).Error)
	require.NoError(t, db.Model(&models.PlatformAdmin{}).Where("user_id = ?", "user-inactive-admin").Update("is_active", false).Error)
	shadow := &models.UserShadow{ClerkUserID: "user-inactive-admin", Email: "inactive@deft.com", DisplayName: "Inactive Admin"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-inactive-admin")
	require.NoError(t, err)
	assert.Equal(t, tier.TierRegistered, result.Tier)
}

func TestResolveTier_DeftMemberTakesPrecedenceOverCustomer(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// User is in both deft org and a customer org.
	var deftOrg models.Org
	require.NoError(t, db.Where("slug = ?", seed.DeftOrgSlug).First(&deftOrg).Error)
	orgMem := &models.OrgMembership{OrgID: deftOrg.ID, UserID: "user-deft-plus-customer", Role: models.RoleContributor}
	require.NoError(t, db.Create(orgMem).Error)

	custOrg := &models.Org{Name: "CustOrg", Slug: "cust-org-precedence", Metadata: "{}"}
	require.NoError(t, db.Create(custOrg).Error)
	custMem := &models.OrgMembership{OrgID: custOrg.ID, UserID: "user-deft-plus-customer", Role: models.RoleOwner}
	require.NoError(t, db.Create(custMem).Error)

	shadow := &models.UserShadow{ClerkUserID: "user-deft-plus-customer", Email: "dual@deft.com", DisplayName: "Dual User"}
	require.NoError(t, db.Create(shadow).Error)

	result, err := svc.ResolveTier("user-deft-plus-customer")
	require.NoError(t, err)
	assert.Equal(t, tier.TierDeftEmployee, result.Tier)
}

// --- Home preferences tests ---

func TestHomePreferences_GetReturnsNilForNewUser(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	prefs, err := svc.GetHomePreferences("nonexistent-user")
	require.NoError(t, err)
	assert.Nil(t, prefs)
}

func TestHomePreferences_SaveAndGet(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	prefs := &models.UserHomePreferences{
		UserID: "user-prefs",
		Tier:   2,
		Layout: `[{"widget_id":"profile","visible":true}]`,
	}
	require.NoError(t, svc.SaveHomePreferences(prefs))

	got, err := svc.GetHomePreferences("user-prefs")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "user-prefs", got.UserID)
	assert.Equal(t, 2, got.Tier)
	assert.Contains(t, got.Layout, "profile")
}

func TestHomePreferences_Update(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	prefs := &models.UserHomePreferences{
		UserID: "user-update",
		Tier:   2,
		Layout: `[{"widget_id":"profile","visible":true}]`,
	}
	require.NoError(t, svc.SaveHomePreferences(prefs))

	// Update.
	prefs.Layout = `[{"widget_id":"profile","visible":false},{"widget_id":"forum","visible":true}]`
	prefs.Tier = 3
	require.NoError(t, svc.SaveHomePreferences(prefs))

	got, err := svc.GetHomePreferences("user-update")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 3, got.Tier)
	assert.Contains(t, got.Layout, "forum")
}

func TestHomePreferences_ValidationErrors(t *testing.T) {
	db := testDB(t)
	svc := tier.NewService(tier.NewRepository(db))

	// Missing user_id.
	err := svc.SaveHomePreferences(&models.UserHomePreferences{Tier: 2, Layout: "[]"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user_id")

	// Missing layout.
	err = svc.SaveHomePreferences(&models.UserHomePreferences{UserID: "u", Tier: 2})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "layout")

	// Invalid tier.
	err = svc.SaveHomePreferences(&models.UserHomePreferences{UserID: "u", Tier: 0, Layout: "[]"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tier")
}

// --- HTTP Handler tests ---

func newTestRouter(db *gorm.DB) *chi.Mux {
	repo := tier.NewRepository(db)
	svc := tier.NewService(repo)
	h := tier.NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/me/tier", h.GetTier)
	r.Get("/me/home-preferences", h.GetHomePreferences)
	r.Put("/me/home-preferences", h.PutHomePreferences)
	return r
}

func withAuth(r *http.Request, userID string) *http.Request {
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{
		UserID:     userID,
		AuthMethod: auth.AuthMethodJWT,
	})
	return r.WithContext(ctx)
}

func TestHandler_GetTier_Anonymous(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/me/tier", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result tier.TierResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, tier.TierAnonymous, result.Tier)
}

func TestHandler_GetTier_Registered(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	shadow := &models.UserShadow{ClerkUserID: "handler-reg", Email: "reg@example.com", DisplayName: "Reg"}
	require.NoError(t, db.Create(shadow).Error)

	req := withAuth(httptest.NewRequest(http.MethodGet, "/me/tier", nil), "handler-reg")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result tier.TierResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, tier.TierRegistered, result.Tier)
}

func TestHandler_GetTier_PlatformAdmin(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	admin := &models.PlatformAdmin{UserID: "handler-admin", IsActive: true, GrantedBy: "system"}
	require.NoError(t, db.Create(admin).Error)

	req := withAuth(httptest.NewRequest(http.MethodGet, "/me/tier", nil), "handler-admin")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result tier.TierResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, tier.TierPlatformAdmin, result.Tier)
}

func TestHandler_GetHomePreferences_Unauthenticated(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/me/home-preferences", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetHomePreferences_Empty(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	req := withAuth(httptest.NewRequest(http.MethodGet, "/me/home-preferences", nil), "user-no-prefs")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "null\n", w.Body.String())
}

func TestHandler_PutHomePreferences_Success(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"tier":2,"layout":[{"widget_id":"profile","visible":true},{"widget_id":"forum","visible":false}]}`
	req := withAuth(httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body)), "user-put-prefs")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify round-trip.
	req2 := withAuth(httptest.NewRequest(http.MethodGet, "/me/home-preferences", nil), "user-put-prefs")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var prefs models.UserHomePreferences
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &prefs))
	assert.Equal(t, "user-put-prefs", prefs.UserID)
	assert.Equal(t, 2, prefs.Tier)
}

func TestHandler_PutHomePreferences_EmptyLayout(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"tier":2,"layout":[]}`
	req := withAuth(httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body)), "user-empty")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PutHomePreferences_EmptyWidgetID(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"tier":2,"layout":[{"widget_id":"","visible":true}]}`
	req := withAuth(httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body)), "user-empty-id")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PutHomePreferences_DuplicateWidgetID(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"tier":2,"layout":[{"widget_id":"dup","visible":true},{"widget_id":"dup","visible":false}]}`
	req := withAuth(httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body)), "user-dup")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PutHomePreferences_InvalidTier(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"tier":99,"layout":[{"widget_id":"profile","visible":true}]}`
	req := withAuth(httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body)), "user-bad-tier")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PutHomePreferences_InvalidJSON(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{not json}`
	req := withAuth(httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body)), "user-bad-json")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PutHomePreferences_Unauthenticated(t *testing.T) {
	db := testDB(t)
	router := newTestRouter(db)

	body := `{"tier":2,"layout":[{"widget_id":"profile","visible":true}]}`
	req := httptest.NewRequest(http.MethodPut, "/me/home-preferences", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
