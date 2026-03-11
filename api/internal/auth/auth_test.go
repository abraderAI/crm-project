package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/config"
	"github.com/abraderAI/crm-project/api/internal/models"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// --- Context Tests ---

func TestSetGetUserContext(t *testing.T) {
	uc := &UserContext{UserID: "user-1", AuthMethod: AuthMethodJWT}
	ctx := SetUserContext(context.Background(), uc)
	got := GetUserContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, AuthMethodJWT, got.AuthMethod)
}

func TestGetUserContext_Missing(t *testing.T) {
	got := GetUserContext(context.Background())
	assert.Nil(t, got)
}

func TestUserContext_APIKey(t *testing.T) {
	uc := &UserContext{UserID: "apikey:123", AuthMethod: AuthMethodAPIKey, OrgID: "org-1"}
	ctx := SetUserContext(context.Background(), uc)
	got := GetUserContext(ctx)
	require.NotNil(t, got)
	assert.Equal(t, "apikey:123", got.UserID)
	assert.Equal(t, AuthMethodAPIKey, got.AuthMethod)
	assert.Equal(t, "org-1", got.OrgID)
}

// --- JWT Validation Tests ---

func TestJWT_ValidToken(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://clerk.example.com"
	v := testValidator(t, kp, issuer)

	token := signTestJWT(t, kp, testValidClaims(issuer))
	claims, err := v.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "user_test123", claims.Subject)
	assert.Equal(t, issuer, claims.Issuer)
}

func TestJWT_ExpiredToken(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://clerk.example.com"
	v := testValidator(t, kp, issuer)

	token := signTestJWT(t, kp, testExpiredClaims(issuer))
	_, err := v.Validate(token)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestJWT_WrongIssuer(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://clerk.example.com")

	token := signTestJWT(t, kp, testWrongIssuerClaims())
	_, err := v.Validate(token)
	assert.ErrorIs(t, err, ErrTokenIssuer)
}

func TestJWT_NotYetValid(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://clerk.example.com"
	v := testValidator(t, kp, issuer)

	token := signTestJWT(t, kp, testFutureClaims(issuer))
	_, err := v.Validate(token)
	assert.ErrorIs(t, err, ErrTokenNotYet)
}

func TestJWT_MalformedToken(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://clerk.example.com")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dots", "nodots"},
		{"one dot", "one.dot"},
		{"four dots", "a.b.c.d"},
		{"invalid base64 header", "!!!.YWJj.ZGVm"},
		{"invalid base64 payload", "eyJhbGciOiJSUzI1NiJ9.!!!.ZGVm"},
		{"invalid json header", "YWJj.eyJ0ZXN0IjoxfQ.ZGVm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestJWT_WrongSigningKey(t *testing.T) {
	kp := generateTestKeyPair(t)
	wrongKP := generateTestKeyPair(t)
	issuer := "https://clerk.example.com"
	v := testValidator(t, kp, issuer)

	// Sign with wrong key.
	token := signTestJWT(t, wrongKP, testValidClaims(issuer))
	_, err := v.Validate(token)
	assert.ErrorIs(t, err, ErrTokenSignature)
}

func TestJWT_UnknownKid(t *testing.T) {
	kp := generateTestKeyPair(t)
	kp2 := generateTestKeyPair(t)
	kp2.Kid = "unknown-kid"
	issuer := "https://clerk.example.com"
	v := testValidator(t, kp, issuer)

	token := signTestJWT(t, kp2, testValidClaims(issuer))
	_, err := v.Validate(token)
	assert.Error(t, err)
}

func TestJWT_FetchJWKS(t *testing.T) {
	kp := generateTestKeyPair(t)
	v, issuerURL := testValidatorWithJWKS(t, kp)

	token := signTestJWT(t, kp, testValidClaims(issuerURL))
	claims, err := v.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "user_test123", claims.Subject)
}

func TestJWT_EmptyIssuerAcceptsAll(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "")

	token := signTestJWT(t, kp, JWTClaims{
		Subject:   "user1",
		Issuer:    "https://any-issuer.com",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	})
	claims, err := v.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "user1", claims.Subject)
}

