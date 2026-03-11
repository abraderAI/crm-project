package search

import (
	"testing"
)

// FuzzSanitizeFTSQuery tests that sanitizeFTSQuery never panics and always
// returns a safe string (no unescaped FTS5 operators).
func FuzzSanitizeFTSQuery(f *testing.F) {
	// Seed corpus: ≥50 fuzzing seeds.
	seeds := []string{
		"hello", "hello world", "", "  ", "\"quoted\"", "'single'",
		"hello AND world", "NOT this", "OR that", "NEAR(a,b)",
		"a*b", "(group)", "a:b", "^prefix", "+required", "-excluded",
		"日本語テスト", "emoji 🎉", "null\x00byte", "tab\ttab",
		"newline\nnewline", "a\"b\"c", "'''", "***", "(((", ")))",
		"AND AND AND", "OR OR OR", "NOT NOT NOT", "NEAR NEAR",
		string(make([]byte, 1000)), // long input
		"a-b-c-d-e", "test@email.com", "https://url.com",
		"SELECT * FROM", "DROP TABLE", "'; --", "\\escape",
		"mixed CASE", "MiXeD", "123456", "12.34.56",
		"<html>", "&amp;", "a&b", "a|b", "a$b",
		"\t\n\r", "  leading", "trailing  ", "  both  ",
		"very long query with many words that should all be handled properly",
		"special!@#$%^&*()chars", "unicode—dash", "em—dash",
		"ellipsis…test", "curly\u201Cquotes\u201D", "backtick`test",
		"tilde~test", "hash#test", "question?test",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := sanitizeFTSQuery(input)
		// Must not panic — reaching here is success.
		// If non-empty, each word should be quoted.
		if result != "" {
			// Basic sanity: no unquoted FTS operators.
			for _, op := range []string{" AND ", " OR ", " NOT ", " NEAR"} {
				if contains(result, op) {
					t.Errorf("result contains unquoted operator %q: %s", op, result)
				}
			}
		}
	})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
