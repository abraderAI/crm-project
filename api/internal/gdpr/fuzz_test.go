package gdpr

import (
	"context"
	"testing"
)

// FuzzExportUserData fuzzes user data export with arbitrary user IDs.
func FuzzExportUserData(f *testing.F) {
	seeds := []string{
		"user_abc123",
		"",
		"user_000000000000",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"user\x00null",
		"user\nnewline",
		"user\ttab",
		"nonexistent_user",
		"a",
		"user-with-hyphens",
		"user_with_underscores",
		"user.with.dots",
		"user@domain.com",
		"clerk_user_abc123",
		"UPPERCASE_USER",
		"MiXeD_CaSe",
		"123numeric",
		"very-long-user-id-that-exceeds-normal-expected-length-for-a-clerk-user-id-value",
		"αβγδ",
		"кириллица",
		"中文用户",
		"user 🎉",
		"anonymized",
		"null",
		"undefined",
		"true",
		"false",
		"0",
		"-1",
		"   ",
		"user/slash",
		"user\\backslash",
		"user#hash",
		"user?query",
		"user+plus",
		"user%percent",
		"user!bang",
		"user*star",
		"user(paren",
		"user)close",
		"user[bracket",
		"user]close",
		"user{brace",
		"user}close",
		"user|pipe",
		"user~tilde",
		"user;semicolon",
		"user:colon",
		"user,comma",
		"user<less",
		"user>greater",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, userID string) {
		db := testDB(t)
		svc := NewService(db)
		ctx := context.Background()
		// Export should never panic for any userID.
		_, _ = svc.ExportUserData(ctx, userID)
	})
}

// FuzzPurgeUser fuzzes user purge with arbitrary user IDs.
func FuzzPurgeUser(f *testing.F) {
	seeds := []string{
		"user_abc123",
		"",
		"nonexistent",
		"' OR 1=1 --",
		"<script>alert(1)</script>",
		"user\x00null",
		"user\nnewline",
		"a",
		"user-hyphens",
		"user_underscores",
		"123numeric",
		"αβγ",
		"кирилл",
		"中文",
		"anonymized",
		"null",
		"user/slash",
		"user\\backslash",
		"   spaces   ",
		"UPPERCASE",
		"very-long-user-id-exceeding-normal-length-for-testing-purposes-here-indeed",
		"user.dots",
		"user@email",
		"user#hash",
		"user?query",
		"user+plus",
		"user%pct",
		"user!bang",
		"user*star",
		"user(paren",
		"user)close",
		"user[bracket",
		"user]close",
		"user{brace",
		"user}close",
		"user|pipe",
		"user~tilde",
		"user;semi",
		"user:colon",
		"user,comma",
		"user<less",
		"user>greater",
		"fuzz-user-purge",
		"clerk_abc",
		"user_xyz",
		"test_user",
		"0",
		"-1",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, userID string) {
		db := testDB(t)
		svc := NewService(db)
		ctx := context.Background()
		// Purge should never panic for any userID.
		_ = svc.PurgeUser(ctx, userID)
	})
}
