package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/config"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun          bool
	deploymentModel string
	artifactPath    string
	githubRepo      string
	githubToken     string
	selfHostedURL   string
	pricingModel    string
	pricePerCall    int64
	monthlyPrice    int64
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
			apiURL = "https://cli.omdr.dev"
		}

		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		// Submit to API
		fmt.Println("Publishing to registry...")

		publishReq := struct {
			Namespace       string          `json:"namespace"`
			Name            string          `json:"name"`
			Description     string          `json:"description"`
			SourceURL       string          `json:"source_url"`
			Version         string          `json:"version"`
			Manifest        json.RawMessage `json:"manifest"`
			DeploymentModel string          `json:"deployment_model,omitempty"`
			GitHubURL       string          `json:"github_url,omitempty"`
			GitHubToken     string          `json:"github_token,omitempty"`
			ArtifactType    string          `json:"artifact_type,omitempty"`
			ArtifactURL     string          `json:"artifact_url,omitempty"`
			PricingModel    string          `json:"pricing_model,omitempty"`
			PricePerCall    int64           `json:"price_per_call,omitempty"`
			MonthlyPrice    int64           `json:"monthly_price,omitempty"`
		}{
			Namespace:       manifest.Name,
			Name:            manifest.Name,
			Description:     manifest.Description,
			SourceURL:       manifest.Repository,
			Version:         manifest.Version,
			Manifest:        manifestData,
			DeploymentModel: deploymentModel,
			PricingModel:    pricingModel,
			PricePerCall:    pricePerCall,
			MonthlyPrice:    monthlyPrice,
		}

		if githubRepo != "" {
			publishReq.DeploymentModel = "hosted_omdr"
			publishReq.ArtifactType = "docker"
			publishReq.GitHubURL = githubRepo
			publishReq.GitHubToken = githubToken
			fmt.Printf("Publishing GitHub repo: %s\n", githubRepo)
			if githubToken != "" {
				fmt.Println("✓ Using provided GitHub token for private repo access")
			}
		} else if artifactPath != "" {
			publishReq.DeploymentModel = "hosted_omdr"
			fmt.Printf("Uploading artifact: %s\n", artifactPath)

			artifactURL, artifactType, err := uploadArtifact(cmd.Context(), apiClient, artifactPath)
			if err != nil {
				return fmt.Errorf("uploading artifact: %w", err)
			}

			publishReq.ArtifactType = artifactType
			publishReq.ArtifactURL = artifactURL
			fmt.Printf("✓ Artifact uploaded: %s\n", artifactURL)
		} else if selfHostedURL != "" {
			publishReq.DeploymentModel = "self_hosted"
			publishReq.ArtifactType = "endpoint"
			publishReq.ArtifactURL = selfHostedURL
			fmt.Printf("Publishing self-hosted endpoint: %s\n", selfHostedURL)
		} else if deploymentModel == "" {
			publishReq.DeploymentModel = "local"
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

		if err := apiClient.Post(cmd.Context(), "/api/v1/servers", publishReq, &publishResp); err != nil {
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
	publishCmd.Flags().StringVar(&deploymentModel, "deployment", "local", "Deployment model: local, hosted_omdr, self_hosted")
	publishCmd.Flags().StringVar(&artifactPath, "artifact", "", "Path to artifact (Docker image, WASM, etc.) for hosted deployment")
	publishCmd.Flags().StringVar(&githubRepo, "github", "", "GitHub repository URL for OMDR-hosted deployment")
	publishCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub personal access token for private repos")
	publishCmd.Flags().StringVar(&selfHostedURL, "self-hosted", "", "Self-hosted endpoint URL")
	publishCmd.Flags().StringVar(&pricingModel, "pricing", "free", "Pricing model: free, per_call, subscription")
	publishCmd.Flags().Int64Var(&pricePerCall, "price-per-call", 0, "Price per call in cents")
	publishCmd.Flags().Int64Var(&monthlyPrice, "monthly-price", 0, "Monthly subscription price in cents")
}

func uploadArtifact(ctx context.Context, apiClient *client.Client, artifactPath string) (string, string, error) {
	file, err := os.Open(artifactPath)
	if err != nil {
		return "", "", fmt.Errorf("opening artifact file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(artifactPath))
	var artifactType string
	switch ext {
	case ".wasm":
		artifactType = "wasm"
	case ".tar", ".tar.gz", ".tgz":
		artifactType = "docker"
	default:
		return "", "", fmt.Errorf("unsupported artifact type: %s (supported: .wasm, .tar, .tar.gz)", ext)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("artifact", filepath.Base(artifactPath))
	if err != nil {
		return "", "", fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", "", fmt.Errorf("copying file data: %w", err)
	}

	if err := writer.WriteField("artifact_type", artifactType); err != nil {
		return "", "", fmt.Errorf("writing artifact type: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", "", fmt.Errorf("closing multipart writer: %w", err)
	}

	var uploadResp struct {
		ArtifactURL  string `json:"artifact_url"`
		ArtifactType string `json:"artifact_type"`
	}

	if err := apiClient.PostMultipart(ctx, "/api/v1/artifacts/upload", writer.FormDataContentType(), body, &uploadResp); err != nil {
		return "", "", fmt.Errorf("uploading artifact: %w", err)
	}

	return uploadResp.ArtifactURL, uploadResp.ArtifactType, nil
}
