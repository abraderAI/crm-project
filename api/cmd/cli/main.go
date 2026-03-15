// Package main is the entry point for the deft CLI binary.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cliauth "github.com/abraderAI/crm-project/api/internal/cli/auth"
	cliconfig "github.com/abraderAI/crm-project/api/internal/cli/config"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var configPath string
	var jsonOutput bool
	var orgOverride string
	var limitOverride int

	root := &cobra.Command{
		Use:           "deft",
		Short:         "DEFT CRM AI-powered CLI",
		Long:          "Interact with the DEFT CRM platform using natural language or direct commands.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&configPath, "config", "", "path to config file (default ~/.deft-cli.yaml)")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output raw JSON")
	root.PersistentFlags().StringVar(&orgOverride, "org", "", "override default org")
	root.PersistentFlags().IntVar(&limitOverride, "limit", 0, "limit results for pagination")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newLoginCmd(&configPath))
	root.AddCommand(newLogoutCmd(&configPath))
	root.AddCommand(newWhoamiCmd(&configPath))
	root.AddCommand(newChatCmd(&configPath, &jsonOutput, &orgOverride))
	root.AddCommand(newAskCmd(&configPath, &jsonOutput, &orgOverride))

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("deft version %s\n", Version)
		},
	}
}

func newLoginCmd(configPath *string) *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the DEFT API",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cliconfig.Load(*configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			store := cliauth.NewStore(&memoryKeyring{})
			var creds *cliauth.Credentials

			if token != "" {
				creds = &cliauth.Credentials{Token: token}
			} else {
				fmt.Print("Enter API key: ")
				reader := bufio.NewReader(os.Stdin)
				apiKey, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading API key: %w", err)
				}
				apiKey = strings.TrimSpace(apiKey)
				if apiKey == "" {
					return fmt.Errorf("API key cannot be empty")
				}
				creds = &cliauth.Credentials{APIKey: apiKey}
			}

			// Validate credentials against the API.
			if err := cliauth.ValidateCredentials(cfg.APIURL, creds); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			if err := store.Save(creds); err != nil {
				return fmt.Errorf("saving credentials: %w", err)
			}

			// Also save API key in config for persistence.
			if creds.APIKey != "" {
				cfg.APIKey = creds.APIKey
				if err := cliconfig.Save(*configPath, cfg); err != nil {
					return fmt.Errorf("saving config: %w", err)
				}
			}

			fmt.Println("Login successful!")
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "JWT token for authentication")
	return cmd
}

func newLogoutCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			store := cliauth.NewStore(&memoryKeyring{})
			if err := store.Clear(); err != nil {
				return fmt.Errorf("clearing credentials: %w", err)
			}

			// Clear API key from config.
			cfg, err := cliconfig.Load(*configPath)
			if err == nil {
				cfg.APIKey = ""
				_ = cliconfig.Save(*configPath, cfg)
			}

			fmt.Println("Logged out successfully.")
			return nil
		},
	}
}

func newWhoamiCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current authentication state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cliconfig.Load(*configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			store := cliauth.NewStore(&memoryKeyring{})
			creds, err := store.Load()
			if err != nil {
				return fmt.Errorf("loading credentials: %w", err)
			}

			// If keyring is empty, check config.
			if creds.IsEmpty() && cfg.APIKey != "" {
				creds = &cliauth.Credentials{APIKey: cfg.APIKey}
			}

			if creds.IsEmpty() {
				fmt.Println("Not authenticated. Run 'deft login' to authenticate.")
				return nil
			}

			fmt.Printf("API URL: %s\n", cfg.APIURL)
			if cfg.DefaultOrg != "" {
				fmt.Printf("Default Org: %s\n", cfg.DefaultOrg)
			}
			if creds.APIKey != "" {
				masked := creds.APIKey
				if len(masked) > 8 {
					masked = masked[:8] + "..."
				}
				fmt.Printf("Auth: API Key (%s)\n", masked)
			} else if creds.Token != "" {
				fmt.Println("Auth: JWT Token")
			}
			return nil
		},
	}
}

func newChatCmd(configPath *string, jsonOutput *bool, orgOverride *string) *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Enter interactive AI chat mode",
		Long:  "Start an interactive REPL for natural language CRM queries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, err := setupAgent(*configPath, *orgOverride)
			if err != nil {
				return err
			}
			// In a full implementation this would call chat.New(ag, os.Stdin, os.Stdout).RunREPL()
			// For now, print a message indicating that the LLM provider must be configured.
			fmt.Println("Interactive chat requires an LLM provider to be configured.")
			fmt.Println("Set DEFT_LLM_PROVIDER environment variable to enable.")
			return nil
		},
	}
}

func newAskCmd(configPath *string, jsonOutput *bool, orgOverride *string) *cobra.Command {
	return &cobra.Command{
		Use:   "ask [query]",
		Short: "Ask a one-shot natural language query",
		Long:  "Send a single natural language query to the AI assistant.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, err := setupAgent(*configPath, *orgOverride)
			if err != nil {
				return err
			}
			fmt.Println("One-shot query requires an LLM provider to be configured.")
			fmt.Println("Set DEFT_LLM_PROVIDER environment variable to enable.")
			return nil
		},
	}
}

// setupAgent loads config, credentials, and creates the API client.
func setupAgent(configPath, orgOverride string) (*cliconfig.Config, *cliauth.Credentials, string, error) {
	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading config: %w", err)
	}

	store := cliauth.NewStore(&memoryKeyring{})
	creds, err := store.Load()
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading credentials: %w", err)
	}

	if creds.IsEmpty() && cfg.APIKey != "" {
		creds = &cliauth.Credentials{APIKey: cfg.APIKey}
	}

	org := cfg.DefaultOrg
	if orgOverride != "" {
		org = orgOverride
	}

	return cfg, creds, org, nil
}

// memoryKeyring is a simple in-memory keyring for environments without OS keyring support.
// In production, this would be replaced with go-keyring.
type memoryKeyring struct {
	data map[string]string
}

func (m *memoryKeyring) Set(service, account, password string) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[service+"/"+account] = password
	return nil
}

func (m *memoryKeyring) Get(service, account string) (string, error) {
	if m.data == nil {
		return "", fmt.Errorf("not found")
	}
	v, ok := m.data[service+"/"+account]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}

func (m *memoryKeyring) Delete(service, account string) error {
	if m.data != nil {
		delete(m.data, service+"/"+account)
	}
	return nil
}
