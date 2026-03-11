// Package auth provides authentication and authorization for the API server.
package auth

import (
	"context"
)

type contextKey string

const userContextKey contextKey = "user_context"

// AuthMethod represents how a user was authenticated.
type AuthMethod string

const (
	// AuthMethodJWT indicates authentication via Clerk JWT.
	AuthMethodJWT AuthMethod = "jwt"
	// AuthMethodAPIKey indicates authentication via API key.
	AuthMethodAPIKey AuthMethod = "api_key"
)

// UserContext holds authenticated user information set by auth middleware.
type UserContext struct {
	UserID     string     `json:"user_id"`
	AuthMethod AuthMethod `json:"auth_method"`
	OrgID      string     `json:"org_id,omitempty"` // Set when authenticated via API key.
}

// SetUserContext stores the UserContext in the request context.
func SetUserContext(ctx context.Context, uc *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, uc)
}

// GetUserContext retrieves the UserContext from the request context.
// Returns nil if no user context is set.
func GetUserContext(ctx context.Context) *UserContext {
	if uc, ok := ctx.Value(userContextKey).(*UserContext); ok {
		return uc
	}
	return nil
}
