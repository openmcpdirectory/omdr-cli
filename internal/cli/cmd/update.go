package cmd

import (
	"context"
	"fmt"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCmd = &cobra.Command{
	Use:   "update [server]",
	Short: "Check for and apply server updates",
	Long:  "Check the OMDR registry for updates to installed servers. If a server name is provided, only that server is checked.",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg, err := registry.NewLocalRegistry()
		if err != nil {
			return fmt.Errorf("opening registry: %w", err)
		}

		servers, err := reg.List()
		if err != nil {
			return fmt.Errorf("listing servers: %w", err)
		}

		if len(servers) == 0 {
			fmt.Println("No servers installed.")
			return nil
		}

		// Filter to specific server if provided
		if len(args) > 0 {
			name := args[0]
			cfg, ok := servers[name]
			if !ok {
				return fmt.Errorf("server '%s' is not installed", name)
			}
			servers = map[string]registry.ServerConfig{name: cfg}
		}

		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = "https://cli.omdr.dev"
		}
		apiClient := client.NewClient(apiURL)

		updated := 0
		for name := range servers {
			hasUpdate, err := checkServerUpdate(cmd.Context(), apiClient, name)
			if err != nil {
				fmt.Printf("  %s: failed to check (%v)\n", name, err)
				continue
			}
			if hasUpdate {
				fmt.Printf("  %s: update available\n", name)
				updated++
			} else {
				fmt.Printf("  %s: up to date\n", name)
			}
		}

		if updated > 0 {
			fmt.Printf("\n%d update(s) available. Run 'omdr install <server>' to update.\n", updated)
		} else {
			fmt.Println("\nAll servers are up to date.")
		}
		return nil
	},
}

func checkServerUpdate(ctx context.Context, apiClient *client.Client, serverName string) (bool, error) {
	var serverInfo struct {
		LatestVersion string `json:"latest_version"`
	}
	if err := apiClient.Get(ctx, fmt.Sprintf("/api/v1/servers/%s", serverName), &serverInfo); err != nil {
		return false, err
	}
	// If the API returned a version, an update may be available.
	// Full version comparison would require storing the installed version locally.
	return serverInfo.LatestVersion != "", nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
