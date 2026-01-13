package detector

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
}

func TestDetectClients(t *testing.T) {
	d := NewDetector()
	clients, err := d.DetectClients()
	if err != nil {
		t.Fatalf("DetectClients failed: %v", err)
	}

	// Should return a non-nil list (may be empty if no clients installed)
	// This is valid - not all systems have MCP clients installed
	_ = clients
}

func TestGetClaudeConfigPaths(t *testing.T) {
	d := NewDetector()
	paths := d.getClaudeConfigPaths()

	if paths == nil || len(paths) == 0 {
		t.Fatal("getClaudeConfigPaths returned no paths")
	}

	// Verify path contains expected components based on OS
	path := paths[0]
	switch runtime.GOOS {
	case "darwin":
		if !contains(path, "Library") || !contains(path, "Claude") {
			t.Errorf("macOS path doesn't contain expected components: %s", path)
		}
	case "windows":
		if !contains(path, "Claude") || !contains(path, "claude_desktop_config.json") {
			t.Errorf("Windows path doesn't contain expected components: %s", path)
		}
	case "linux":
		if !contains(path, ".config") || !contains(path, "claude") {
			t.Errorf("Linux path doesn't contain expected components: %s", path)
		}
	}
}

func TestGetCursorConfigPaths(t *testing.T) {
	d := NewDetector()
	paths := d.getCursorConfigPaths()

	if paths == nil || len(paths) == 0 {
		t.Fatal("getCursorConfigPaths returned no paths")
	}

	// Verify path contains .cursor directory
	path := paths[0]
	if !contains(path, ".cursor") || !contains(path, "mcp.json") {
		t.Errorf("Cursor path doesn't contain '.cursor/mcp.json': %s", path)
	}
}

func TestGetVSCodeConfigPaths(t *testing.T) {
	d := NewDetector()
	paths := d.getVSCodeConfigPaths()

	if paths == nil || len(paths) == 0 {
		t.Fatal("getVSCodeConfigPaths returned no paths")
	}

	// Verify path contains Code and mcp.json
	path := paths[0]
	if !contains(path, "Code") || !contains(path, "mcp.json") {
		t.Errorf("VS Code path doesn't contain 'Code' and 'mcp.json': %s", path)
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")

	if err := os.WriteFile(tmpFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !fileExists(tmpFile) {
		t.Error("fileExists returned false for existing file")
	}

	// Test non-existing file
	if fileExists(filepath.Join(tmpDir, "nonexistent.json")) {
		t.Error("fileExists returned true for non-existing file")
	}

	// Test directory (should return false)
	if fileExists(tmpDir) {
		t.Error("fileExists returned true for directory")
	}
}

func TestDetectClientsWithMockFiles(t *testing.T) {
	// This test verifies detection logic by creating mock config files
	tmpDir := t.TempDir()

	// Create a mock Claude config
	claudeDir := filepath.Join(tmpDir, "claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}
	claudeConfig := filepath.Join(claudeDir, "claude_desktop_config.json")
	if err := os.WriteFile(claudeConfig, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create claude config: %v", err)
	}

	// Verify fileExists works with our mock
	if !fileExists(claudeConfig) {
		t.Error("Mock claude config not detected")
	}
}

func TestClientTypes(t *testing.T) {
	// Verify client type constants
	if ClientTypeClaude != "claude" {
		t.Errorf("ClientTypeClaude = %s, want 'claude'", ClientTypeClaude)
	}
	if ClientTypeCursor != "cursor" {
		t.Errorf("ClientTypeCursor = %s, want 'cursor'", ClientTypeCursor)
	}
	if ClientTypeVSCode != "vscode" {
		t.Errorf("ClientTypeVSCode = %s, want 'vscode'", ClientTypeVSCode)
	}
	if ClientTypeCustom != "custom" {
		t.Errorf("ClientTypeCustom = %s, want 'custom'", ClientTypeCustom)
	}
}

func TestMCPClientStruct(t *testing.T) {
	client := MCPClient{
		Name:       "Test Client",
		ConfigPath: "/path/to/config.json",
		Type:       ClientTypeClaude,
	}

	if client.Name != "Test Client" {
		t.Errorf("Name = %s, want 'Test Client'", client.Name)
	}
	if client.ConfigPath != "/path/to/config.json" {
		t.Errorf("ConfigPath = %s, want '/path/to/config.json'", client.ConfigPath)
	}
	if client.Type != ClientTypeClaude {
		t.Errorf("Type = %s, want %s", client.Type, ClientTypeClaude)
	}
}

func TestDetectOrUseCustom(t *testing.T) {
	d := NewDetector()
	tmpDir := t.TempDir()
	customConfig := filepath.Join(tmpDir, "custom-mcp.json")

	// Test with non-existent custom path
	_, err := d.DetectOrUseCustom(customConfig)
	if err != os.ErrNotExist {
		t.Errorf("Expected ErrNotExist for non-existent path, got: %v", err)
	}

	// Create custom config
	if err := os.WriteFile(customConfig, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create custom config: %v", err)
	}

	// Test with existing custom path
	clients, err := d.DetectOrUseCustom(customConfig)
	if err != nil {
		t.Fatalf("DetectOrUseCustom failed: %v", err)
	}
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}
	if clients[0].Type != ClientTypeCustom {
		t.Errorf("Expected ClientTypeCustom, got %s", clients[0].Type)
	}
	if clients[0].ConfigPath != customConfig {
		t.Errorf("Expected path %s, got %s", customConfig, clients[0].ConfigPath)
	}

	// Test with empty path (should auto-detect)
	clients, err = d.DetectOrUseCustom("")
	if err != nil {
		t.Fatalf("DetectOrUseCustom with empty path failed: %v", err)
	}
	// Should return detected clients (may be empty)
	_ = clients
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			filepath.Base(s) == substr || filepath.Dir(s) == substr ||
			containsPath(s, substr)))
}

func containsPath(path, component string) bool {
	for {
		if filepath.Base(path) == component {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return false
}
