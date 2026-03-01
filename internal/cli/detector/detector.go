package detector

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ClientType represents the type of MCP client
type ClientType string

const (
	ClientTypeClaude     ClientType = "claude"
	ClientTypeCursor     ClientType = "cursor"
	ClientTypeVSCode     ClientType = "vscode"
	ClientTypeWindsurf   ClientType = "windsurf"
	ClientTypeZed        ClientType = "zed"
	ClientTypeCline      ClientType = "cline"
	ClientTypeClaudeCode ClientType = "claude-code"
	ClientTypeCodex      ClientType = "codex"
	ClientTypeCustom     ClientType = "custom"
)

// MCPClient represents a detected MCP client installation
type MCPClient struct {
	Name       string
	ConfigPath string
	Type       ClientType
}

// ConfigKeyForClient returns the JSON key used for MCP server entries per client type
func ConfigKeyForClient(ct ClientType) string {
	switch ct {
	case ClientTypeVSCode:
		return "servers"
	case ClientTypeZed:
		return "context_servers"
	default:
		return "mcpServers"
	}
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

	detectors := []func() *MCPClient{
		d.detectClaude,
		d.detectCursor,
		d.detectVSCode,
		d.detectWindsurf,
		d.detectZed,
		d.detectCline,
		d.detectClaudeCode,
		d.detectCodex,
	}

	for _, detect := range detectors {
		if c := detect(); c != nil {
			clients = append(clients, *c)
		}
	}

	return clients, nil
}

// DetectOrUseCustom detects clients or uses a custom config path
func (d *Detector) DetectOrUseCustom(customPath string) ([]MCPClient, error) {
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
	return d.DetectClients()
}

func (d *Detector) detectClaude() *MCPClient {
	for _, path := range d.getClaudeConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "Claude Desktop", ConfigPath: path, Type: ClientTypeClaude}
		}
	}
	return nil
}

func (d *Detector) detectCursor() *MCPClient {
	for _, path := range d.getCursorConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "Cursor", ConfigPath: path, Type: ClientTypeCursor}
		}
	}
	return nil
}

func (d *Detector) detectVSCode() *MCPClient {
	for _, path := range d.getVSCodeConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "VS Code", ConfigPath: path, Type: ClientTypeVSCode}
		}
	}
	return nil
}

func (d *Detector) detectWindsurf() *MCPClient {
	for _, path := range d.getWindsurfConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "Windsurf", ConfigPath: path, Type: ClientTypeWindsurf}
		}
	}
	return nil
}

func (d *Detector) detectZed() *MCPClient {
	for _, path := range d.getZedConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "Zed", ConfigPath: path, Type: ClientTypeZed}
		}
	}
	return nil
}

func (d *Detector) detectCline() *MCPClient {
	for _, path := range d.getClineConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "Cline", ConfigPath: path, Type: ClientTypeCline}
		}
	}
	return nil
}

func (d *Detector) detectClaudeCode() *MCPClient {
	if _, err := exec.LookPath("claude"); err == nil {
		return &MCPClient{Name: "Claude Code", ConfigPath: "", Type: ClientTypeClaudeCode}
	}
	return nil
}

func (d *Detector) detectCodex() *MCPClient {
	for _, path := range d.getCodexConfigPaths() {
		if fileExists(path) {
			return &MCPClient{Name: "Codex", ConfigPath: path, Type: ClientTypeCodex}
		}
	}
	return nil
}

func (d *Detector) getClaudeConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{filepath.Join(appData, "Claude", "claude_desktop_config.json")}
	case "linux":
		return []string{filepath.Join(home, ".config", "claude", "claude_desktop_config.json")}
	default:
		return nil
	}
}

func (d *Detector) getCursorConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".cursor", "mcp.json")}
}

func (d *Detector) getVSCodeConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json")}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{filepath.Join(appData, "Code", "User", "mcp.json")}
	case "linux":
		return []string{filepath.Join(home, ".config", "Code", "User", "mcp.json")}
	default:
		return nil
	}
}

func (d *Detector) getWindsurfConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")}
}

func (d *Detector) getZedConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	switch runtime.GOOS {
	case "darwin", "linux":
		return []string{filepath.Join(home, ".config", "zed", "settings.json")}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{filepath.Join(appData, "Zed", "settings.json")}
	default:
		return nil
	}
}

func (d *Detector) getClineConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	const clineExtID = "saoudrizwan.claude-dev"
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", clineExtID, "cline_mcp_settings.json")}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return []string{filepath.Join(appData, "Code", "User", "globalStorage", clineExtID, "cline_mcp_settings.json")}
	case "linux":
		return []string{filepath.Join(home, ".config", "Code", "User", "globalStorage", clineExtID, "cline_mcp_settings.json")}
	default:
		return nil
	}
}

func (d *Detector) getCodexConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, ".codex", "config.toml")}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
