package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/openmcpdirectory/omdr-cli/internal/entity"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
)

// ConfigPatcher handles modification of MCP client configuration files
type ConfigPatcher struct {
	RegistryHome string // Optional override for testing
}

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

	// Register server in local registry
	var reg *registry.LocalRegistry
	var regErr error

	if p.RegistryHome != "" {
		reg, regErr = registry.NewLocalRegistryWithHome(p.RegistryHome)
	} else {
		reg, regErr = registry.NewLocalRegistry()
	}

	if regErr != nil {
		return fmt.Errorf("initializing registry: %w", regErr)
	}

	serverConfigData := registry.ServerConfig{
		Command: manifest.Runtime.Command,
		Args:    manifest.Runtime.Args,
		Env:     manifest.Runtime.Env,
	}

	if err := reg.Register(serverKey, serverConfigData); err != nil {
		return fmt.Errorf("registering server: %w", err)
	}

	// Build server config for the client (using omdr run)
	// We need the absolute path to the omdr executable or assume it's in PATH.
	// Users usually have omdr in PATH.
	omdrCmd := "omdr"
	executable, err := os.Executable()
	if err == nil {
		omdrCmd = executable
	}

	clientServerConfig := map[string]interface{}{
		"command": omdrCmd,
		"args":    []string{"run", serverKey},
		// We don't verify env here, they are injected by omdr run
	}

	// Add/update server entry (preserves existing entries)
	mcpServers[serverKey] = clientServerConfig
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

// HostedServerConfig represents configuration for a hosted server
type HostedServerConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// PatchHostedConfig adds a hosted MCP server to a client's configuration
func (p *ConfigPatcher) PatchHostedConfig(client detector.MCPClient, server entity.Server, hostedConfig HostedServerConfig) error {
	// Ensure config directory exists
	configDir := filepath.Dir(client.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Read existing config
	data, err := os.ReadFile(client.ConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config: %w", err)
	}

	// Parse config
	var config map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Create backup
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

	// Build server key
	serverKey := fmt.Sprintf("%s-%s", server.Namespace, server.Name)

	// Build hosted server config
	serverConfig := map[string]interface{}{
		"command": hostedConfig.Command,
		"args":    hostedConfig.Args,
	}

	if len(hostedConfig.Env) > 0 {
		serverConfig["env"] = hostedConfig.Env
	}

	// Add server entry
	mcpServers[serverKey] = serverConfig
	config["mcpServers"] = mcpServers

	// Write updated config
	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(client.ConfigPath, output, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
