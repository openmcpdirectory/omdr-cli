package mcpspec

import (
	"encoding/json"
	"testing"
)

func TestValidateManifest_ValidManifest(t *testing.T) {
	manifest := &MCPManifest{
		Name:        "test-server",
		Version:     "1.0.0",
		Description: "A test MCP server",
		Runtime: RuntimeConfig{
			Type:    "node",
			Command: "node",
			Args:    []string{"index.js"},
		},
	}

	err := ValidateManifest(manifest)
	if err != nil {
		t.Errorf("expected no error for valid manifest, got: %v", err)
	}
}

func TestValidateManifest_MissingName(t *testing.T) {
	manifest := &MCPManifest{
		Version: "1.0.0",
		Runtime: RuntimeConfig{
			Type:    "node",
			Command: "node",
		},
	}

	err := ValidateManifest(manifest)
	if err == nil {
		t.Error("expected error for missing name")
	}

	validationErrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	if len(validationErrs) == 0 {
		t.Error("expected at least one validation error")
	}

	found := false
	for _, e := range validationErrs {
		if e.Field == "name" {
			found = true
			if e.Message != "field is required" {
				t.Errorf("expected 'field is required', got '%s'", e.Message)
			}
		}
	}

	if !found {
		t.Error("expected error for 'name' field")
	}
}

func TestValidateManifest_InvalidSemver(t *testing.T) {
	testCases := []struct {
		name    string
		version string
		valid   bool
	}{
		{"valid basic", "1.0.0", true},
		{"valid with prerelease", "1.0.0-alpha", true},
		{"valid with build", "1.0.0+build.123", true},
		{"valid complex", "2.1.3-beta.1+build.456", true},
		{"invalid no patch", "1.0", false},
		{"invalid no minor", "1", false},
		{"invalid text", "v1.0.0", false},
		{"invalid leading zero", "01.0.0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := &MCPManifest{
				Name:    "test",
				Version: tc.version,
				Runtime: RuntimeConfig{
					Type:    "node",
					Command: "node",
				},
			}

			err := ValidateManifest(manifest)
			if tc.valid && err != nil {
				t.Errorf("expected valid semver %s, got error: %v", tc.version, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected invalid semver %s to fail validation", tc.version)
			}
		})
	}
}

func TestValidateManifestJSON_ValidJSON(t *testing.T) {
	jsonData := []byte(`{
		"name": "test-server",
		"version": "1.0.0",
		"description": "Test server",
		"runtime": {
			"type": "node",
			"command": "node",
			"args": ["index.js"]
		}
	}`)

	manifest, err := ValidateManifestJSON(jsonData)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if manifest == nil {
		t.Fatal("expected manifest to be non-nil")
	}

	if manifest.Name != "test-server" {
		t.Errorf("expected name 'test-server', got '%s'", manifest.Name)
	}
}

func TestValidateManifestJSON_InvalidJSON(t *testing.T) {
	jsonData := []byte(`{invalid json}`)

	_, err := ValidateManifestJSON(jsonData)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	validationErrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	if len(validationErrs) == 0 {
		t.Error("expected at least one validation error")
	}
}

func TestValidateManifestJSON_EmptyData(t *testing.T) {
	_, err := ValidateManifestJSON([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestValidateManifest_WithTools(t *testing.T) {
	manifest := &MCPManifest{
		Name:    "test-server",
		Version: "1.0.0",
		Runtime: RuntimeConfig{
			Type:    "python",
			Command: "python",
		},
		Tools: []ToolDefinition{
			{
				Name:        "test-tool",
				Description: "A test tool",
				InputSchema: json.RawMessage(`{"type": "object"}`),
			},
		},
	}

	err := ValidateManifest(manifest)
	if err != nil {
		t.Errorf("expected no error for manifest with tools, got: %v", err)
	}
}

func TestValidateManifest_InvalidRuntimeType(t *testing.T) {
	manifest := &MCPManifest{
		Name:    "test-server",
		Version: "1.0.0",
		Runtime: RuntimeConfig{
			Type:    "invalid",
			Command: "node",
		},
	}

	err := ValidateManifest(manifest)
	if err == nil {
		t.Error("expected error for invalid runtime type")
	}

	validationErrs, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	found := false
	for _, e := range validationErrs {
		if e.Field == "runtime.type" {
			found = true
		}
	}

	if !found {
		t.Error("expected error for 'runtime.type' field")
	}
}
