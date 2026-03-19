package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// buildDisplayName constructs a display name from first and last name claims.
// Returns empty string when both parts are empty.
func buildDisplayName(firstName, lastName string) string {
	first := strings.TrimSpace(firstName)
	last := strings.TrimSpace(lastName)
	switch {
	case first != "" && last != "":
		return first + " " + last
	case first != "":
		return first
	case last != "":
		return last
	default:
		return ""
	}
}

// JWTAuth middleware validates JWT tokens from the Authorization header.
func JWTAuth(validator *JWTValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				apierrors.Unauthorized(w, "authorization token is required")
				return
			}

			claims, err := validator.Validate(token)
			if err != nil {
				writeAuthError(w, err)
				return
			}

			ctx := SetUserContext(r.Context(), &UserContext{
				UserID:      claims.Subject,
				AuthMethod:  AuthMethodJWT,
				Email:       strings.TrimSpace(claims.Email),
				DisplayName: buildDisplayName(claims.FirstName, claims.LastName),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyAuthMiddleware
func APIKeyAuthMiddleware(service *APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				apierrors.Unauthorized(w, "API key is required")
				return
			}

			key, err := service.ValidateKey(rawKey)
			if err != nil {
				if errors.Is(err, ErrAPIKeyExpired) {
					apierrors.Unauthorized(w, "API key has expired")
					return
				}
				apierrors.Unauthorized(w, "invalid API key")
				return
			}

			ctx := SetUserContext(r.Context(), &UserContext{
				UserID:     "apikey:" + key.ID,
				AuthMethod: AuthMethodAPIKey,
				OrgID:      key.OrgID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DualAuth middleware accepts either a Clerk JWT (Authorization: Bearer) or
// an API key (X-API-Key header). Returns 401 if neither is present or valid.
func DualAuth(validator *JWTValidator, apiKeyService *APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try JWT first.
			token := extractBearerToken(r)
			if token != "" {
				claims, err := validator.Validate(token)
				if err != nil {
					writeAuthError(w, err)
					return
				}
				ctx := SetUserContext(r.Context(), &UserContext{
					UserID:      claims.Subject,
					AuthMethod:  AuthMethodJWT,
					Email:       strings.TrimSpace(claims.Email),
					DisplayName: buildDisplayName(claims.FirstName, claims.LastName),
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try API key.
			rawKey := r.Header.Get("X-API-Key")
			if rawKey != "" {
				key, err := apiKeyService.ValidateKey(rawKey)
				if err != nil {
					if errors.Is(err, ErrAPIKeyExpired) {
						apierrors.Unauthorized(w, "API key has expired")
						return
					}
					apierrors.Unauthorized(w, "invalid API key")
					return
				}
				ctx := SetUserContext(r.Context(), &UserContext{
					UserID:     "apikey:" + key.ID,
					AuthMethod: AuthMethodAPIKey,
					OrgID:      key.OrgID,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Neither present.
			apierrors.Unauthorized(w, "authorization is required (Bearer token or X-API-Key)")
		})
	}
}

// RequirePermission returns middleware that checks the authenticated user has
// the specified permission on the entity identified by URL params.
func RequirePermission(rbac *RBACEngine, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uc := GetUserContext(r.Context())
			if uc == nil {
				apierrors.Unauthorized(w, "authentication required")
				return
			}

			// Determine entity type and ID from URL params.
			entityType, entityID := resolveEntityFromURL(r)
			if entityType == "" || entityID == "" {
				// No entity context; only require authentication (no RBAC).
				next.ServeHTTP(w, r)
				return
			}

			role, err := rbac.ResolveRole(r.Context(), uc.UserID, entityType, entityID)
			if err != nil {
				apierrors.InternalError(w, "failed to resolve permissions")
				return
			}

			if role == "" {
				apierrors.Forbidden(w, "you do not have access to this resource")
				return
			}

			if !rbac.HasPermission(role, permission) {
				apierrors.Forbidden(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// resolveEntityFromURL extracts the most specific entity type and ID from
// chi URL params. Returns the most deeply nested entity found.
func resolveEntityFromURL(r *http.Request) (entityType, entityID string) {
	// Check from most specific to least specific.
	if id := chi.URLParam(r, "message"); id != "" {
		return "message", id
	}
	if id := chi.URLParam(r, "thread"); id != "" {
		return "thread", id
	}
	if id := chi.URLParam(r, "board"); id != "" {
		return "board", id
	}
	if id := chi.URLParam(r, "space"); id != "" {
		return "space", id
	}
	if id := chi.URLParam(r, "org"); id != "" {
		return "org", id
	}
	return "", ""
}

// extractBearerToken extracts the token from Authorization: Bearer header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

// writeAuthError writes the appropriate RFC 7807 error for JWT validation failures.
func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTokenExpired):
		apierrors.Unauthorized(w, "token has expired")
	case errors.Is(err, ErrTokenNotYet):
		apierrors.Unauthorized(w, "token is not yet valid")
	case errors.Is(err, ErrTokenIssuer):
		apierrors.Unauthorized(w, "invalid token issuer")
	case errors.Is(err, ErrTokenSignature):
		apierrors.Unauthorized(w, "invalid token signature")
	case errors.Is(err, ErrTokenKeyNotFound):
		apierrors.Unauthorized(w, "signing key not found")
	case errors.Is(err, ErrTokenMalformed):
		apierrors.Unauthorized(w, "malformed token")
	default:
		apierrors.Unauthorized(w, "authentication failed")
	}
}
