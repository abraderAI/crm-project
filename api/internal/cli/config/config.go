// Package config provides CLI configuration loading from file and environment variables.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigFileName is the default config file name.
	DefaultConfigFileName = ".deft-cli.yaml"

	// EnvAPIURL overrides the API URL from config.
	EnvAPIURL = "DEFT_API_URL"
	// EnvAPIKey overrides the API key from config.
	EnvAPIKey = "DEFT_API_KEY"
	// EnvOrg overrides the default org from config.
	EnvOrg = "DEFT_ORG"
)

// Config holds CLI configuration values.
type Config struct {
	APIURL     string `yaml:"api_url" json:"api_url"`
	APIKey     string `yaml:"api_key" json:"api_key"`
	DefaultOrg string `yaml:"default_org" json:"default_org"`
}

// DefaultConfigPath returns the default config file path (~/.deft-cli.yaml).
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigFileName), nil
}

// Load reads the config file at the given path and applies environment variable overrides.
// If path is empty, uses the default config path.
// Missing config file is not an error; defaults are returned.
func Load(path string) (*Config, error) {
	cfg := &Config{
		APIURL: "http://localhost:8080",
	}

	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return applyEnvOverrides(cfg), nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return applyEnvOverrides(cfg), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return applyEnvOverrides(cfg), nil
}

// Save writes the config to the given path.
func Save(path string, cfg *Config) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return fmt.Errorf("getting default config path: %w", err)
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) *Config {
	if v := os.Getenv(EnvAPIURL); v != "" {
		cfg.APIURL = v
	}
	if v := os.Getenv(EnvAPIKey); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv(EnvOrg); v != "" {
		cfg.DefaultOrg = v
	}
	return cfg
}