func TestJWT_NoExpiry(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://clerk.example.com"
	v := testValidator(t, kp, issuer)

	token := signTestJWT(t, kp, JWTClaims{
		Subject: "user1",
		Issuer:  issuer,
	})
	claims, err := v.Validate(token)
	require.NoError(t, err)
	assert.Equal(t, "user1", claims.Subject)
}

// --- Base64URL Tests ---

func TestBase64URLDecode_Padding(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no padding needed", "AQAB"},
		{"one pad char", "YWI"},
		{"two pad chars", "YQ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := base64URLDecode(tt.input)
			assert.NoError(t, err)
		})
	}
}

// --- RBAC Engine Tests ---

func testRBACPolicy(t *testing.T) *config.RBACPolicy {
	t.Helper()
	yaml := `
resolution:
  strategy: "explicit_override_with_parent_fallback"
  order: [board, space, org]
roles:
  hierarchy: [viewer, commenter, contributor, moderator, admin, owner]
  permissions:
    viewer: [read]
    commenter: [read, comment]
    contributor: [read, comment, create, update_own]
    moderator: [read, comment, create, update_own, update_any, delete_any, pin, lock, move, moderate]
    admin: [read, comment, create, update_own, update_any, delete_any, pin, lock, move, moderate, manage_members, manage_settings]
    owner: [read, comment, create, update_own, update_any, delete_any, pin, lock, move, moderate, manage_members, manage_settings, delete_entity, manage_billing, transfer_ownership]
defaults:
  org_member_role: "viewer"
  space_member_role: "viewer"
  board_member_role: "viewer"
`
	policy, err := config.ParseRBACPolicy([]byte(yaml))
	require.NoError(t, err)
	return policy
}

func TestRBAC_ResolveRole_OrgLevel(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "Test", Slug: "rbac-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleAdmin}).Error)

	role, err := engine.ResolveRole(context.Background(), "user-1", "org", org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleAdmin, role)
}

func TestRBAC_ResolveRole_SpaceLevel(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "rbac-space-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "rbac-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	require.NoError(t, db.Create(&models.SpaceMembership{SpaceID: space.ID, UserID: "user-1", Role: models.RoleModerator}).Error)

	role, err := engine.ResolveRole(context.Background(), "user-1", "space", space.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleModerator, role)
}

func TestRBAC_ResolveRole_BoardLevel(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "rbac-board-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "rbac-board-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "rbac-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	require.NoError(t, db.Create(&models.BoardMembership{BoardID: board.ID, UserID: "user-1", Role: models.RoleContributor}).Error)

	role, err := engine.ResolveRole(context.Background(), "user-1", "board", board.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleContributor, role)
}

func TestRBAC_ResolveRole_BoardFallsBackToSpace(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "rbac-fb-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "rbac-fb-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "rbac-fb-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	// Only space membership, no board membership.
	require.NoError(t, db.Create(&models.SpaceMembership{SpaceID: space.ID, UserID: "user-1", Role: models.RoleAdmin}).Error)

	role, err := engine.ResolveRole(context.Background(), "user-1", "board", board.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleAdmin, role)
}

func TestRBAC_ResolveRole_BoardFallsBackToOrg(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "rbac-fbo-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "rbac-fbo-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "rbac-fbo-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	// Only org membership.
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleViewer}).Error)

	role, err := engine.ResolveRole(context.Background(), "user-1", "board", board.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleViewer, role)
}

func TestRBAC_ResolveRole_NoMembership(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "rbac-none-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	role, err := engine.ResolveRole(context.Background(), "nonexistent-user", "org", org.ID)
	require.NoError(t, err)
	assert.Equal(t, models.Role(""), role)
}

