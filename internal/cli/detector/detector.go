package detector

import (
	"os"
	"path/filepath"
	"runtime"
)

// ClientType represents the type of MCP client
type ClientType string

const (
	ClientTypeClaude ClientType = "claude"
	ClientTypeCursor ClientType = "cursor"
	ClientTypeVSCode ClientType = "vscode"
	ClientTypeCustom ClientType = "custom"
)

// MCPClient represents a detected MCP client installation
type MCPClient struct {
	Name       string
	ConfigPath string
	Type       ClientType
}

// Detector finds installed MCP clients on the system
type Detector struct{}

// NewDetector creates a new client detector
func NewDetector() *Detector {
	return &Detector{}
}

// DetectClients finds all installed MCP clients and returns their config paths
func (d *Detector) DetectClients() ([]MCPClient, error) {
	var clients []MCPClient

	// Detect Claude Desktop
	if claudeClient := d.detectClaude(); claudeClient != nil {
		clients = append(clients, *claudeClient)
	}

	// Detect Cursor
	if cursorClient := d.detectCursor(); cursorClient != nil {
		clients = append(clients, *cursorClient)
	}

	// Detect VS Code
	if vscodeClient := d.detectVSCode(); vscodeClient != nil {
		clients = append(clients, *vscodeClient)
	}

	return clients, nil
}

// DetectOrUseCustom detects clients or uses a custom config path
func (d *Detector) DetectOrUseCustom(customPath string) ([]MCPClient, error) {
	// If custom path provided, use it
	if customPath != "" {
		if !fileExists(customPath) {
			return nil, os.ErrNotExist
		}
		return []MCPClient{
			{
				Name:       "Custom",
				ConfigPath: customPath,
				Type:       ClientTypeCustom,
			},
		}, nil
	}

	// Otherwise detect automatically
	return d.DetectClients()
}

// detectClaude checks for Claude Desktop installation
func (d *Detector) detectClaude() *MCPClient {
	paths := d.getClaudeConfigPaths()
	for _, path := range paths {
		if fileExists(path) {
			return &MCPClient{
				Name:       "Claude Desktop",
				ConfigPath: path,
				Type:       ClientTypeClaude,
			}
		}
	}
	return nil
}

// detectCursor checks for Cursor installation
func (d *Detector) detectCursor() *MCPClient {
	paths := d.getCursorConfigPaths()
	for _, path := range paths {
		if fileExists(path) {
			return &MCPClient{
				Name:       "Cursor",
				ConfigPath: path,
				Type:       ClientTypeCursor,
			}
		}
	}
	return nil
}

// detectVSCode checks for VS Code MCP extension
func (d *Detector) detectVSCode() *MCPClient {
	paths := d.getVSCodeConfigPaths()
	for _, path := range paths {
		if fileExists(path) {
			return &MCPClient{
				Name:       "VS Code",
				ConfigPath: path,
				Type:       ClientTypeVSCode,
			}
		}
	}
	return nil
}

// getClaudeConfigPaths returns OS-specific paths for Claude Desktop config
func (d *Detector) getClaudeConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{
			filepath.Join(appData, "Claude", "claude_desktop_config.json"),
		}
	case "linux":
		return []string{
			filepath.Join(home, ".config", "claude", "claude_desktop_config.json"),
		}
	default:
		return nil
	}
}

// getCursorConfigPaths returns OS-specific paths for Cursor config
func (d *Detector) getCursorConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Cursor uses a simple ~/.cursor/mcp.json for global config
	return []string{
		filepath.Join(home, ".cursor", "mcp.json"),
	}
}

// getVSCodeConfigPaths returns OS-specific paths for VS Code MCP extension config
func (d *Detector) getVSCodeConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json"),
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{
			filepath.Join(appData, "Code", "User", "mcp.json"),
		}
	case "linux":
		return []string{
			filepath.Join(home, ".config", "Code", "User", "mcp.json"),
		}
	default:
		return nil
	}
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
