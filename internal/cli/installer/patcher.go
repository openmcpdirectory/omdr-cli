package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/registry"
	"github.com/openmcpdirectory/omdr-cli/internal/entity"
	mcpspec "github.com/openmcpdirectory/omdr-cli/pkg/mcp-spec"
)

// ConfigPatcher handles modification of MCP client configuration files
type ConfigPatcher struct {
	RegistryHome string
}

// NewConfigPatcher creates a new config patcher
func NewConfigPatcher() *ConfigPatcher {
	return &ConfigPatcher{}
}

// PatchConfig adds an MCP server to a client's configuration file
func (p *ConfigPatcher) PatchConfig(client detector.MCPClient, serverVersion entity.ServerVersion) error {
	var manifest mcpspec.MCPManifest
	if err := json.Unmarshal(serverVersion.Manifest, &manifest); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	serverKey := fmt.Sprintf("%s/%s", manifest.Name, serverVersion.Version)

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

	if err := reg.Register(serverKey, registry.ServerConfig{
		Command: manifest.Runtime.Command,
		Args:    manifest.Runtime.Args,
		Env:     manifest.Runtime.Env,
	}); err != nil {
		return fmt.Errorf("registering server: %w", err)
	}

	omdrCmd := resolveOmdrBinary()

	return p.patchClientConfig(client, serverKey, omdrCmd, []string{"run", serverKey}, nil)
}

// HostedServerConfig represents configuration for a hosted server
type HostedServerConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// PatchHostedConfig adds a hosted MCP server to a client's configuration
func (p *ConfigPatcher) PatchHostedConfig(client detector.MCPClient, server entity.Server, hostedConfig HostedServerConfig) error {
	serverKey := fmt.Sprintf("%s-%s", server.Namespace, server.Name)
	return p.patchClientConfig(client, serverKey, hostedConfig.Command, hostedConfig.Args, hostedConfig.Env)
}

// RemoveServerFromConfig removes an MCP server entry from a client config
func (p *ConfigPatcher) RemoveServerFromConfig(client detector.MCPClient, serverKey string) error {
	switch client.Type {
	case detector.ClientTypeClaudeCode:
		return removeFromClaudeCode(serverKey)
	case detector.ClientTypeCodex:
		return removeFromCodexTOML(client.ConfigPath, serverKey)
	case detector.ClientTypeZed:
		return removeFromZedConfig(client.ConfigPath, serverKey)
	default:
		return removeFromJSONConfig(client, serverKey)
	}
}

func (p *ConfigPatcher) patchClientConfig(client detector.MCPClient, serverKey, command string, args []string, env map[string]string) error {
	switch client.Type {
	case detector.ClientTypeClaudeCode:
		return patchClaudeCode(serverKey, command, args, env)
	case detector.ClientTypeCodex:
		return patchCodexTOML(client.ConfigPath, serverKey, command, args, env)
	case detector.ClientTypeZed:
		return patchZedConfig(client.ConfigPath, serverKey, command, args, env)
	default:
		return patchJSONConfig(client, serverKey, command, args, env)
	}
}

func patchJSONConfig(client detector.MCPClient, serverKey, command string, args []string, env map[string]string) error {
	configDir := filepath.Dir(client.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := os.ReadFile(client.ConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config: %w", err)
	}

	var config map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing config (malformed JSON): %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	if len(data) > 0 {
		backupPath := fmt.Sprintf("%s.backup.%d", client.ConfigPath, time.Now().Unix())
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
	}

	configKey := detector.ConfigKeyForClient(client.Type)

	servers, ok := config[configKey].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	// Wrap command on Windows for npx-based commands in some clients
	cmd := command
	finalArgs := args
	if runtime.GOOS == "windows" && needsCmdWrapper(command) {
		cmd = "cmd"
		finalArgs = append([]string{"/c", command}, args...)
	}

	entry := map[string]interface{}{
		"command": cmd,
		"args":    finalArgs,
	}

	if len(env) > 0 {
		entry["env"] = env
	}

	servers[serverKey] = entry
	config[configKey] = servers

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(client.ConfigPath, output, 0644)
}

func removeFromJSONConfig(client detector.MCPClient, serverKey string) error {
	data, err := os.ReadFile(client.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	backupPath := fmt.Sprintf("%s.backup.%d", client.ConfigPath, time.Now().Unix())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	configKey := detector.ConfigKeyForClient(client.Type)
	if servers, ok := config[configKey].(map[string]interface{}); ok {
		delete(servers, serverKey)
		config[configKey] = servers
	}

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(client.ConfigPath, output, 0644)
}

func patchClaudeCode(serverKey, command string, args []string, env map[string]string) error {
	cmdArgs := []string{"mcp", "add", "--transport", "stdio", sanitizeServerKey(serverKey), "--"}
	cmdArgs = append(cmdArgs, command)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("claude", cmdArgs...)
	for k, v := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude mcp add failed: %s: %w", string(output), err)
	}
	return nil
}

func removeFromClaudeCode(serverKey string) error {
	cmd := exec.Command("claude", "mcp", "remove", sanitizeServerKey(serverKey))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude mcp remove failed: %s: %w", string(output), err)
	}
	return nil
}

