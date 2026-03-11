package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"SERVER_PORT", "SERVER_HOST", "SQLITE_PATH", "CLERK_SECRET_KEY",
		"CLERK_PUBLISHABLE_KEY", "CLERK_ISSUER_URL", "LOG_LEVEL",
		"CORS_ORIGINS", "UPLOAD_DIR", "UPLOAD_MAX_SIZE", "OTEL_ENDPOINT",
		"OTEL_ENABLED", "RBAC_POLICY_PATH",
	}
	for _, key := range envVars {
		os.Unsetenv(key)
	}
}

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "0.0.0.0", cfg.ServerHost)
	assert.Equal(t, "data/deft.db", cfg.SQLitePath)
	assert.Equal(t, "", cfg.ClerkSecretKey)
	assert.Equal(t, "", cfg.ClerkPublishKey)
	assert.Equal(t, "", cfg.ClerkIssuerURL)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.CORSOrigins)
	assert.Equal(t, "uploads", cfg.UploadDir)
	assert.Equal(t, int64(104857600), cfg.UploadMaxSize)
	assert.Equal(t, "", cfg.OTelEndpoint)
	assert.False(t, cfg.OTelEnabled)
	assert.Equal(t, "config/rbac-policy.yaml", cfg.RBACPolicyPath)
}

func TestLoadFromEnv(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SQLITE_PATH", "/tmp/test.db")
	t.Setenv("CLERK_SECRET_KEY", "sk_test")
	t.Setenv("CLERK_PUBLISHABLE_KEY", "pk_test")
	t.Setenv("CLERK_ISSUER_URL", "https://clerk.test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("CORS_ORIGINS", "https://example.com, https://other.com")
	t.Setenv("UPLOAD_DIR", "/tmp/uploads")
	t.Setenv("UPLOAD_MAX_SIZE", "52428800")
	t.Setenv("OTEL_ENDPOINT", "http://otel:4317")
	t.Setenv("OTEL_ENABLED", "true")
	t.Setenv("RBAC_POLICY_PATH", "/etc/rbac.yaml")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, 9090, cfg.ServerPort)
	assert.Equal(t, "127.0.0.1", cfg.ServerHost)
	assert.Equal(t, "/tmp/test.db", cfg.SQLitePath)
	assert.Equal(t, "sk_test", cfg.ClerkSecretKey)
	assert.Equal(t, "pk_test", cfg.ClerkPublishKey)
	assert.Equal(t, "https://clerk.test", cfg.ClerkIssuerURL)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, []string{"https://example.com", "https://other.com"}, cfg.CORSOrigins)
	assert.Equal(t, "/tmp/uploads", cfg.UploadDir)
	assert.Equal(t, int64(52428800), cfg.UploadMaxSize)
	assert.Equal(t, "http://otel:4317", cfg.OTelEndpoint)
	assert.True(t, cfg.OTelEnabled)
	assert.Equal(t, "/etc/rbac.yaml", cfg.RBACPolicyPath)
}

func TestValidateInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"too high port", 65536},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ServerPort: tt.port, SQLitePath: "test.db", LogLevel: "info"}
			assert.Error(t, cfg.Validate())
		})
	}
}

func TestValidateValidPorts(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"min port", 1},
		{"common port", 8080},
		{"max port", 65535},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ServerPort: tt.port, SQLitePath: "test.db", LogLevel: "info"}
			assert.NoError(t, cfg.Validate())
		})
	}
}

func TestValidateEmptySQLitePath(t *testing.T) {
	cfg := &Config{ServerPort: 8080, SQLitePath: "", LogLevel: "info"}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite path")
}

func TestValidateNegativeUploadMaxSize(t *testing.T) {
	cfg := &Config{ServerPort: 8080, SQLitePath: "test.db", LogLevel: "info", UploadMaxSize: -1}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload max size")
}

func TestValidateInvalidLogLevel(t *testing.T) {
	cfg := &Config{ServerPort: 8080, SQLitePath: "test.db", LogLevel: "verbose"}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "log level")
}

func TestValidateAllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}
	for _, lvl := range levels {
		t.Run(lvl, func(t *testing.T) {
			cfg := &Config{ServerPort: 8080, SQLitePath: "test.db", LogLevel: lvl}
			assert.NoError(t, cfg.Validate())
		})
	}
}

