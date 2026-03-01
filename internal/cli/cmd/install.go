package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/config"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/installer"
	clilogger "github.com/openmcpdirectory/omdr-cli/internal/cli/logger"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/runtime"
	"github.com/openmcpdirectory/omdr-cli/internal/entity"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	targetClient string
	configPath   string
	hosted       bool
)

var installCmd = &cobra.Command{
	Use:   "install <package>",
	Short: "Install an MCP server",
	Long:  "Install an MCP server from the registry and configure it in your MCP clients",
	Example: `  omdr install @namespace/server
  omdr install @namespace/server --hosted
  omdr install @namespace/server --auth-mode auth_only`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]

		// Parse package name (format: namespace/name or just name)
		namespace, name := parsePackageName(packageName)

		// Get API client
		apiURL := viper.GetString("api_url")
		if apiURL == "" {
			apiURL = "https://cli.omdr.dev"
		}

		apiClient := client.NewClient(apiURL)

		// Get auth token if available
		mgr, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("initializing config manager: %w", err)
		}

		token, _ := mgr.Get("auth.token")
		if token != "" {
			apiClient.SetToken(token)
			clilogger.Verbose("Using authentication token")
		}

		// Fetch server manifest from API
		fmt.Printf("Fetching %s from registry...\n", packageName)
		clilogger.Verbose("API URL: %s", apiURL)
		clilogger.Verbose("Request path: /api/v1/servers/%s/%s", namespace, name)

		var serverResp struct {
			Server       entity.Server        `json:"server"`
			Version      entity.ServerVersion `json:"version"`
			PaidServices []entity.PaidService `json:"paid_services,omitempty"`
			ForkInfo     *entity.ForkInfo     `json:"fork_info,omitempty"`
		}

		path := fmt.Sprintf("/api/v1/servers/%s/%s", namespace, name)
		if err := apiClient.Get(cmd.Context(), path, &serverResp); err != nil {
			return fmt.Errorf("fetching server: %w", err)
		}

		clilogger.Verbose("Server fetched: %s/%s version %s", serverResp.Server.Namespace, serverResp.Server.Name, serverResp.Version.Version)

		// Display paid service warnings if detected
		if len(serverResp.PaidServices) > 0 {
			fmt.Println("\n⚠ WARNING: This server uses paid third-party services")
			fmt.Println("The following paid services were detected:")
			for _, ps := range serverResp.PaidServices {
				costInfo := ""
				if ps.EstimatedCost != nil && *ps.EstimatedCost != "" {
					costInfo = fmt.Sprintf(" (estimated cost: %s)", *ps.EstimatedCost)
				}
				fmt.Printf("  • %s (%s)%s\n", ps.ServiceName, ps.ServiceType, costInfo)
			}
			fmt.Println("\nYou may need to provide API keys or incur additional costs to use this server.")

			// Prompt for confirmation
			fmt.Print("\nDo you want to continue with the installation? (y/N): ")
			var response string
			fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))

			if response != "y" && response != "yes" {
				fmt.Println("Installation cancelled.")
				return nil
			}
		}

		// Display fork information if detected
		if serverResp.ForkInfo != nil && serverResp.ForkInfo.IsFork {
			fmt.Println("\nℹ Fork Information:")
			fmt.Printf("  This server is a fork of: %s\n", serverResp.ForkInfo.ParentRepoURL)
			if serverResp.ForkInfo.ParentNamespace != "" && serverResp.ForkInfo.ParentName != "" {
				fmt.Printf("  Original: %s/%s\n", serverResp.ForkInfo.ParentNamespace, serverResp.ForkInfo.ParentName)
			}
			if serverResp.ForkInfo.ParentTrustScore != nil {
				fmt.Printf("  Parent trust score: %.1f/100\n", *serverResp.ForkInfo.ParentTrustScore)
			}
		}

		// Extract auth method if available
		authMethod := ""
		if serverResp.Server.AuthMethod != nil {
			authMethod = *serverResp.Server.AuthMethod
			clilogger.Verbose("Server auth method: %s", authMethod)
		}

		// Parse manifest to check runtime requirements
		var manifest mcpspec.MCPManifest
		if err := json.Unmarshal(serverResp.Version.Manifest, &manifest); err != nil {
			return fmt.Errorf("parsing manifest: %w", err)
		}

		// Detect installed MCP clients (needed for both local and hosted)
		fmt.Println("Detecting MCP clients...")
		clilogger.Verbose("Scanning for MCP client configurations...")
		det := detector.NewDetector()
		clients, err := det.DetectOrUseCustom(configPath)
		if err != nil {
			if err == os.ErrNotExist {
				return fmt.Errorf("config file not found: %s", configPath)
			}
			return fmt.Errorf("detecting clients: %w", err)
		}

		if len(clients) == 0 {
			return fmt.Errorf("no MCP clients detected. Use --config-path to specify a custom config file, or install Claude Desktop, Cursor, VS Code, Windsurf, Zed, Cline, Claude Code, or Codex")
		}

		// Filter by target client if specified
		if targetClient != "" {
			clients = filterClientsByType(clients, targetClient)
			if len(clients) == 0 {
				return fmt.Errorf("specified client '%s' not found", targetClient)
			}
		}

		fmt.Printf("Found %d MCP client(s):\n", len(clients))
		for _, c := range clients {
			fmt.Printf("  - %s (%s)\n", c.Name, c.ConfigPath)
		}

		// Check if hosted installation is requested or required
		if hosted {
			fmt.Println("\nInstalling as hosted server...")
			return installHosted(mgr, serverResp.Server, serverResp.Version, manifest, clients, authMethod)
		}

		// Check runtime requirements for local installation
		fmt.Println("\nChecking runtime requirements...")
		clilogger.Verbose("Runtime type: %s", manifest.Runtime.Type)

		// Check engine version constraints
		if manifest.Engines != nil {
			clilogger.Verbose("Checking engine requirements...")

			reqs := map[string]string{
				"node":   manifest.Engines.Node,
				"python": manifest.Engines.Python,
				"docker": manifest.Engines.Docker,
			}

			if err := runtime.CheckEngineRequirements(reqs); err != nil {
				return fmt.Errorf("engine requirement check failed: %w", err)
			}
			fmt.Println("  ✓ Engine requirements met")
		}

		if err := checkRuntimeRequirements(manifest.Runtime); err != nil {
			return fmt.Errorf("runtime check failed: %w", err)
		}

		// Patch client configs
		patcher := installer.NewConfigPatcher()
		successCount := 0
		var lastErr error

		for _, c := range clients {
			fmt.Printf("\nConfiguring %s...\n", c.Name)
			if err := patcher.PatchConfig(c, serverResp.Version); err != nil {
				fmt.Fprintf(os.Stderr, "  Failed to configure %s: %v\n", c.Name, err)
				lastErr = err
				continue
			}
			fmt.Printf("  ✓ Successfully configured %s\n", c.Name)
			successCount++
		}

		// Display results
		fmt.Println()
		if successCount == 0 {
			return fmt.Errorf("failed to configure any clients: %w", lastErr)
		}

		if successCount < len(clients) {
			fmt.Printf("⚠ Partially installed: %d/%d clients configured\n", successCount, len(clients))
		} else {
			fmt.Println("✓ Installation successful!")
		}

		fmt.Printf("\nInstalled: %s/%s@%s\n", serverResp.Server.Namespace, serverResp.Server.Name, serverResp.Version.Version)
		fmt.Printf("Description: %s\n", serverResp.Server.Description)

		if len(manifest.Tools) > 0 {
			fmt.Printf("Tools: %d\n", len(manifest.Tools))
		}
		if len(manifest.Resources) > 0 {
			fmt.Printf("Resources: %d\n", len(manifest.Resources))
		}
		if len(manifest.Prompts) > 0 {
			fmt.Printf("Prompts: %d\n", len(manifest.Prompts))
		}

		fmt.Println("\nRestart your MCP client(s) to use the new server.")

		return nil
	},
}