// codexConfig represents the top-level Codex config.toml structure
type codexConfig struct {
	MCPServers map[string]codexServerEntry `toml:"mcp_servers"`
	Remaining  map[string]interface{}      `toml:"-"`
}

type codexServerEntry struct {
	Command string            `toml:"command,omitempty"`
	Args    []string          `toml:"args,omitempty"`
	Env     map[string]string `toml:"env,omitempty"`
}

func patchCodexTOML(configPath, serverKey, command string, args []string, env map[string]string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config: %w", err)
	}

	if len(data) > 0 {
		backupPath := fmt.Sprintf("%s.backup.%d", configPath, time.Now().Unix())
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
	}

	// Parse existing as generic map to preserve unknown fields
	var raw map[string]interface{}
	if len(data) > 0 {
		if err := toml.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
	} else {
		raw = make(map[string]interface{})
	}

	servers, _ := raw["mcp_servers"].(map[string]interface{})
	if servers == nil {
		servers = make(map[string]interface{})
	}

	key := sanitizeServerKey(serverKey)
	entry := map[string]interface{}{
		"command": command,
		"args":    args,
	}
	if len(env) > 0 {
		entry["env"] = env
	}
	servers[key] = entry
	raw["mcp_servers"] = servers

	out, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, out, 0644)
}

func removeFromCodexTOML(configPath, serverKey string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading config: %w", err)
	}

	backupPath := fmt.Sprintf("%s.backup.%d", configPath, time.Now().Unix())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	if servers, ok := raw["mcp_servers"].(map[string]interface{}); ok {
		delete(servers, sanitizeServerKey(serverKey))
		raw["mcp_servers"] = servers
	}

	out, err := toml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, out, 0644)
}

// stripJSONCComments removes // and /* */ comments from JSONC data.
// It correctly handles strings so comment sequences inside strings are preserved.
func stripJSONCComments(data []byte) []byte {
	var buf bytes.Buffer
	i := 0
	n := len(data)
	for i < n {
		if data[i] == '"' {
			// String: copy as-is, handling backslash escapes
			buf.WriteByte(data[i])
			i++
			for i < n {
				if data[i] == '\\' && i+1 < n {
					buf.WriteByte(data[i])
					buf.WriteByte(data[i+1])
					i += 2
				} else if data[i] == '"' {
					buf.WriteByte(data[i])
					i++
					break
				} else {
					buf.WriteByte(data[i])
					i++
				}
			}
		} else if i+1 < n && data[i] == '/' && data[i+1] == '/' {
			// Line comment: skip to end of line
			i += 2
			for i < n && data[i] != '\n' {
				i++
			}
		} else if i+1 < n && data[i] == '/' && data[i+1] == '*' {
			// Block comment: skip to */
			i += 2
			for i+1 < n && !(data[i] == '*' && data[i+1] == '/') {
				i++
			}
			i += 2
		} else {
			buf.WriteByte(data[i])
			i++
		}
	}
	return buf.Bytes()
}

// patchZedConfig adds an MCP server to Zed's settings.json.
// Zed uses JSONC (JSON with comments) and requires the nested command format:
//
//	"context_servers": { "name": { "command": { "path": "...", "args": [...] } } }
func patchZedConfig(configPath, serverKey, command string, args []string, env map[string]string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config: %w", err)
	}

	var config map[string]interface{}
	if len(data) > 0 {
		clean := stripJSONCComments(data)
		if err := json.Unmarshal(clean, &config); err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	if len(data) > 0 {
		backupPath := fmt.Sprintf("%s.backup.%d", configPath, time.Now().Unix())
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("creating backup: %w", err)
		}
	}

	servers, ok := config["context_servers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}

	cmd := command
	finalArgs := args
	if runtime.GOOS == "windows" && needsCmdWrapper(command) {
		cmd = "cmd"
		finalArgs = append([]string{"/c", command}, args...)
	}

	cmdObj := map[string]interface{}{
		"path": cmd,
		"args": finalArgs,
	}
	if len(env) > 0 {
		cmdObj["env"] = env
	}

	servers[sanitizeServerKey(serverKey)] = map[string]interface{}{
		"command": cmdObj,
	}
	config["context_servers"] = servers

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, output, 0644)
}

func removeFromZedConfig(configPath, serverKey string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading config: %w", err)
	}

	clean := stripJSONCComments(data)
	var config map[string]interface{}
	if err := json.Unmarshal(clean, &config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	backupPath := fmt.Sprintf("%s.backup.%d", configPath, time.Now().Unix())
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	if servers, ok := config["context_servers"].(map[string]interface{}); ok {
		delete(servers, sanitizeServerKey(serverKey))
		config["context_servers"] = servers
	}

	output, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, output, 0644)
}

func resolveOmdrBinary() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "omdr"
}

func sanitizeServerKey(key string) string {
	r := strings.NewReplacer("/", "-", "@", "-")
	return r.Replace(key)
}

func needsCmdWrapper(command string) bool {
	lc := strings.ToLower(command)
	return lc == "npx" || lc == "npm" || lc == "node"
}
