package cmd

import (
	"fmt"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/installer"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <server>",
	Short: "Uninstall an MCP server",
	Long:  "Remove an MCP server from the local registry and all detected client configurations",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]

		reg, err := registry.NewLocalRegistry()
		if err != nil {
			return fmt.Errorf("opening registry: %w", err)
		}

		// Verify server exists
		if _, err := reg.Get(serverName); err != nil {
			return fmt.Errorf("server '%s' is not installed", serverName)
		}

		// Remove from all detected MCP client configs
		d := detector.NewDetector()
		clients, _ := d.DetectClients()

		p := installer.NewConfigPatcher()
		removedFrom := 0
		for _, client := range clients {
			if err := p.RemoveServerFromConfig(client, serverName); err != nil {
				fmt.Printf("  Warning: failed to remove from %s: %v\n", client.Name, err)
			} else {
				removedFrom++
			}
		}

		// Remove from local registry
		if err := reg.Unregister(serverName); err != nil {
			return fmt.Errorf("removing from registry: %w", err)
		}

		fmt.Printf("Uninstalled %s\n", serverName)
		if removedFrom > 0 {
			fmt.Printf("Removed from %d client configuration(s)\n", removedFrom)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
