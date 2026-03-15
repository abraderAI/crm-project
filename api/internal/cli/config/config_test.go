package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefault(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.APIURL)
	assert.Empty(t, cfg.APIKey)
	assert.Empty(t, cfg.DefaultOrg)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `api_url: https://api.example.com
api_key: test-key-123
default_org: my-org
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", cfg.APIURL)
	assert.Equal(t, "test-key-123", cfg.APIKey)
	assert.Equal(t, "my-org", cfg.DefaultOrg)
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("DEFT_API_URL", "https://env.example.com")
	t.Setenv("DEFT_API_KEY", "env-key")
	t.Setenv("DEFT_ORG", "env-org")

	cfg, err := Load("/nonexistent/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, "https://env.example.com", cfg.APIURL)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "env-org", cfg.DefaultOrg)
}

func TestLoadEnvOverridesFileValues(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `api_url: https://file.example.com
api_key: file-key
default_org: file-org
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))
	t.Setenv("DEFT_API_URL", "https://env.example.com")

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "https://env.example.com", cfg.APIURL)
	assert.Equal(t, "file-key", cfg.APIKey)
	assert.Equal(t, "file-org", cfg.DefaultOrg)
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.yaml")
	// Binary content that isn't valid YAML.
	require.NoError(t, os.WriteFile(cfgPath, []byte("api_url: [invalid\n  broken"), 0o600))

	_, err := Load(cfgPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config file")
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")

	cfg := &Config{
		APIURL:     "https://saved.example.com",
		APIKey:     "saved-key",
		DefaultOrg: "saved-org",
	}

	err := Save(cfgPath, cfg)
	require.NoError(t, err)

	loaded, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, cfg.APIURL, loaded.APIURL)
	assert.Equal(t, cfg.APIKey, loaded.APIKey)
	assert.Equal(t, cfg.DefaultOrg, loaded.DefaultOrg)
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.yaml")

	err := Save(cfgPath, &Config{APIURL: "http://test"})
	require.NoError(t, err)

	_, err = os.Stat(cfgPath)
	assert.NoError(t, err)
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	require.NoError(t, err)
	assert.Contains(t, path, DefaultConfigFileName)
}

func TestLoadEmptyPath(t *testing.T) {
	// When path is empty, it should try default path and not error.
	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestApplyEnvOverridesPartial(t *testing.T) {
	t.Setenv("DEFT_API_KEY", "partial-key")

	cfg := &Config{APIURL: "http://original"}
	result := applyEnvOverrides(cfg)
	assert.Equal(t, "http://original", result.APIURL)
	assert.Equal(t, "partial-key", result.APIKey)
}

func TestSaveEmptyPath(t *testing.T) {
	// Save with empty path should use default path.
	cfg := &Config{APIURL: "http://test"}
	err := Save("", cfg)
	// This may succeed or fail depending on home dir permissions, but should not panic.
	_ = err
}
