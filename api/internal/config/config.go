// Package config provides environment-based configuration loading for the API server.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration values for the API server.
type Config struct {
	// Server settings.
	ServerPort int
	ServerHost string

	// Database settings.
	SQLitePath string

	// Auth settings (Clerk).
	ClerkSecretKey  string
	ClerkPublishKey string
	ClerkIssuerURL  string

	// Logging.
	LogLevel string

	// CORS.
	CORSOrigins []string

	// File uploads.
	UploadDir     string
	UploadMaxSize int64

	// OpenTelemetry.
	OTelEndpoint string
	OTelEnabled  bool

	// RBAC policy path.
	RBACPolicyPath string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:      getEnvInt("SERVER_PORT", 8080),
		ServerHost:      getEnv("SERVER_HOST", "0.0.0.0"),
		SQLitePath:      getEnv("SQLITE_PATH", "data/deft.db"),
		ClerkSecretKey:  getEnv("CLERK_SECRET_KEY", ""),
		ClerkPublishKey: getEnv("CLERK_PUBLISHABLE_KEY", ""),
		ClerkIssuerURL:  getEnv("CLERK_ISSUER_URL", ""),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		CORSOrigins:     getEnvSlice("CORS_ORIGINS", []string{"http://localhost:3000"}),
		UploadDir:       getEnv("UPLOAD_DIR", "uploads"),
		UploadMaxSize:   getEnvInt64("UPLOAD_MAX_SIZE", 104857600), // 100MB
		OTelEndpoint:    getEnv("OTEL_ENDPOINT", ""),
		OTelEnabled:     getEnvBool("OTEL_ENABLED", false),
		RBACPolicyPath:  getEnv("RBAC_POLICY_PATH", "config/rbac-policy.yaml"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that required configuration values are present and valid.
func (c *Config) Validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d (must be 1-65535)", c.ServerPort)
	}

	if c.SQLitePath == "" {
		return fmt.Errorf("sqlite path must not be empty")
	}

	if c.UploadMaxSize < 0 {
		return fmt.Errorf("upload max size must be non-negative")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %q (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

// Address returns the server listen address (host:port).
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvInt64(key string, fallback int64) int64 {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvSlice(key string, fallback []string) []string {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
