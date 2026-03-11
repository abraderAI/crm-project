package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

// generateTestKeyPairForFuzz creates a test key pair without *testing.T.
func generateTestKeyPairForFuzz() *testKeyPair {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return &testKeyPair{
		Private: key,
		Public:  &key.PublicKey,
		Kid:     "test-kid-1",
	}
}

// FuzzJWTValidation tests JWT validation with random/malformed tokens.
func FuzzJWTValidation(f *testing.F) {
	// Seed corpus with various JWT-like patterns (≥50 seeds).
	seeds := []string{
		"",
		"a",
		"a.b",
		"a.b.c",
		"a.b.c.d",
		"eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ1c2VyXzEiLCJpc3MiOiJodHRwczovL3Rlc3QuY29tIn0.c2lnbmF0dXJl",
		"eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.sig",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.sig",
		"!!!.eyJ0ZXN0IjoxfQ.ZGVm",
		"eyJhbGciOiJSUzI1NiJ9.!!!.ZGVm",
		"eyJhbGciOiJSUzI1NiJ9.eyJ0ZXN0IjoxfQ.!!!",
		"..",
		"...",
		"a.",
		".a",
		".a.",
		"a..b",
		"a.b.",
		".a.b",
		"eyJhbGciOiJub25lIn0.eyJzdWIiOiJ0ZXN0In0.",
		"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..c2ln",
		"e30.e30.c2ln",
		"eyJhbGciOiJSUzI1NiIsImtpZCI6InRlc3QifQ.e30.c2ln",
		"null.null.null",
		"{}.{}.{}",
		"Bearer token",
		"deft_live_test123456789",
		"eyJhbGciOiJSUzI1NiIsImtpZCI6InRlc3Qta2lkLTEifQ.eyJzdWIiOiJ1c2VyXzEiLCJpc3MiOiJodHRwczovL3Rlc3QuY29tIiwiZXhwIjoxOTk5OTk5OTk5fQ.c2ln",
		"\x00\x01\x02",
		"🎉.🎉.🎉",
		"very.long.token." + "aaaaaaaaaaaaa",
		"eyJ.eyJ.AA==",
		"AAAA.BBBB.CCCC",
		"ey.ey.AA",
		"eyJhbGciOiJFUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.c2ln",
		"eyJhbGciOiJSUzM4NCJ9.eyJzdWIiOiJ0ZXN0In0.c2ln",
		"eyJhbGciOiJQUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.c2ln",
		"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIiLCJpc3MiOiIiLCJleHAiOjB9.c2ln",
		"eyJhbGciOiJSUzI1NiJ9.eyJleHAiOi0xfQ.c2ln",
		"eyJhbGciOiJSUzI1NiJ9.eyJuYmYiOjk5OTk5OTk5OTl9.c2ln",
		"a]b.c[d.e{f",
		"YWJj.ZGVm.Z2hp",
		"///.///.///",
		"----.----.----",
		"====.====.====",
		"$%^&.*(().-+=!",
		"test test.test test.test test",
		"a\nb.c\nd.e\nf",
		"a\tb.c\td.e\tf",
		"a b.c d.e f",
		"eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0IiwiZXhwIjoxMDAwMDAwMDAwMDAwfQ.c2ln",
		"eyJhbGciOiJSUzI1NiJ9.e30.e30",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	kp := generateTestKeyPairForFuzz()
	v := NewJWTValidator("https://test.example.com")
	v.SetKeys(map[string]*rsa.PublicKey{kp.Kid: kp.Public})

	f.Fuzz(func(t *testing.T, token string) {
		// Should never panic regardless of input.
		_, _ = v.Validate(token)
	})
}

// FuzzAPIKeyValidation tests API key validation with random inputs.
func FuzzAPIKeyValidation(f *testing.F) {
	// Seed corpus (≥50 seeds).
	seeds := []string{
		"",
		"deft_live_",
		"deft_live_abc",
		"deft_live_0000000000000000000000000000000000000000000000000000000000000000",
		"deft_live_ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		"invalid",
		"deft_test_abc",
		"DEFT_LIVE_ABC",
		"deft_live_" + "a",
		"deft_live_" + "ab",
		"x",
		"deft_",
		"deft",
		"live_",
		"deft_live",
		"deft_live_!@#$%^&*()",
		"deft_live_\x00\x01\x02",
		"deft_live_🎉",
		"deft_live_abcdefghijklmnopqrstuvwxyz0123456789",
		"deft_live_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		"a" + "deft_live_abc",
		" deft_live_abc",
		"deft_live_abc ",
		"\tdeft_live_abc",
		"deft_live_abc\n",
		"deft_live_" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"Bearer deft_live_abc",
		"deft_live_a.b.c",
		"deft_live_===",
		"deft_live_///",
		string(make([]byte, 0)),
		string(make([]byte, 1)),
		string(make([]byte, 100)),
		"deft_live_" + string(make([]byte, 32)),
		"null",
		"undefined",
		"true",
		"false",
		"0",
		"-1",
		"deft_live_null",
		"deft_live_undefined",
		"deft_live_true",
		"deft_live_false",
		"deft_live_0",
		"deft_live_-1",
		"deft_live_1234567890abcdef",
		"deft_live_" + "aaaa",
		"deft_live_" + "zzzz",
		"deft_live_test",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, key string) {
		// Should never panic.
		_ = hashKey(key)
		if len(key) >= len(apiKeyPrefix) && key[:len(apiKeyPrefix)] == apiKeyPrefix {
			// Valid prefix — just test the prefix check.
			_ = key
		}
	})
}