func TestAddress(t *testing.T) {
	cfg := &Config{ServerHost: "127.0.0.1", ServerPort: 9090}
	assert.Equal(t, "127.0.0.1:9090", cfg.Address())

	cfg2 := &Config{ServerHost: "0.0.0.0", ServerPort: 8080}
	assert.Equal(t, "0.0.0.0:8080", cfg2.Address())
}

func TestGetEnvInt_InvalidValue(t *testing.T) {
	t.Setenv("TEST_INT", "notanumber")
	result := getEnvInt("TEST_INT", 42)
	assert.Equal(t, 42, result)
}

func TestGetEnvInt64_InvalidValue(t *testing.T) {
	t.Setenv("TEST_INT64", "notanumber")
	result := getEnvInt64("TEST_INT64", 99)
	assert.Equal(t, int64(99), result)
}

func TestGetEnvBool_InvalidValue(t *testing.T) {
	t.Setenv("TEST_BOOL", "maybe")
	result := getEnvBool("TEST_BOOL", true)
	assert.True(t, result)
}

func TestGetEnvBool_ValidValues(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Setenv("TEST_BOOL_V", tt.input)
			assert.Equal(t, tt.want, getEnvBool("TEST_BOOL_V", !tt.want))
		})
	}
}

func TestGetEnvSlice_Empty(t *testing.T) {
	t.Setenv("TEST_SLICE", "")
	result := getEnvSlice("TEST_SLICE", []string{"default"})
	assert.Equal(t, []string{"default"}, result)
}

func TestGetEnvSlice_WithSpaces(t *testing.T) {
	t.Setenv("TEST_SLICE", " a , b , c ")
	result := getEnvSlice("TEST_SLICE", nil)
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestGetEnvSlice_AllEmpty(t *testing.T) {
	t.Setenv("TEST_SLICE", "  ,  ,  ")
	result := getEnvSlice("TEST_SLICE", []string{"fallback"})
	assert.Equal(t, []string{"fallback"}, result)
}

func TestLoadInvalidPort(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_PORT", "99999")
	_, err := Load()
	assert.Error(t, err)
}

func TestLoadInvalidLogLevel(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("LOG_LEVEL", "TRACE")
	_, err := Load()
	assert.Error(t, err)
}

// FuzzGetEnvInt exercises integer parsing with random inputs.
func FuzzGetEnvInt(f *testing.F) {
	seeds := []string{
		"", "0", "1", "-1", "8080", "65535", "99999",
		"abc", "1.5", " 42 ", "2147483647", "-2147483648",
		"not-a-number", "12abc", "0x1F", "+100",
		"000", "007", "9999999999999",
		"\t5", "5\n", " ", "NULL",
		"1e5", "NaN", "Inf",
		"2147483648", "-2147483649",
		"00000000000000001", "true", "false",
		"0b1010", "0o77", "0xDEAD",
		"3.14159", "-0", "+0",
		"--1", "++1", "1 2 3",
		"①②③", "٣٢١", "¹²³",
		"MAX_INT", "MIN_INT", "OVERFLOW",
		"999999999999999999999999999999",
		"\r\n42", "42\r\n", "\t\t42\t\t",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		t.Setenv("FUZZ_INT", input)
		result := getEnvInt("FUZZ_INT", 42)
		assert.NotPanics(t, func() { _ = result })
	})
}

// FuzzGetEnvBool exercises boolean parsing with random inputs.
func FuzzGetEnvBool(f *testing.F) {
	seeds := []string{
		"", "true", "false", "1", "0", "yes", "no",
		"TRUE", "FALSE", "True", "False",
		"t", "f", "T", "F",
		"on", "off", "ON", "OFF",
		"2", "-1", "maybe", "null", "nil",
		"truetrue", "falsefalse",
		" true", "true ", " true ",
		"\ttrue", "true\n",
		"yep", "nope", "si", "non",
		"👍", "👎", "✓", "✗",
		"0.0", "1.0", "-0",
		"enabled", "disabled",
		"ENABLED", "DISABLED",
		"active", "inactive",
		"y", "n", "Y", "N",
		"oui", "non", "ja", "nein",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		t.Setenv("FUZZ_BOOL", input)
		result := getEnvBool("FUZZ_BOOL", false)
		assert.NotPanics(t, func() { _ = result })
	})
}
