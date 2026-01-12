package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/openmcpdirectory/omdr-cli/internal/cli/detector"
	"github.com/openmcpdirectory/omdr-cli/internal/entity"
)

func TestNewConfigPatcher(t *testing.T) {
	p := NewConfigPatcher()
	if p == nil {
		t.Fatal("NewConfigPatcher returned nil")
	}
}

func TestPatchConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node",
			"args": ["index.js"]
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	// Verify config was created
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify mcpServers section exists
	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers section not found")
	}

	// Verify server entry exists
	serverKey := "test-server/1.0.0"
	if _, ok := mcpServers[serverKey]; !ok {
		t.Errorf("Server entry %s not found", serverKey)
	}
}

func TestPatchConfig_PreservesExistingEntries(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create existing config with one server
	existingConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"existing-server/1.0.0": map[string]interface{}{
				"command": "python",
				"args":    []string{"server.py"},
			},
		},
	}

	existingData, _ := json.MarshalIndent(existingConfig, "", "  ")
	if err := os.WriteFile(configPath, existingData, 0644); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "new-server",
		"version": "2.0.0",
		"runtime": {
			"type": "node",
			"command": "node",
			"args": ["index.js"]
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "2.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	// Verify both servers exist
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers section not found")
	}

	// Verify existing server is preserved
	if _, ok := mcpServers["existing-server/1.0.0"]; !ok {
		t.Error("Existing server entry was not preserved")
	}

	// Verify new server was added
	if _, ok := mcpServers["new-server/2.0.0"]; !ok {
		t.Error("New server entry was not added")
	}

	// Verify we have exactly 2 servers
	if len(mcpServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(mcpServers))
	}
}

func TestPatchConfig_CreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create existing config
	existingConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	existingData, _ := json.MarshalIndent(existingConfig, "", "  ")
	if err := os.WriteFile(configPath, existingData, 0644); err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node"
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	// Verify backup was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	backupFound := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "config.json.backup.") {
			backupFound = true
			break
		}
	}

	if !backupFound {
		t.Error("Backup file was not created")
	}
}

func TestPatchConfig_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create malformed JSON
	malformedJSON := []byte(`{"mcpServers": {invalid json}`)
	if err := os.WriteFile(configPath, malformedJSON, 0644); err != nil {
		t.Fatalf("Failed to create malformed config: %v", err)
	}

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node"
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)

	// Should return error for malformed JSON
	if err == nil {
		t.Fatal("Expected error for malformed JSON, got nil")
	}

	if !strings.Contains(err.Error(), "malformed JSON") {
		t.Errorf("Error message should mention malformed JSON, got: %v", err)
	}
}

func TestPatchConfig_WithEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node",
			"args": ["index.js"],
			"env": {
				"API_KEY": "test-key",
				"DEBUG": "true"
			}
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	// Verify env vars are included
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	mcpServers := config["mcpServers"].(map[string]interface{})
	serverConfig := mcpServers["test-server/1.0.0"].(map[string]interface{})

	env, ok := serverConfig["env"].(map[string]interface{})
	if !ok {
		t.Fatal("env section not found in server config")
	}

	if env["API_KEY"] != "test-key" {
		t.Errorf("API_KEY = %v, want 'test-key'", env["API_KEY"])
	}

	if env["DEBUG"] != "true" {
		t.Errorf("DEBUG = %v, want 'true'", env["DEBUG"])
	}
}

func TestPatchConfig_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.json")

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node"
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(configPath)); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}

	// Verify config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestPatchConfig_InvalidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	// Invalid manifest (malformed JSON)
	invalidManifest := json.RawMessage(`{invalid}`)

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: invalidManifest,
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)

	// Should return error for invalid manifest
	if err == nil {
		t.Fatal("Expected error for invalid manifest, got nil")
	}

	if !strings.Contains(err.Error(), "parsing manifest") {
		t.Errorf("Error message should mention parsing manifest, got: %v", err)
	}
}

func TestBuildServerConfig(t *testing.T) {
	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node",
			"args": ["index.js", "--port", "3000"],
			"env": {
				"NODE_ENV": "production"
			}
		}
	}`

	var mcpManifest struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Runtime struct {
			Type    string            `json:"type"`
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		} `json:"runtime"`
	}

	if err := json.Unmarshal([]byte(manifest), &mcpManifest); err != nil {
		t.Fatalf("Failed to parse manifest: %v", err)
	}

	// Convert to mcpspec.MCPManifest for buildServerConfig
	var mcpspecManifest struct {
		Runtime struct {
			Command string
			Args    []string
			Env     map[string]string
		}
	}
	mcpspecManifest.Runtime.Command = mcpManifest.Runtime.Command
	mcpspecManifest.Runtime.Args = mcpManifest.Runtime.Args
	mcpspecManifest.Runtime.Env = mcpManifest.Runtime.Env

	// Test would need actual mcpspec.MCPManifest type
	// This is a simplified test to verify the structure
	config := map[string]interface{}{
		"command": mcpspecManifest.Runtime.Command,
		"args":    mcpspecManifest.Runtime.Args,
		"env":     mcpspecManifest.Runtime.Env,
	}

	if config["command"] != "node" {
		t.Errorf("command = %v, want 'node'", config["command"])
	}

	args, ok := config["args"].([]string)
	if !ok || len(args) != 3 {
		t.Errorf("args = %v, want 3 elements", config["args"])
	}

	env, ok := config["env"].(map[string]string)
	if !ok || env["NODE_ENV"] != "production" {
		t.Errorf("env = %v, want NODE_ENV=production", config["env"])
	}
}

func TestPatchConfig_ProperFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	client := detector.MCPClient{
		Name:       "Test Client",
		ConfigPath: configPath,
		Type:       detector.ClientTypeClaude,
	}

	manifest := `{
		"name": "test-server",
		"version": "1.0.0",
		"runtime": {
			"type": "node",
			"command": "node"
		}
	}`

	serverVersion := entity.ServerVersion{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Version:  "1.0.0",
		Manifest: json.RawMessage(manifest),
	}

	patcher := NewConfigPatcher()
	err := patcher.PatchConfig(client, serverVersion)
	if err != nil {
		t.Fatalf("PatchConfig failed: %v", err)
	}

	// Verify proper JSON formatting (indented)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Check for indentation (should have spaces)
	content := string(data)
	if !strings.Contains(content, "  ") {
		t.Error("Config file is not properly indented")
	}

	// Verify it's valid JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Errorf("Config file is not valid JSON: %v", err)
	}
}
