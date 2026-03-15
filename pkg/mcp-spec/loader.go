package mcpspec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// ManifestFileNames lists the filenames searched in priority order.
var ManifestFileNames = []string{
	"omdr.json",
	"omdr.toml",
	"mcp.json",
}

// LoadManifest auto-detects and loads a manifest from dir.
// It searches for omdr.json → omdr.toml → mcp.json and returns the first
// valid manifest found.
func LoadManifest(dir string) (*MCPManifest, error) {
	for _, name := range ManifestFileNames {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}

		manifest, err := parseManifestFile(name, data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}

		abs, _ := filepath.Abs(path)
		manifest.SourceFile = abs
		return manifest, nil
	}

	return nil, fmt.Errorf("no manifest found in %s (looked for %v)", dir, ManifestFileNames)
}

// LoadManifestFrom reads a manifest from an explicit file path.
func LoadManifestFrom(path string) (*MCPManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	name := filepath.Base(path)
	manifest, err := parseManifestFile(name, data)
	if err != nil {
		return nil, err
	}

	abs, _ := filepath.Abs(path)
	manifest.SourceFile = abs
	return manifest, nil
}

func parseManifestFile(filename string, data []byte) (*MCPManifest, error) {
	ext := filepath.Ext(filename)
	switch ext {
	case ".toml":
		return parseTOML(data)
	default:
		return parseJSON(data)
	}
}

func parseJSON(data []byte) (*MCPManifest, error) {
	var m MCPManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &m, nil
}

func parseTOML(data []byte) (*MCPManifest, error) {
	var m MCPManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid TOML: %w", err)
	}
	return &m, nil
}

// GenerateJSON serialises a manifest to indented JSON suitable for omdr.json.
func GenerateJSON(m *MCPManifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// GenerateTOML serialises a manifest to TOML suitable for omdr.toml.
func GenerateTOML(m *MCPManifest) ([]byte, error) {
	return toml.Marshal(m)
}
