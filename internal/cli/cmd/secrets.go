package cmd

import (
	"fmt"
	"os"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/secret"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secure secrets for installed MCP servers",
	Long: `Manage secure secrets (API keys, tokens) for installed MCP servers.
Secrets are stored in your operating system's native keychain (Windows Credential Manager, macOS Keychain, etc.)
and are injected as environment variables when the server is run via 'omdr run'.`,
	Example: `  omdr secrets set @namespace/server API_KEY my-secret-value
  omdr secrets list @namespace/server
  omdr secrets delete @namespace/server API_KEY`,
}

// setCmd: omdr secrets set <server> <key> <value>
var secretsSetCmd = &cobra.Command{
	Use:   "set <server> <key> <value>",
	Short: "Set a secret environment variable for a server",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		key := args[1]
		value := args[2]

		// 1. Load registry
		reg, err := registry.NewLocalRegistry()
		if err != nil {
			return fmt.Errorf("initializing registry: %w", err)
		}

		// 2. Get existing config
		config, err := reg.Get(serverName)
		if err != nil {
			return fmt.Errorf("server not found: %s", serverName)
		}

		// 3. Store secret in keychain
		keychainUser := fmt.Sprintf("omdr-env:%s:%s", serverName, key)
		if err := secret.Store(secret.ServiceName, keychainUser, value); err != nil {
			return fmt.Errorf("storing secret: %w", err)
		}

		// 4. Update registry list if not present
		exists := false
		for _, s := range config.Secrets {
			if s == key {
				exists = true
				break
			}
		}

		if !exists {
			config.Secrets = append(config.Secrets, key)
			if err := reg.Register(serverName, *config); err != nil {
				return fmt.Errorf("updating registry: %w", err)
			}
		}

		fmt.Printf("Secret '%s' set for server '%s'\n", key, serverName)
		return nil
	},
}

// listCmd: omdr secrets list <server>
var secretsListCmd = &cobra.Command{
	Use:   "list <server>",
	Short: "List configured secret keys for a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]

		reg, err := registry.NewLocalRegistry()
		if err != nil {
			return fmt.Errorf("initializing registry: %w", err)
		}

		config, err := reg.Get(serverName)
		if err != nil {
			return fmt.Errorf("server not found: %s", serverName)
		}

		if len(config.Secrets) == 0 {
			fmt.Printf("No secrets configured for server '%s'\n", serverName)
			return nil
		}

		fmt.Printf("Secrets for '%s':\n", serverName)
		for _, s := range config.Secrets {
			fmt.Println("  - " + s)
		}
		return nil
	},
}

// deleteCmd: omdr secrets delete <server> <key>
var secretsDeleteCmd = &cobra.Command{
	Use:   "delete <server> <key>",
	Short: "Delete a secret environment variable",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		key := args[1]

		reg, err := registry.NewLocalRegistry()
		if err != nil {
			return fmt.Errorf("initializing registry: %w", err)
		}

		config, err := reg.Get(serverName)
		if err != nil {
			return fmt.Errorf("server not found: %s", serverName)
		}

		// Remove from keychain
		keychainUser := fmt.Sprintf("omdr-env:%s:%s", serverName, key)
		if err := secret.Delete(secret.ServiceName, keychainUser); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete from keychain: %v\n", err)
		}

		// Update registry
		newSecrets := []string{}
		found := false
		for _, s := range config.Secrets {
			if s == key {
				found = true
				continue
			}
			newSecrets = append(newSecrets, s)
		}

		if found {
			config.Secrets = newSecrets
			if err := reg.Register(serverName, *config); err != nil {
				return fmt.Errorf("updating registry: %w", err)
			}
			fmt.Printf("Secret '%s' deleted for server '%s'\n", key, serverName)
		} else {
			fmt.Printf("Secret '%s' not found for server '%s'\n", key, serverName)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)
}
