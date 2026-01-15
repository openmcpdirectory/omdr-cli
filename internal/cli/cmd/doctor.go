package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	stdruntime "runtime"
	"strings"
	"time"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/client"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/runtime"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ExitSuccess       = 0
	ExitCriticalIssue = 1
)

type CheckResult struct {
	Name       string
	Status     string // "OK", "WARNING", "ERROR"
	Message    string
	Suggestion string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose environment setup",
	Long:  "Check for installed MCP clients, required runtimes, and network connectivity to diagnose installation issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		results := []CheckResult{}
		hasCriticalIssue := false

		fmt.Println("Running OMDR environment diagnostics...")
		fmt.Println()

		// Check MCP clients
		clientResults := checkMCPClients()
		results = append(results, clientResults...)

		// Check runtimes
		runtimeResults := checkRuntimes()
		results = append(results, runtimeResults...)

		// Check network connectivity
		networkResult := checkNetworkConnectivity()
		results = append(results, networkResult)

		// Display results
		fmt.Println("=== Diagnostic Results ===")
		fmt.Println()
		for _, result := range results {
			displayResult(result)
			if result.Status == "ERROR" {
				hasCriticalIssue = true
			}
		}

		// Summary
		fmt.Println("\n=== Summary ===")
		errorCount := 0
		warningCount := 0
		okCount := 0

		for _, result := range results {
			switch result.Status {
			case "ERROR":
				errorCount++
			case "WARNING":
				warningCount++
			case "OK":
				okCount++
			}
		}

		fmt.Printf("✓ %d checks passed\n", okCount)
		if warningCount > 0 {
			fmt.Printf("⚠ %d warnings\n", warningCount)
		}
		if errorCount > 0 {
			fmt.Printf("✗ %d errors\n", errorCount)
		}

		if hasCriticalIssue {
			fmt.Println("\nCritical issues detected. Please address the errors above.")
			os.Exit(ExitCriticalIssue)
		}

		if warningCount > 0 {
			fmt.Println("\nSome optional features may not be available. See warnings above.")
		} else {
			fmt.Println("\nAll checks passed! Your environment is ready.")
		}

		return nil
	},
}

func checkMCPClients() []CheckResult {
	results := []CheckResult{}

	d := detector.NewDetector()
	clients, err := d.DetectClients()

	if err != nil {
		results = append(results, CheckResult{
			Name:       "MCP Client Detection",
			Status:     "ERROR",
			Message:    fmt.Sprintf("Failed to detect clients: %v", err),
			Suggestion: "Ensure you have read permissions for application directories",
		})
		return results
	}

	if len(clients) == 0 {
		results = append(results, CheckResult{
			Name:       "MCP Clients",
			Status:     "WARNING",
			Message:    "No MCP clients detected",
			Suggestion: "Install Claude Desktop, Cursor, or VS Code with MCP extension to use OMDR",
		})
	} else {
		for _, client := range clients {
			version := detectClientVersion(client)
			versionInfo := ""
			if version != "" {
				versionInfo = fmt.Sprintf(" (version: %s)", version)
			}

			results = append(results, CheckResult{
				Name:    fmt.Sprintf("MCP Client: %s", client.Name),
				Status:  "OK",
				Message: fmt.Sprintf("Found at %s%s", client.ConfigPath, versionInfo),
			})
		}
	}

	return results
}

func detectClientVersion(client detector.MCPClient) string {
	// Try to detect version based on client type
	switch client.Type {
	case detector.ClientTypeClaude:
		return detectClaudeVersion()
	case detector.ClientTypeCursor:
		return detectCursorVersion()
	case detector.ClientTypeVSCode:
		return detectVSCodeVersion()
	}
	return ""
}

func detectClaudeVersion() string {
	// Claude Desktop version detection varies by OS
	switch stdruntime.GOOS {
	case "darwin":
		// On macOS, check the app bundle
		cmd := exec.Command("defaults", "read", "/Applications/Claude.app/Contents/Info.plist", "CFBundleShortVersionString")
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	case "windows":
		// On Windows, check registry or executable version
		// This is a simplified check
		return "installed"
	case "linux":
		return "installed"
	}
	return ""
}

