package vote

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abraderAI/crm-project/api/internal/models"
)

// FuzzWeightCalculation fuzzes the weight calculation with random roles and tiers.
func FuzzWeightCalculation(f *testing.F) {
	// Seed corpus with known values.
	seeds := []struct {
		role string
		tier string
	}{
		{"viewer", "free"},
		{"commenter", "pro"},
		{"contributor", "enterprise"},
		{"moderator", "free"},
		{"admin", "pro"},
		{"owner", "enterprise"},
		{"", ""},
		{"unknown", "unknown"},
		{"viewer", ""},
		{"", "free"},
		// Additional seeds to reach ≥50.
		{"viewer", "pro"},
		{"viewer", "enterprise"},
		{"commenter", "free"},
		{"commenter", "enterprise"},
		{"contributor", "free"},
		{"contributor", "pro"},
		{"moderator", "pro"},
		{"moderator", "enterprise"},
		{"admin", "free"},
		{"admin", "enterprise"},
		{"owner", "free"},
		{"owner", "pro"},
		{"VIEWER", "FREE"},
		{"Admin", "Pro"},
		{"a", "b"},
		{"x", "y"},
		{"role1", "tier1"},
		{"role2", "tier2"},
		{"viewer!", "free!"},
		{"view er", "fr ee"},
		{"🙂", "🙂"},
		{"viewer\n", "free\t"},
		{"viewer\x00", "free\x00"},
		{"<script>", "<img>"},
		{"' OR 1=1", "'; DROP TABLE"},
		{"a very long role name that should not cause any issues", "a very long tier name"},
		{"viewer", "gold"},
		{"owner", "platinum"},
		{"mod", "silver"},
		{"contrib", "bronze"},
		{"commenter", "starter"},
		{"viewer", "basic"},
		{"admin", "trial"},
		{"owner", "legacy"},
		{"moderator", "premium"},
		{"viewer", "custom"},
		{"contributor", "custom_tier"},
		{"admin", "nonprofit"},
		{"viewer", "education"},
		{"owner", "government"},
		{"moderator", "healthcare"},
		{"commenter", "financial"},
		{"viewer", "startup"},
		{"admin", "corporate"},
		{"owner", "unlimited"},
		{"moderator", "limited"},
	}

	for _, s := range seeds {
		f.Add(s.role, s.tier)
	}

	wc := DefaultWeightConfig()

	f.Fuzz(func(t *testing.T, role, tier string) {
		weight := wc.CalculateWeight(models.Role(role), tier)
		// Weight should always be >= 1 (default weight is 1, bonuses are non-negative).
		assert.GreaterOrEqual(t, weight, 1, "weight must be >= 1 for role=%q tier=%q", role, tier)
	})
}

// FuzzVoteToggle fuzzes the vote toggle operation with random user IDs.
func FuzzVoteToggle(f *testing.F) {
	seeds := []string{
		"user1", "user2", "user_abc", "",
		"a", "abc123", "user-with-dashes",
		"user.with.dots", "user@email.com",
		"user with spaces", "🙂user",
		"user\nline", "user\ttab",
		"user\x00null", "' OR 1=1 --",
		"<script>alert(1)</script>",
		"a-very-long-user-id-that-is-realistic",
		"usr_01234567890123456789",
		"usr", "u", "ab",
		"user_A", "USER_B",
		"123", "000", "999",
		"user-1", "user-2", "user-3",
		"test_user_alpha", "test_user_beta",
		"admin_user", "mod_user", "viewer_user",
		"contributor_user", "commenter_user",
		"owner_user", "new_user",
		"old_user", "temp_user",
		"bot_user", "service_account",
		"ci_user", "deploy_user",
		"api_user", "webhook_user",
		"system", "root", "anonymous",
		"guest", "superadmin",
		"user/slash", "user\\backslash",
		"user%percent", "user#hash",
		"user?question", "user&ampersand",
		"user=equals", "user+plus",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, userID string) {
		if userID == "" {
			return // Skip empty — service requires userID.
		}
		db := testDB(t)
		repo := NewRepository(db)
		svc := NewService(repo, nil)
		thread := seedThread(t, db)

		// The toggle should not panic.
		result, err := svc.Toggle(context.Background(), thread.ID, userID, models.RoleViewer, "free")
		if err != nil {
			return // Some inputs may fail validation; that's fine.
		}
		require.NotNil(t, result)
		assert.True(t, result.Voted)
		assert.GreaterOrEqual(t, result.VoteScore, 1)
	})
}
