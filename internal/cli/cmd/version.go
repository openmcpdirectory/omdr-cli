package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/branding"
	"github.com/openmcpdirectory/omdr-cli/internal/version"
	"github.com/spf13/cobra"
)

var jsonOutput bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Print the version, commit hash, and build date of the OMDR CLI.`,
	Example: `  omdr version
  omdr version --json`,
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			v := map[string]string{
				"version": version.Version,
				"commit":  version.Commit,
				"date":    version.Date,
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			enc.Encode(v)
			return
		}

		if !noBanner {
			branding.ShowBanner()
		}
		fmt.Println(version.GetFullVersion())
	},
}

func init() {
	versionCmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.AddCommand(versionCmd)
}