func TestRBAC_ResolveRole_BoardOverridesSpace(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "rbac-override-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "rbac-override-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "rbac-override-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)

	// Both space and board memberships.
	require.NoError(t, db.Create(&models.SpaceMembership{SpaceID: space.ID, UserID: "user-1", Role: models.RoleAdmin}).Error)
	require.NoError(t, db.Create(&models.BoardMembership{BoardID: board.ID, UserID: "user-1", Role: models.RoleViewer}).Error)

	// Board level overrides space level.
	role, err := engine.ResolveRole(context.Background(), "user-1", "board", board.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleViewer, role)
}

func TestRBAC_ResolveRole_UnknownEntityType(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	_, err := engine.ResolveRole(context.Background(), "user-1", "unknown", "some-id")
	assert.Error(t, err)
}

func TestRBAC_HasPermission(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	tests := []struct {
		role       models.Role
		permission string
		expected   bool
	}{
		{models.RoleViewer, "read", true},
		{models.RoleViewer, "create", false},
		{models.RoleContributor, "create", true},
		{models.RoleContributor, "delete_any", false},
		{models.RoleModerator, "moderate", true},
		{models.RoleAdmin, "manage_members", true},
		{models.RoleOwner, "transfer_ownership", true},
		{models.RoleOwner, "nonexistent", false},
		{models.Role("invalid"), "read", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role)+"/"+tt.permission, func(t *testing.T) {
			assert.Equal(t, tt.expected, engine.HasPermission(tt.role, tt.permission))
		})
	}
}

func TestRBAC_IsHigherOrEqual(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	assert.True(t, engine.IsHigherOrEqual(models.RoleOwner, models.RoleViewer))
	assert.True(t, engine.IsHigherOrEqual(models.RoleAdmin, models.RoleAdmin))
	assert.False(t, engine.IsHigherOrEqual(models.RoleViewer, models.RoleAdmin))
}

func TestRBAC_LookupOrgForEntity(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "lookup-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	space := &models.Space{OrgID: org.ID, Name: "S", Slug: "lookup-space", Type: models.SpaceTypeGeneral, Metadata: "{}"}
	require.NoError(t, db.Create(space).Error)
	board := &models.Board{SpaceID: space.ID, Name: "B", Slug: "lookup-board", Metadata: "{}"}
	require.NoError(t, db.Create(board).Error)
	thread := &models.Thread{BoardID: board.ID, Title: "T", Slug: "lookup-thread", AuthorID: "u1", Metadata: "{}"}
	require.NoError(t, db.Create(thread).Error)
	msg := &models.Message{ThreadID: thread.ID, Body: "M", AuthorID: "u1", Type: models.MessageTypeComment, Metadata: "{}"}
	require.NoError(t, db.Create(msg).Error)

	ctx := context.Background()

	orgID, err := engine.LookupOrgForEntity(ctx, "org", org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, orgID)

	orgID, err = engine.LookupOrgForEntity(ctx, "space", space.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, orgID)

	orgID, err = engine.LookupOrgForEntity(ctx, "board", board.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, orgID)

	orgID, err = engine.LookupOrgForEntity(ctx, "thread", thread.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, orgID)

	orgID, err = engine.LookupOrgForEntity(ctx, "message", msg.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, orgID)

	_, err = engine.LookupOrgForEntity(ctx, "unknown", "id")
	assert.Error(t, err)
}

// --- API Key Tests ---

