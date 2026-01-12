package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/config"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun bool
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish an MCP server to the registry",
	Long:  "Publish your MCP server to the OMDR registry by reading the local mcp.json manifest",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read local mcp.json manifest
		manifestPath := "mcp.json"
		if verbose {
			fmt.Printf("Reading manifest from %s...\n", manifestPath)
		}

		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to read mcp.json: %w\nMake sure mcp.json exists in the current directory", err)
		}

		// Validate manifest locally before API call
		fmt.Println("Validating manifest...")
		manifest, err := mcpspec.ValidateManifestJSON(manifestData)
		if err != nil {
			if validationErrs, ok := err.(mcpspec.ValidationErrors); ok {
				fmt.Fprintln(os.Stderr, "Manifest validation failed:")
				for _, ve := range validationErrs {
					fmt.Fprintf(os.Stderr, "  - %s: %s\n", ve.Field, ve.Message)
					if ve.Value != nil {
						fmt.Fprintf(os.Stderr, "    Got: %v\n", ve.Value)
					}
				}
				return fmt.Errorf("manifest validation failed")
			}
			return fmt.Errorf("manifest validation failed: %w", err)
		}

		fmt.Println("✓ Manifest is valid")

		// Handle dry-run mode (skip auth check and API call)
		if dryRun {
			fmt.Println("\n✓ Dry-run successful!")
			fmt.Println("Manifest is valid and ready to publish")
			fmt.Printf("\nServer: %s/%s@%s\n", manifest.Name, manifest.Name, manifest.Version)
			fmt.Printf("Description: %s\n", manifest.Description)
			if manifest.Author != "" {
				fmt.Printf("Author: %s\n", manifest.Author)
			}
			if manifest.License != "" {
				fmt.Printf("License: %s\n", manifest.License)
			}
			fmt.Println("\nRun without --dry-run to publish")
			return nil
		}

		// Check authentication (required for actual publish)
		mgr, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("initializing config manager: %w", err)
		}

		token, err := mgr.Get(tokenKey)
		if err != nil || token == "" {
			fmt.Fprintln(os.Stderr, "Authentication required to publish servers")
			fmt.Fprintln(os.Stderr, "Run 'omdr auth login' to authenticate")
			return fmt.Errorf("not authenticated")
		}

		// Get API client
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = "http://localhost:8080"
		}

		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		// Submit to API
		fmt.Println("Publishing to registry...")

		publishReq := struct {
			Namespace   string          `json:"namespace"`
			Name        string          `json:"name"`
			Description string          `json:"description"`
			SourceURL   string          `json:"source_url"`
			Version     string          `json:"version"`
			Manifest    json.RawMessage `json:"manifest"`
		}{
			Namespace:   manifest.Name, // Use name as namespace for now
			Name:        manifest.Name,
			Description: manifest.Description,
			SourceURL:   manifest.Repository,
			Version:     manifest.Version,
			Manifest:    manifestData,
		}

		var publishResp struct {
			Server struct {
				ID          string `json:"id"`
				Namespace   string `json:"namespace"`
				Name        string `json:"name"`
				Description string `json:"description"`
				TrustScore  int    `json:"trust_score"`
			} `json:"server"`
			Version struct {
				Version string `json:"version"`
			} `json:"version"`
		}

		if err := apiClient.Post("/api/v1/servers", publishReq, &publishResp); err != nil {
			// Check for specific error types
			if apiErr, ok := err.(*client.APIError); ok {
				switch apiErr.Code {
				case "UNAUTHORIZED":
					fmt.Fprintln(os.Stderr, "Authentication token is invalid or expired")
					fmt.Fprintln(os.Stderr, "Run 'omdr auth login' to re-authenticate")
					return fmt.Errorf("authentication failed")
				case "VALIDATION_ERROR":
					fmt.Fprintln(os.Stderr, "Server validation failed:")
					fmt.Fprintf(os.Stderr, "  %s\n", apiErr.Message)
					if apiErr.Details != nil {
						if details, ok := apiErr.Details.(map[string]interface{}); ok {
							for field, msg := range details {
								fmt.Fprintf(os.Stderr, "  - %s: %v\n", field, msg)
							}
						}
					}
					return fmt.Errorf("validation failed")
				case "CONFLICT":
					fmt.Fprintln(os.Stderr, "Server already exists")
					fmt.Fprintln(os.Stderr, "Use 'omdr update' to update an existing server")
					return fmt.Errorf("server already exists")
				case "FORBIDDEN":
					fmt.Fprintln(os.Stderr, "You don't have permission to publish to this namespace")
					fmt.Fprintln(os.Stderr, "Make sure you own the namespace or use a different one")
					return fmt.Errorf("permission denied")
				}
			}

			// Check for rate limiting
			if isRateLimited, retryAfter := apiClient.IsRateLimited(err); isRateLimited {
				return fmt.Errorf("rate limited. Please retry after %v", retryAfter)
			}

			return fmt.Errorf("publish failed: %w", err)
		}

		// Display success result
		fmt.Println("\n✓ Successfully published!")
		fmt.Printf("\nServer: %s/%s@%s\n", publishResp.Server.Namespace, publishResp.Server.Name, publishResp.Version.Version)
		fmt.Printf("Description: %s\n", publishResp.Server.Description)
		fmt.Printf("Trust Score: %d/100\n", publishResp.Server.TrustScore)
		fmt.Printf("Server ID: %s\n", publishResp.Server.ID)

		// Display registry URL
		registryURL := fmt.Sprintf("%s/servers/%s/%s", apiURL, publishResp.Server.Namespace, publishResp.Server.Name)
		fmt.Printf("\nView in registry: %s\n", registryURL)

		fmt.Println("\nUsers can now install your server with:")
		fmt.Printf("  omdr install %s/%s\n", publishResp.Server.Namespace, publishResp.Server.Name)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate manifest without publishing")
}
