package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
	"github.com/spf13/cobra"
)

var (
	initFormat string
	initForce  bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new omdr.json manifest",
	Long:  "Interactive wizard that scaffolds an omdr.json (or omdr.toml) manifest in the current directory",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initFormat, "format", "json", "Output format: json or toml")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing manifest")
}

func runInit(_ *cobra.Command, _ []string) error {
	// Check for existing manifest.
	for _, name := range mcpspec.ManifestFileNames {
		if _, err := os.Stat(name); err == nil && !initForce {
			return fmt.Errorf("%s already exists (use --force to overwrite)", name)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	prompt := func(label, fallback string) string {
		if fallback != "" {
			fmt.Printf("%s [%s]: ", label, fallback)
		} else {
			fmt.Printf("%s: ", label)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return fallback
		}
		return line
	}

	choose := func(label string, options []string, fallback string) string {
		fmt.Printf("%s (%s) [%s]: ", label, strings.Join(options, "/"), fallback)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return fallback
		}
		return line
	}

	// --- Gather info ---
	name := prompt("Server name", "my-mcp-server")
	version := prompt("Version", "0.1.0")
	description := prompt("Description", "")
	author := prompt("Author", "")
	license := prompt("License", "MIT")
	repository := prompt("Repository URL", "")
	runtimeType := choose("Runtime type", []string{"node", "python", "docker", "go"}, "node")
	command := prompt("Start command", runtimeType)

	deployment := choose("Deployment mode", []string{"local", "hosted", "self_hosted"}, "local")

	manifest := &mcpspec.MCPManifest{
		Name:        name,
		Version:     version,
		Description: description,
		Author:      author,
		License:     license,
		Repository:  repository,
		Runtime: mcpspec.RuntimeConfig{
			Type:    runtimeType,
			Command: command,
		},
		OMDR: &mcpspec.OMDRExtension{
			Version:    "1",
			Deployment: mcpspec.DeploymentMode(deployment),
		},
	}

	// Hosting details for non-local deployments.
	if deployment == "hosted" {
		artifactType := choose("Artifact type", []string{"docker", "wasm", "npm", "python"}, "docker")
		hosting := &mcpspec.HostingConfig{
			ArtifactType: mcpspec.ArtifactKind(artifactType),
		}

		if artifactType == "docker" {
			hosting.Dockerfile = prompt("Dockerfile path", "Dockerfile")
		}

		hosting.GitHubURL = prompt("GitHub repository URL", repository)
		hosting.HealthCheck = prompt("Health check path", "/health")
		manifest.OMDR.Hosting = hosting
	} else if deployment == "self_hosted" {
		endpoint := prompt("Endpoint URL", "")
		healthCheck := prompt("Health check path", "/health")
		manifest.OMDR.Hosting = &mcpspec.HostingConfig{
			EndpointURL: endpoint,
			HealthCheck: healthCheck,
		}
	}

	// Pricing.
	pricing := choose("Pricing model", []string{"free", "per_call", "subscription"}, "free")
	if pricing != "free" {
		manifest.OMDR.Pricing = &mcpspec.PricingConfig{
			Model: mcpspec.PricingModel(pricing),
		}
		if pricing == "per_call" {
			fmt.Print("Price per call (cents): ")
			var cents int
			fmt.Scanln(&cents)
			manifest.OMDR.Pricing.PerCallCents = cents
		} else if pricing == "subscription" {
			fmt.Print("Monthly price (cents): ")
			var cents int
			fmt.Scanln(&cents)
			manifest.OMDR.Pricing.MonthlyCents = cents
		}
	}

	// Write file.
	var (
		filename string
		data     []byte
		err      error
	)
	if initFormat == "toml" {
		filename = "omdr.toml"
		data, err = mcpspec.GenerateTOML(manifest)
	} else {
		filename = "omdr.json"
		data, err = mcpspec.GenerateJSON(manifest)
	}
	if err != nil {
		return fmt.Errorf("generating manifest: %w", err)
	}

	if err := os.WriteFile(filename, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	fmt.Printf("\n✓ Created %s\n", filename)
	fmt.Println("Run 'omdr validate' to check your manifest")
	return nil
}
