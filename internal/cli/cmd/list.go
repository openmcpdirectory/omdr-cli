package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed MCP servers",
	Long:  "Show all MCP servers installed in the local registry",
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
			fmt.Println("Run 'omdr install <server>' to install one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tCOMMAND\tUPDATED")

		// Sort by name for consistent output
		names := make([]string, 0, len(servers))
		for name := range servers {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			cfg := servers[name]
			updated := cfg.UpdatedAt.Format("2006-01-02 15:04")
			cmdStr := cfg.Command
			if len(cfg.Args) > 0 {
				cmdStr += " ..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, cmdStr, updated)
		}
		w.Flush()

		fmt.Printf("\n%d server(s) installed\n", len(servers))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