func detectCursorVersion() string {
	cmd := exec.Command("cursor", "--version")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}
	return ""
}

func detectVSCodeVersion() string {
	cmd := exec.Command("code", "--version")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			return strings.TrimSpace(lines[0])
		}
	}
	return ""
}

func checkRuntimes() []CheckResult {
	results := []CheckResult{}

	// Check Node.js
	nodeCheck := runtime.CheckNode()
	if nodeCheck.Available {
		results = append(results, CheckResult{
			Name:    "Node.js",
			Status:  "OK",
			Message: fmt.Sprintf("Found: %s", nodeCheck.Version),
		})
	} else {
		results = append(results, CheckResult{
			Name:       "Node.js",
			Status:     "WARNING",
			Message:    "Node.js not found",
			Suggestion: "Install Node.js if you need to run MCP servers that require it",
		})
	}

	// Check Python
	pythonCheck := runtime.CheckPython()
	if pythonCheck.Available {
		results = append(results, CheckResult{
			Name:    "Python",
			Status:  "OK",
			Message: fmt.Sprintf("Found: %s", pythonCheck.Version),
		})
	} else {
		results = append(results, CheckResult{
			Name:       "Python",
			Status:     "WARNING",
			Message:    "Python not found",
			Suggestion: "Install Python if you need to run MCP servers that require it",
		})
	}

	// Check Docker
	dockerCheck := runtime.CheckDocker(true)
	if dockerCheck.Available {
		if dockerCheck.Error != nil {
			results = append(results, CheckResult{
				Name:       "Docker",
				Status:     "WARNING",
				Message:    fmt.Sprintf("Found: %s (daemon not running)", dockerCheck.Version),
				Suggestion: "Start Docker daemon to run containerized MCP servers",
			})
		} else {
			results = append(results, CheckResult{
				Name:    "Docker",
				Status:  "OK",
				Message: fmt.Sprintf("Found: %s (daemon running)", dockerCheck.Version),
			})
		}
	} else {
		results = append(results, CheckResult{
			Name:       "Docker",
			Status:     "WARNING",
			Message:    "Docker not found",
			Suggestion: "Install Docker if you need to run containerized MCP servers",
		})
	}

	return results
}

func checkNetworkConnectivity() CheckResult {
	apiURL := viper.GetString("api_url")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	apiClient := client.NewClient(apiURL)

	// Try to reach a simple endpoint (health check or servers list)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a channel to handle the API call with timeout
	done := make(chan error, 1)
	go func() {
		var result interface{}
		done <- apiClient.Get(ctx, "/api/v1/servers?limit=1", &result)
	}()

	select {
	case err := <-done:
		if err != nil {
			return CheckResult{
				Name:       "Network Connectivity",
				Status:     "ERROR",
				Message:    fmt.Sprintf("Cannot reach API at %s: %v", apiURL, err),
				Suggestion: "Check your internet connection and verify the API URL in config",
			}
		}
		return CheckResult{
			Name:    "Network Connectivity",
			Status:  "OK",
			Message: fmt.Sprintf("Successfully connected to API at %s", apiURL),
		}
	case <-ctx.Done():
		return CheckResult{
			Name:       "Network Connectivity",
			Status:     "ERROR",
			Message:    fmt.Sprintf("Timeout connecting to API at %s", apiURL),
			Suggestion: "Check your internet connection and verify the API URL in config",
		}
	}
}

func displayResult(result CheckResult) {
	var icon string
	switch result.Status {
	case "OK":
		icon = "✓"
	case "WARNING":
		icon = "⚠"
	case "ERROR":
		icon = "✗"
	}

	fmt.Printf("%s %s: %s\n", icon, result.Name, result.Message)
	if result.Suggestion != "" {
		fmt.Printf("  → %s\n", result.Suggestion)
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
