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
	"github.com/openmcpdirectory/omdr-cli/internal/cli/defaults"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun           bool
	artifactPath     string
	githubToken      string
	publishNamespace string
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish an MCP server to the registry",
	Long: `Publish your MCP server to the OMDR registry by reading the local
omdr.json (or omdr.toml / mcp.json) manifest.

Deployment mode, pricing, and hosting are read from the "omdr" section
of the manifest. Use 'omdr init' to generate one interactively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		manifest, err := mcpspec.LoadManifest(dir)
		if err != nil {
			return fmt.Errorf("loading manifest: %w\nRun 'omdr init' to create one", err)
		}

		if verbose {
			fmt.Printf("Loaded manifest from %s\n", manifest.SourceFile)
		}

		// Validate.
		fmt.Println("Validating manifest...")
		if err := mcpspec.ValidateManifest(manifest); err != nil {
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

		// Derive deployment info from OMDR extension.
		deploymentModel := "local"
		pricingModel := "free"
		var pricePerCall, monthlyPrice int64

		if ext := manifest.OMDR; ext != nil {
			if ext.Deployment != "" {
				deploymentModel = string(ext.Deployment)
			}
			if ext.Pricing != nil {
				pricingModel = string(ext.Pricing.Model)
				pricePerCall = int64(ext.Pricing.PerCallCents)
				monthlyPrice = int64(ext.Pricing.MonthlyCents)
			}
		}

		// Dry-run: stop early.
		if dryRun {
			fmt.Println("\n✓ Dry-run successful!")
			fmt.Printf("\nServer: %s@%s\n", manifest.Name, manifest.Version)
			fmt.Printf("Deployment: %s\n", deploymentModel)
			fmt.Printf("Pricing: %s\n", pricingModel)
			if manifest.Description != "" {
				fmt.Printf("Description: %s\n", manifest.Description)
			}
			fmt.Println("\nRun without --dry-run to publish")
			return nil
		}

		// Auth check.
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

		// API client.
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = defaults.RegistryURL
		}
		apiClient := client.NewClient(apiURL)
		apiClient.SetToken(token)

		// Resolve namespace.
		ns := publishNamespace
		if ns == "" {
			var userInfo struct {
				Username string `json:"username"`
			}
			if err := apiClient.Get(cmd.Context(), "/api/v1/users/me", &userInfo); err != nil {
				return fmt.Errorf("failed to resolve namespace (use --namespace flag): %w", err)
			}
			ns = userInfo.Username
		}

		// Serialise full manifest to send as JSON payload.
		manifestData, err := json.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("serialising manifest: %w", err)
		}

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
			Namespace:       ns,
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

		// Fill hosting details from OMDR extension.
		if manifest.OMDR != nil && manifest.OMDR.Hosting != nil {
			h := manifest.OMDR.Hosting
			if h.ArtifactType != "" {
				publishReq.ArtifactType = string(h.ArtifactType)
			}
			if h.GitHubURL != "" {
				publishReq.GitHubURL = h.GitHubURL
			}
			if h.EndpointURL != "" {
				publishReq.ArtifactURL = h.EndpointURL
			}
		}

		// CLI flag overrides for GitHub token.
		if githubToken != "" {
			publishReq.GitHubToken = githubToken
		}

		// Artifact upload for hosted deployments.
		if artifactPath != "" {
			publishReq.DeploymentModel = "hosted"
			fmt.Printf("Uploading artifact: %s\n", artifactPath)
			artifactURL, artifactType, err := uploadArtifact(cmd.Context(), apiClient, ns, manifest.Name, artifactPath)
			if err != nil {
				return fmt.Errorf("uploading artifact: %w", err)
			}
			publishReq.ArtifactType = artifactType
			publishReq.ArtifactURL = artifactURL
			fmt.Printf("✓ Artifact uploaded: %s\n", artifactURL)
		}

		fmt.Println("Publishing to registry...")

		// The API returns the server entity at the top level for regular publishes,
		// or {"message":"...", "server":..., "build_job_id":"..."} for hosted builds.
		// We use a struct that handles both: embedded fields for direct, plus
		// a Server field for the hosted-build wrapper.
		var publishResp struct {
			// Direct (non-hosted) response fields:
			ID          string `json:"id"`
			Namespace   string `json:"namespace"`
			Name        string `json:"name"`
			Description string `json:"description"`
			TrustScore  int    `json:"trust_score"`
			// Hosted-build wrapper:
			Message    string `json:"message,omitempty"`
			BuildJobID string `json:"build_job_id,omitempty"`
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

		respNS := publishResp.Namespace
		if respNS == "" {
			respNS = publishReq.Namespace
		}
		respName := publishResp.Name
		if respName == "" {
			respName = publishReq.Name
		}

		fmt.Printf("\nServer: %s/%s@%s\n", respNS, respName, publishReq.Version)
		if publishResp.Description != "" {
			fmt.Printf("Description: %s\n", publishResp.Description)
		}
		if publishResp.TrustScore > 0 {
			fmt.Printf("Trust Score: %d/100\n", publishResp.TrustScore)
		}
		if publishResp.ID != "" {
			fmt.Printf("Server ID: %s\n", publishResp.ID)
		}
		if publishResp.BuildJobID != "" {
			fmt.Printf("Build Job: %s (queued)\n", publishResp.BuildJobID)
		}

		// Display registry URL
		registryURL := fmt.Sprintf("%s/servers/%s/%s", apiURL, respNS, respName)
		fmt.Printf("\nView in registry: %s\n", registryURL)

		fmt.Println("\nUsers can now install your server with:")
		fmt.Printf("  omdr install %s/%s\n", respNS, respName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
	publishCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate manifest without publishing")
	publishCmd.Flags().StringVar(&artifactPath, "artifact", "", "Path to artifact (Docker image, WASM, etc.) for hosted deployment")
	publishCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub personal access token for private repos")
	publishCmd.Flags().StringVar(&publishNamespace, "namespace", "", "Namespace to publish under (defaults to your username)")
}

func uploadArtifact(ctx context.Context, apiClient *client.Client, namespace, name, artifactPath string) (string, string, error) {
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

	if err := apiClient.PostMultipart(ctx, fmt.Sprintf("/api/v1/servers/%s/%s/artifacts", namespace, name), writer.FormDataContentType(), body, &uploadResp); err != nil {
		return "", "", fmt.Errorf("uploading artifact: %w", err)
	}

	return uploadResp.ArtifactURL, uploadResp.ArtifactType, nil
}
