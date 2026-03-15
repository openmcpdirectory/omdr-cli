package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate an OMDR/MCP manifest",
	Long:  "Validate an omdr.json, omdr.toml, or mcp.json manifest. Defaults to auto-detecting in the current directory.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(_ *cobra.Command, args []string) error {
	var (
		manifest *mcpspec.MCPManifest
		err      error
	)

	if len(args) == 1 {
		manifest, err = mcpspec.LoadManifestFrom(args[0])
	} else {
		dir, _ := os.Getwd()
		manifest, err = mcpspec.LoadManifest(dir)
	}
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	fmt.Printf("Loaded %s\n", manifest.SourceFile)

	if err := mcpspec.ValidateManifest(manifest); err != nil {
		if validationErrs, ok := err.(mcpspec.ValidationErrors); ok {
			fmt.Fprintln(os.Stderr, "Validation failed:")
			for _, ve := range validationErrs {
				fmt.Fprintf(os.Stderr, "  ✗ %s: %s\n", ve.Field, ve.Message)
				if ve.Value != nil {
					fmt.Fprintf(os.Stderr, "    Got: %v\n", ve.Value)
				}
			}
			return fmt.Errorf("%d validation error(s)", len(validationErrs))
		}
		return err
	}

	fmt.Println("✓ Manifest is valid")
	fmt.Printf("  Name:    %s\n", manifest.Name)
	fmt.Printf("  Version: %s\n", manifest.Version)
	fmt.Printf("  Runtime: %s\n", manifest.Runtime.Type)

	if manifest.OMDR != nil {
		fmt.Printf("  Deploy:  %s\n", manifest.OMDR.Deployment)
		if manifest.OMDR.Pricing != nil {
			fmt.Printf("  Pricing: %s\n", manifest.OMDR.Pricing.Model)
		}
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(manifest, "", "  ")
		fmt.Println(string(data))
	}

	return nil
}
