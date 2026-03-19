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

// clerkTestServer creates a httptest.Server that returns the given Clerk user JSON,
// and a ClerkClient pointing at it.
func clerkTestServer(t *testing.T, responseBody string, statusCode int) (*httptest.Server, *auth.ClerkClient) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(ts.Close)
	// Build a ClerkClient that targets the test server.
	client := auth.NewClerkClientForTest("test-secret", ts.URL)
	return ts, client
}

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

func TestUserShadowSync_PropagatesEmailAndDisplayName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	handler := UserShadowSync(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := auth.SetUserContext(r.Context(), &auth.UserContext{
		UserID:      "email_user",
		AuthMethod:  auth.AuthMethodJWT,
		Email:       "user@example.com",
		DisplayName: "Email User",
	})
	handler.ServeHTTP(w, r.WithContext(ctx))
	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(100 * time.Millisecond)

	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "email_user").First(&shadow).Error
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", shadow.Email)
	assert.Equal(t, "Email User", shadow.DisplayName)
}

func TestSyncUserShadow_PreservesExistingEmail(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// First sync with real data.
	svc.SyncUserShadow(ctx, "preserve_user", "real@example.com", "Real Name")
	time.Sleep(50 * time.Millisecond)

	// Second sync with empty data — should NOT overwrite email or display_name.
	svc.SyncUserShadow(ctx, "preserve_user", "", "")
	time.Sleep(50 * time.Millisecond)

	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "preserve_user").First(&shadow).Error
	require.NoError(t, err)
	assert.Equal(t, "real@example.com", shadow.Email, "email should be preserved")
	assert.Equal(t, "Real Name", shadow.DisplayName, "display_name should be preserved")
}

func TestSyncUserShadow_ClerkFallback_NewUser(t *testing.T) {
	db := setupTestDB(t)
	_, clerkClient := clerkTestServer(t, `{
		"email_addresses": [{"email_address": "jane@example.com"}],
		"first_name": "Jane",
		"last_name": "Doe"
	}`, http.StatusOK)

	svc := NewService(db).withClerkClient(clerkClient)
	svc.SyncUserShadow(context.Background(), "new_clerk_user", "", "")

	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "new_clerk_user").First(&shadow).Error
	require.NoError(t, err)
	assert.Equal(t, "jane@example.com", shadow.Email, "Clerk API email should be stored")
	assert.Equal(t, "Jane Doe", shadow.DisplayName, "Clerk API name should be stored")
}

func TestSyncUserShadow_ClerkFallback_ExistingEmailSkipsClerk(t *testing.T) {
	db := setupTestDB(t)
	clerkCallCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		clerkCallCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"email_addresses": [{"email_address": "other@example.com"}]}`))
	}))
	t.Cleanup(ts.Close)
	clerkClient := auth.NewClerkClientForTest("key", ts.URL)

	svc := NewService(db).withClerkClient(clerkClient)
	ctx := context.Background()

	// Pre-seed shadow with an email.
	svc.SyncUserShadow(ctx, "existing_email_user", "existing@example.com", "Existing User")
	clerkCallCount = 0 // reset after first sync (which itself won't call Clerk since email is present)

	// Sync with empty JWT claims — should NOT call Clerk because email already stored.
	svc.SyncUserShadow(ctx, "existing_email_user", "", "")

	assert.Equal(t, 0, clerkCallCount, "Clerk API should not be called when email already in shadow")
	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "existing_email_user").First(&shadow).Error
	require.NoError(t, err)
	assert.Equal(t, "existing@example.com", shadow.Email, "original email should be preserved")
}

func TestSyncUserShadow_ClerkFallback_SkippedWhenNoClient(t *testing.T) {
	db := setupTestDB(t)
	// No Clerk client configured — should not panic and should still create shadow.
	svc := NewService(db)
	svc.SyncUserShadow(context.Background(), "no_clerk_user", "", "")

	var shadow models.UserShadow
	err := db.Where("clerk_user_id = ?", "no_clerk_user").First(&shadow).Error
	require.NoError(t, err)
	assert.Equal(t, "", shadow.Email, "email should remain empty without Clerk client")
}
