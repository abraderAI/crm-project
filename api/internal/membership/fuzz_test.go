package membership

import (
	"context"
	"testing"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// FuzzOrgMemberOperations fuzzes org membership operations with random user IDs and roles.
func FuzzOrgMemberOperations(f *testing.F) {
	type seed struct{ userID, role string }
	seeds := []seed{
		{"user1", "admin"},
		{"user2", "member"},
		{"user3", "owner"},
		{"user4", "viewer"},
		{"user5", "moderator"},
		{"user6", "contributor"},
		{"", "admin"},
		{"user", ""},
		{"user", "invalid_role"},
		{"user", "ADMIN"},
		{"user", "Admin"},
		{"' OR 1=1 --", "admin"},
		{"<script>alert(1)</script>", "member"},
		{"user\x00null", "viewer"},
		{"user\nnewline", "admin"},
		{"user_with_underscores", "owner"},
		{"user-with-hyphens", "member"},
		{"user.with.dots", "admin"},
		{"user@domain.com", "viewer"},
		{"clerk_user_abc123", "admin"},
		{"clerk_user_xyz789", "owner"},
		{"user_000000000000", "member"},
		{"a", "a"},
		{"   ", "admin"},
		{"null", "null"},
		{"true", "false"},
		{"0", "0"},
		{"-1", "-1"},
		{"user123", "OWNER"},
		{"user123", "Owner"},
		{"user123", "super_admin"},
		{"user123", "platform_admin"},
		{"user123", "god"},
		{"user123", "none"},
		{"αβγδ", "admin"},
		{"кириллица", "member"},
		{"中文用户", "viewer"},
		{"user 🎉", "admin"},
		{"very-long-user-id-that-exceeds-normal-length-for-testing-purposes-here", "admin"},
		{"user/slash", "member"},
		{"user\\backslash", "owner"},
		{"user#hash", "viewer"},
		{"user?query", "admin"},
		{"user+plus", "admin"},
		{"user%percent", "viewer"},
		{"user!bang", "member"},
		{"user*star", "owner"},
	}
	for _, s := range seeds {
		f.Add(s.userID, s.role)
	}

	f.Fuzz(func(t *testing.T, userID, role string) {
		env := setupDB(t)
		repo := NewRepository(env.db)
		ctx := context.Background()
		m := &models.OrgMembership{OrgID: env.orgID, UserID: userID, Role: models.Role(role)}
		// Should not panic regardless of input.
		_ = repo.AddOrgMember(ctx, m)
	})
}
