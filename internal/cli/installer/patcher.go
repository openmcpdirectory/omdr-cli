package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/entity"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
)

// ConfigPatcher handles modification of MCP client configuration files
type ConfigPatcher struct{}

// NewConfigPatcher creates a new config patcher
func NewConfigPatcher() *ConfigPatcher {
	return &ConfigPatcher{}
}

// PatchConfig adds an MCP server to a client's configuration file
// It handles missing/empty files, creates backups, and preserves existing entries
func (p *ConfigPatcher) PatchConfig(client detector.MCPClient, serverVersion entity.ServerVersion) error {
	// Parse the manifest to get runtime config
	var manifest mcpspec.MCPManifest
	if err := json.Unmarshal(serverVersion.Manifest, &manifest); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(client.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Read existing config (handle missing/empty)
	data, err := os.ReadFile(client.ConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config: %w", err)
	}

	// Parse config (handle empty/missing file)
	var config map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing config (malformed JSON): %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Create backup before modification (only if file exists and has content)
	if len(data) > 0 {
		backupPath := fmt.Sprintf("%s.backup.%d", client.ConfigPath, time.Now().Unix())
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
	}

	// Get or create mcpServers section
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Build server key (namespace/name format)
	serverKey := fmt.Sprintf("%s/%s", manifest.Name, serverVersion.Version)

	// Build server config from manifest
	serverConfig := buildServerConfig(manifest)

	// Add/update server entry (preserves existing entries)
	mcpServers[serverKey] = serverConfig
	config["mcpServers"] = mcpServers

	// Write updated config with proper formatting
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(client.ConfigPath, output, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// buildServerConfig creates the server configuration object from a manifest
func buildServerConfig(manifest mcpspec.MCPManifest) map[string]interface{} {
	config := map[string]interface{}{
		"command": manifest.Runtime.Command,
	}

	// Add args if present
	if len(manifest.Runtime.Args) > 0 {
		config["args"] = manifest.Runtime.Args
	}

	// Add env if present
	if len(manifest.Runtime.Env) > 0 {
		config["env"] = manifest.Runtime.Env
	}

	return config
}
