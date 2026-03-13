package admin

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/abraderAI/crm-project/api/internal/auth"
	apierrors "github.com/abraderAI/crm-project/api/pkg/errors"
)

// PlatformAdminOnly middleware checks that the authenticated user is an active
// platform admin. Returns 403 RFC 7807 if not.
func PlatformAdminOnly(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uc := auth.GetUserContext(r.Context())
			if uc == nil {
				apierrors.Unauthorized(w, "authentication required")
				return
			}

			isAdmin, err := svc.IsPlatformAdmin(r.Context(), uc.UserID)
			if err != nil {
				apierrors.InternalError(w, "failed to verify admin status")
				return
			}
			if !isAdmin {
				apierrors.Forbidden(w, "platform admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BanCheck middleware checks if the authenticated user is banned.
// Banned users receive 403 on all authenticated requests.
func BanCheck(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uc := auth.GetUserContext(r.Context())
			if uc == nil {
				// No user context — let auth middleware handle it.
				next.ServeHTTP(w, r)
				return
			}

			banned, err := svc.IsUserBanned(r.Context(), uc.UserID)
			if err != nil {
				apierrors.InternalError(w, "failed to check ban status")
				return
			}
			if banned {
				apierrors.Forbidden(w, "your account has been suspended")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OrgSuspensionCheck middleware blocks write operations on suspended orgs.
// Write methods (POST, PUT, PATCH, DELETE) return 503 for suspended orgs.
// Read methods (GET, HEAD, OPTIONS) are always allowed.
func OrgSuspensionCheck(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only check write methods.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Extract org from URL if present.
			orgParam := chi.URLParam(r, "org")
			if orgParam == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip admin routes — admins need to write to manage suspended orgs.
			if strings.Contains(r.URL.Path, "/v1/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			suspended, err := svc.IsOrgSuspended(r.Context(), orgParam)
			if err != nil {
				apierrors.InternalError(w, "failed to check org status")
				return
			}
			if suspended {
				apierrors.WriteProblem(w, apierrors.ProblemDetail{
					Type:   "https://httpstatuses.com/503",
					Title:  "Service Unavailable",
					Status: http.StatusServiceUnavailable,
					Detail: "this organization is currently suspended",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserShadowSync middleware syncs user shadow data on every authenticated request.
func UserShadowSync(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uc := auth.GetUserContext(r.Context())
			if uc != nil && uc.AuthMethod == auth.AuthMethodJWT {
				svc.SyncUserShadow(r.Context(), uc.UserID, "", "")
			}
			next.ServeHTTP(w, r)
		})
	}
}