// parsePackageName splits a package name into namespace and name
// Supports formats: "namespace/name" or just "name" (uses default namespace)
func parsePackageName(pkg string) (namespace, name string) {
	parts := strings.SplitN(pkg, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// Default namespace if not specified
	return "default", parts[0]
}

// checkRuntimeRequirements verifies that required runtimes are available
func checkRuntimeRequirements(rt mcpspec.RuntimeConfig) error {
	switch rt.Type {
	case "docker":
		result := runtime.CheckDocker(false)
		if !result.Available {
			return fmt.Errorf("Docker is required but not available: %w", result.Error)
		}
		fmt.Println("  ✓ Docker is available")

	case "node":
		result := runtime.CheckNode()
		if !result.Available {
			return fmt.Errorf("Node.js is required but not available: %w", result.Error)
		}
		fmt.Println("  ✓ Node.js is available")

	case "python":
		result := runtime.CheckPython()
		if !result.Available {
			return fmt.Errorf("Python is required but not available: %w", result.Error)
		}
		fmt.Println("  ✓ Python is available")

	default:
		return fmt.Errorf("unknown runtime type: %s", rt.Type)
	}

	return nil
}

// filterClientsByType filters clients by the specified type
func filterClientsByType(clients []detector.MCPClient, clientType string) []detector.MCPClient {
	var filtered []detector.MCPClient

	targetType := detector.ClientType(strings.ToLower(clientType))

	for _, c := range clients {
		if c.Type == targetType {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

// installHosted installs a server as a hosted (proxied) server
func installHosted(mgr *config.Manager, server entity.Server, version entity.ServerVersion, manifest mcpspec.MCPManifest, clients []detector.MCPClient, authMethod string) error {
	// Get API key
	apiKey, err := mgr.Get("auth.token")
	if err != nil || apiKey == "" {
		return fmt.Errorf("authentication required for hosted servers. Run 'omdr auth login'")
	}

	// Get path to omdr binary
	omdrPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding omdr binary: %w", err)
	}

	clilogger.Verbose("Using omdr binary at: %s", omdrPath)

	// Create hosted server config
	serverKey := fmt.Sprintf("%s/%s", server.Namespace, server.Name)

	// Build proxy args based on auth method
	proxyArgs := []string{"proxy", serverKey}
	if authMethod == "auth_only" {
		proxyArgs = append(proxyArgs, "--auth-mode", "auth_only")
		clilogger.Verbose("Using auth-only mode for bandwidth optimization")
		fmt.Println("  ℹ Using auth-only mode (direct connection after authentication)")
	} else {
		clilogger.Verbose("Using full proxy mode")
	}

	hostedConfig := installer.HostedServerConfig{
		Command: omdrPath,
		Args:    proxyArgs,
		Env: map[string]string{
			"OMDR_API_KEY": apiKey,
		},
	}

	// Patch client configs
	patcher := installer.NewConfigPatcher()
	successCount := 0
	var lastErr error

	for _, c := range clients {
		fmt.Printf("\nConfiguring %s...\n", c.Name)
		if err := patcher.PatchHostedConfig(c, server, hostedConfig); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to configure %s: %v\n", c.Name, err)
			lastErr = err
			continue
		}
		fmt.Printf("  ✓ Successfully configured %s\n", c.Name)
		successCount++
	}

	// Display results
	fmt.Println()
	if successCount == 0 {
		return fmt.Errorf("failed to configure any clients: %w", lastErr)
	}

	if successCount < len(clients) {
		fmt.Printf("⚠ Partially installed: %d/%d clients configured\n", successCount, len(clients))
	} else {
		fmt.Println("✓ Installation successful!")
	}

	fmt.Printf("\nInstalled (hosted): %s/%s@%s\n", server.Namespace, server.Name, version.Version)
	fmt.Printf("Description: %s\n", server.Description)

	if authMethod == "auth_only" {
		fmt.Println("\n💡 Tip: This server uses auth-only mode for reduced bandwidth usage.")
	} else {
		fmt.Println("\n⚠ Note: This is a hosted server. Usage will be billed according to your subscription.")
	}

	if len(manifest.Tools) > 0 {
		fmt.Printf("Tools: %d\n", len(manifest.Tools))
	}
	if len(manifest.Resources) > 0 {
		fmt.Printf("Resources: %d\n", len(manifest.Resources))
	}
	if len(manifest.Prompts) > 0 {
		fmt.Printf("Prompts: %d\n", len(manifest.Prompts))
	}

	fmt.Println("\nRestart your MCP client(s) to use the new server.")

	return nil
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().StringVar(&targetClient, "client", "", "Target specific client (claude, cursor, vscode, windsurf, zed, cline, claude-code, codex)")
	installCmd.Flags().StringVar(&configPath, "config-path", "", "Custom MCP config file path (e.g., ~/.config/Code/User/mcp.json)")
	installCmd.Flags().BoolVar(&hosted, "hosted", false, "Install as a hosted server (runs on OMDR infrastructure)")
}
