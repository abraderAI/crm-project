package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/auth"
	"github.com/abraderAI/crm-project/api/internal/models"
)

// --- Middleware Tests ---

func TestPlatformAdminOnly_NoAuth(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := PlatformAdminOnly(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPlatformAdminOnly_NotAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := PlatformAdminOnly(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "regular_user", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPlatformAdminOnly_IsAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_, _ = svc.AddPlatformAdmin(context.Background(), "admin_user", "bootstrap")

	handler := PlatformAdminOnly(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "admin_user", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBanCheck_NotBanned(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := BanCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBanCheck_Banned(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	_ = svc.BanUser(context.Background(), "user1", "spam", "admin1")

	handler := BanCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "user1", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestBanCheck_NoUserContext(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := BanCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code) // Passes through, auth middleware handles later.
}

func TestOrgSuspensionCheck_ReadMethod(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	org := models.Org{Name: "Test", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	_ = svc.SuspendOrg(context.Background(), org.Slug, "test", "admin1")

	// GET should be allowed even for suspended orgs.
	router := chi.NewRouter()
	router.Get("/orgs/{org}", func(w http.ResponseWriter, r *http.Request) {
		OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/orgs/"+org.Slug, nil)
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgSuspensionCheck_WriteMethodBlocked(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	org := models.Org{Name: "Test", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	_ = svc.SuspendOrg(context.Background(), org.Slug, "test", "admin1")

	router := chi.NewRouter()
	router.Post("/v1/orgs/{org}/spaces", func(w http.ResponseWriter, r *http.Request) {
		OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.Slug+"/spaces", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestOrgSuspensionCheck_AdminRouteAllowed(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	org := models.Org{Name: "Test", Slug: "test", Metadata: "{}"}
	require.NoError(t, db.Create(&org).Error)
	_ = svc.SuspendOrg(context.Background(), org.Slug, "test", "admin1")

	router := chi.NewRouter()
	router.Post("/v1/admin/orgs/{org}/unsuspend", func(w http.ResponseWriter, r *http.Request) {
		OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/admin/orgs/"+org.Slug+"/unsuspend", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOrgSuspensionCheck_NoOrgParam(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := OrgSuspensionCheck(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/search", strings.NewReader("{}"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserShadowSync_JWT(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := UserShadowSync(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "sync_user", AuthMethod: auth.AuthMethodJWT})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)

	// Give goroutine time to finish.
	time.Sleep(100 * time.Millisecond)

	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "sync_user").First(&shadow).Error
	require.NoError(t, err)
}

func TestUserShadowSync_APIKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := UserShadowSync(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{UserID: "apikey_user", AuthMethod: auth.AuthMethodAPIKey})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)

	// API key auth should NOT trigger sync.
	time.Sleep(100 * time.Millisecond)
	var count int64
	db.Model(&models.UserShadow{}).Where("clerk_user_id = ?", "apikey_user").Count(&count)
	assert.Equal(t, int64(0), count)
}
