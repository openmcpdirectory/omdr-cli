package cmd

import (
	"fmt"
	"os"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/config"
	clilogger "github.com/openmcpdirectory/omdr-cli/internal/cli/logger"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/proxy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	authMode string
)

var proxyCmd = &cobra.Command{
	Use:    "proxy <server>",
	Short:  "MCP protocol bridge (internal use)",
	Long:   "Acts as a bridge between MCP clients and hosted OMDR servers. This command is typically invoked automatically by MCP clients and not meant for direct use.",
	Hidden: true, // Hide from help output
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]

		// Get API key from environment or config
		apiKey := os.Getenv("OMDR_API_KEY")
		if apiKey == "" {
			// Try to get from config
			mgr, err := config.NewManager()
			if err != nil {
				return fmt.Errorf("initializing config manager: %w", err)
			}

			apiKey, err = mgr.Get("auth.token")
			if err != nil || apiKey == "" {
				return fmt.Errorf("authentication required: OMDR_API_KEY not set and no token in config. Run 'omdr auth login'")
			}
		}

		// Get guard URL
		guardURL := viper.GetString("guard_url")
		if guardURL == "" {
			guardURL = "https://guard.omdr.dev"
		}

		clilogger.Verbose("Starting proxy for server: %s", serverName)
		clilogger.Verbose("Guard URL: %s", guardURL)
		clilogger.Verbose("Auth mode: %s", authMode)

		// Create and start proxy server
		server := proxy.NewServer(proxy.Config{
			ServerName: serverName,
			APIKey:     apiKey,
			GuardURL:   guardURL,
			AuthMode:   authMode,
		})

		// Serve stdio (blocks until stdin closes or error)
		if err := server.ServeStdio(); err != nil {
			return fmt.Errorf("proxy error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)
	proxyCmd.Flags().StringVar(&authMode, "auth-mode", "full_proxy", "Authentication mode: auth_only or full_proxy")
}