func TestAPIKey_CreateAndValidate(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	org := &models.Org{Name: "O", Slug: "ak-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	result, err := service.CreateKey(org.ID, "Test Key", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Key)
	assert.NotEmpty(t, result.ID)
	assert.True(t, strings.HasPrefix(result.Key, "deft_live_"))
	assert.Equal(t, "Test Key", result.Name)

	// Validate the key.
	key, err := service.ValidateKey(result.Key)
	require.NoError(t, err)
	assert.Equal(t, org.ID, key.OrgID)
}

func TestAPIKey_List(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	org := &models.Org{Name: "O", Slug: "ak-list-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	_, err := service.CreateKey(org.ID, "Key1", nil)
	require.NoError(t, err)
	_, err = service.CreateKey(org.ID, "Key2", nil)
	require.NoError(t, err)

	keys, err := service.ListKeys(org.ID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestAPIKey_Revoke(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	org := &models.Org{Name: "O", Slug: "ak-revoke-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	result, err := service.CreateKey(org.ID, "ToRevoke", nil)
	require.NoError(t, err)

	err = service.RevokeKey(org.ID, result.ID)
	require.NoError(t, err)

	// Key should now be invalid (soft deleted).
	_, err = service.ValidateKey(result.Key)
	assert.ErrorIs(t, err, ErrAPIKeyInvalid)
}

func TestAPIKey_Expired(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	org := &models.Org{Name: "O", Slug: "ak-exp-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	past := time.Now().Add(-1 * time.Hour)
	result, err := service.CreateKey(org.ID, "Expired", &past)
	require.NoError(t, err)

	_, err = service.ValidateKey(result.Key)
	assert.ErrorIs(t, err, ErrAPIKeyExpired)
}

func TestAPIKey_InvalidPrefix(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	_, err := service.ValidateKey("invalid_prefix_key")
	assert.ErrorIs(t, err, ErrAPIKeyInvalid)
}

func TestAPIKey_NonexistentKey(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	_, err := service.ValidateKey("deft_live_0000000000000000000000000000000000000000000000000000000000000000")
	assert.ErrorIs(t, err, ErrAPIKeyInvalid)
}

func TestAPIKey_RevokeNonexistent(t *testing.T) {
	db := testDB(t)
	service := NewAPIKeyService(db)

	err := service.RevokeKey("nonexistent-org", "nonexistent-key")
	assert.Error(t, err)
}

// --- Middleware Tests ---

func TestDualAuth_JWT(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://test.example.com"
	v := testValidator(t, kp, issuer)
	db := testDB(t)
	akService := NewAPIKeyService(db)

	handler := DualAuth(v, akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := GetUserContext(r.Context())
		require.NotNil(t, uc)
		assert.Equal(t, "user_test123", uc.UserID)
		assert.Equal(t, AuthMethodJWT, uc.AuthMethod)
		w.WriteHeader(http.StatusOK)
	}))

	token := signTestJWT(t, kp, testValidClaims(issuer))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDualAuth_APIKey(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://test.example.com")
	db := testDB(t)
	akService := NewAPIKeyService(db)

	org := &models.Org{Name: "O", Slug: "dual-ak-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	result, err := akService.CreateKey(org.ID, "Test", nil)
	require.NoError(t, err)

	handler := DualAuth(v, akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := GetUserContext(r.Context())
		require.NotNil(t, uc)
		assert.Equal(t, AuthMethodAPIKey, uc.AuthMethod)
		assert.Equal(t, org.ID, uc.OrgID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", result.Key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDualAuth_NoAuth(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://test.example.com")
	db := testDB(t)
	akService := NewAPIKeyService(db)

	handler := DualAuth(v, akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
}

func TestDualAuth_ExpiredJWT(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://test.example.com"
	v := testValidator(t, kp, issuer)
	db := testDB(t)
	akService := NewAPIKeyService(db)

	handler := DualAuth(v, akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	token := signTestJWT(t, kp, testExpiredClaims(issuer))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDualAuth_InvalidAPIKey(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://test.example.com")
	db := testDB(t)
	akService := NewAPIKeyService(db)

	handler := DualAuth(v, akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "deft_live_invalid")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDualAuth_ExpiredAPIKey(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://test.example.com")
	db := testDB(t)
	akService := NewAPIKeyService(db)

	org := &models.Org{Name: "O", Slug: "dual-exp-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	past := time.Now().Add(-1 * time.Hour)
	result, err := akService.CreateKey(org.ID, "Expired", &past)
	require.NoError(t, err)

	handler := DualAuth(v, akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", result.Key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_Middleware(t *testing.T) {
	kp := generateTestKeyPair(t)
	issuer := "https://test.example.com"
	v := testValidator(t, kp, issuer)

	handler := JWTAuth(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := GetUserContext(r.Context())
		require.NotNil(t, uc)
		w.WriteHeader(http.StatusOK)
	}))

	token := signTestJWT(t, kp, testValidClaims(issuer))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_MissingToken(t *testing.T) {
	kp := generateTestKeyPair(t)
	v := testValidator(t, kp, "https://test.example.com")

	handler := JWTAuth(v)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyAuth_Middleware(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	org := &models.Org{Name: "O", Slug: "akmw-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	result, err := akService.CreateKey(org.ID, "Test", nil)
	require.NoError(t, err)

	handler := APIKeyAuthMiddleware(akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uc := GetUserContext(r.Context())
		require.NotNil(t, uc)
		assert.Equal(t, AuthMethodAPIKey, uc.AuthMethod)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", result.Key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)

	handler := APIKeyAuthMiddleware(akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- RequirePermission Tests ---

func TestRequirePermission_Allowed(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "perm-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleAdmin}).Error)

	r := chi.NewRouter()
	r.Route("/v1/orgs/{org}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := SetUserContext(req.Context(), &UserContext{UserID: "user-1", AuthMethod: AuthMethodJWT})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.With(RequirePermission(engine, "read")).Get("/test", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_Denied(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "perm-denied-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	require.NoError(t, db.Create(&models.OrgMembership{OrgID: org.ID, UserID: "user-1", Role: models.RoleViewer}).Error)

	r := chi.NewRouter()
	r.Route("/v1/orgs/{org}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := SetUserContext(req.Context(), &UserContext{UserID: "user-1", AuthMethod: AuthMethodJWT})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.With(RequirePermission(engine, "manage_members")).Get("/test", func(w http.ResponseWriter, req *http.Request) {
			t.Fatal("should not reach handler")
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
}

func TestRequirePermission_NoMembership(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "perm-nomemb-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	r := chi.NewRouter()
	r.Route("/v1/orgs/{org}", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := SetUserContext(req.Context(), &UserContext{UserID: "nonmember", AuthMethod: AuthMethodJWT})
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		r.With(RequirePermission(engine, "read")).Get("/test", func(w http.ResponseWriter, req *http.Request) {
			t.Fatal("should not reach handler")
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_NoAuth(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	handler := RequirePermission(engine, "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_NoEntityContext(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	handler := RequirePermission(engine, "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := SetUserContext(req.Context(), &UserContext{UserID: "user-1", AuthMethod: AuthMethodJWT})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- API Key Handler Tests ---

func TestAPIKeyHandler_Create(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	handler := NewAPIKeyHandler(akService)

	org := &models.Org{Name: "O", Slug: "akh-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/api-keys", handler.Create)

	body := `{"name":"My Key"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var result CreateKeyResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.True(t, strings.HasPrefix(result.Key, "deft_live_"))
	assert.Equal(t, "My Key", result.Name)
}

func TestAPIKeyHandler_Create_MissingName(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	handler := NewAPIKeyHandler(akService)

	org := &models.Org{Name: "O", Slug: "akh-noname-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/api-keys", handler.Create)

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAPIKeyHandler_List(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	handler := NewAPIKeyHandler(akService)

	org := &models.Org{Name: "O", Slug: "akh-list-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	_, err := akService.CreateKey(org.ID, "K1", nil)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Get("/v1/orgs/{org}/api-keys", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/v1/orgs/"+org.ID+"/api-keys", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
}

func TestAPIKeyHandler_Revoke(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	handler := NewAPIKeyHandler(akService)

	org := &models.Org{Name: "O", Slug: "akh-revoke-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)
	result, err := akService.CreateKey(org.ID, "ToRevoke", nil)
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Delete("/v1/orgs/{org}/api-keys/{id}", handler.Revoke)

	req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+org.ID+"/api-keys/"+result.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAPIKeyHandler_Revoke_NotFound(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	handler := NewAPIKeyHandler(akService)

	org := &models.Org{Name: "O", Slug: "akh-rvnf-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	r := chi.NewRouter()
	r.Delete("/v1/orgs/{org}/api-keys/{id}", handler.Revoke)

	req := httptest.NewRequest(http.MethodDelete, "/v1/orgs/"+org.ID+"/api-keys/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- extractBearerToken Tests ---

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"valid", "Bearer mytoken", "mytoken"},
		{"lowercase bearer", "bearer mytoken", "mytoken"},
		{"no bearer", "Basic abc", ""},
		{"empty", "", ""},
		{"bearer only", "Bearer ", ""},
		{"no space", "Bearertoken", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := extractBearerToken(req)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// --- writeAuthError Tests ---

func TestWriteAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"expired", ErrTokenExpired},
		{"not yet", ErrTokenNotYet},
		{"issuer", ErrTokenIssuer},
		{"signature", ErrTokenSignature},
		{"key not found", ErrTokenKeyNotFound},
		{"malformed", ErrTokenMalformed},
		{"generic", errors.New("some error")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeAuthError(w, tt.err)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

			var problem apierrors.ProblemDetail
			require.NoError(t, json.NewDecoder(w.Body).Decode(&problem))
			assert.Equal(t, 401, problem.Status)
		})
	}
}

// --- Permission Check per Role Tests ---

func TestPermissionMatrix(t *testing.T) {
	db := testDB(t)
	policy := testRBACPolicy(t)
	engine := NewRBACEngine(policy, db)

	org := &models.Org{Name: "O", Slug: "matrix-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	roles := []models.Role{
		models.RoleViewer, models.RoleCommenter, models.RoleContributor,
		models.RoleModerator, models.RoleAdmin, models.RoleOwner,
	}
	permissions := []string{
		"read", "comment", "create", "update_own", "update_any",
		"delete_any", "pin", "lock", "move", "moderate",
		"manage_members", "manage_settings", "delete_entity",
		"manage_billing", "transfer_ownership",
	}

	for _, role := range roles {
		for _, perm := range permissions {
			t.Run(string(role)+"/"+perm, func(t *testing.T) {
				result := engine.HasPermission(role, perm)
				// Just verify it doesn't panic and returns a bool.
				_ = result
			})
		}
	}
}

// --- APIKeyAuth expired key middleware test ---

func TestAPIKeyAuth_ExpiredKey(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	org := &models.Org{Name: "O", Slug: "akmw-exp-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	past := time.Now().Add(-1 * time.Hour)
	result, err := akService.CreateKey(org.ID, "Expired", &past)
	require.NoError(t, err)

	handler := APIKeyAuthMiddleware(akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", result.Key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)

	handler := APIKeyAuthMiddleware(akService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "deft_live_0000000000000000000000000000000000000000000000000000000000000000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- resolveEntityFromURL tests ---

func TestResolveEntityFromURL(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		url      string
		wantType string
		wantID   string
	}{
		{"org", "/v1/orgs/{org}", "/v1/orgs/org-1", "org", "org-1"},
		{"space", "/v1/orgs/{org}/spaces/{space}", "/v1/orgs/o/spaces/sp-1", "space", "sp-1"},
		{"board", "/v1/orgs/{org}/spaces/{space}/boards/{board}", "/v1/orgs/o/spaces/s/boards/b-1", "board", "b-1"},
		{"thread", "/v1/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}", "/v1/orgs/o/spaces/s/boards/b/threads/t-1", "thread", "t-1"},
		{"message", "/v1/orgs/{org}/spaces/{space}/boards/{board}/threads/{thread}/messages/{message}", "/v1/orgs/o/spaces/s/boards/b/threads/t/messages/m-1", "message", "m-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			var gotType, gotID string
			r.Get(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				gotType, gotID = resolveEntityFromURL(r)
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantType, gotType)
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}

// --- Handler edge case: invalid JSON body ---

func TestAPIKeyHandler_Create_InvalidBody(t *testing.T) {
	db := testDB(t)
	akService := NewAPIKeyService(db)
	handler := NewAPIKeyHandler(akService)

	org := &models.Org{Name: "O", Slug: "akh-badbody-org", Metadata: "{}"}
	require.NoError(t, db.Create(org).Error)

	r := chi.NewRouter()
	r.Post("/v1/orgs/{org}/api-keys", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/v1/orgs/"+org.ID+"/api-keys", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
