package mcpspec

import "encoding/json"

// MCPManifest represents the mcp.json manifest file structure
type MCPManifest struct {
	Name        string              `json:"name" validate:"required"`
	Version     string              `json:"version" validate:"required,semver"`
	Description string              `json:"description"`
	Author      string              `json:"author"`
	License     string              `json:"license"`
	Repository  string              `json:"repository"`
	Runtime     RuntimeConfig       `json:"runtime"`
	Engines     *EngineRequirements `json:"engines,omitempty"`
	Tools       []ToolDefinition    `json:"tools,omitempty"`
	Resources   []ResourceDef       `json:"resources,omitempty"`
	Prompts     []PromptDef         `json:"prompts,omitempty"`
}

// RuntimeConfig specifies how to execute the MCP server
type RuntimeConfig struct {
	Type    string            `json:"type" validate:"required,oneof=node python docker"`
	Command string            `json:"command" validate:"required"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ToolDefinition describes a tool exposed by the MCP server
type ToolDefinition struct {
	Name        string          `json:"name" validate:"required"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ResourceDef describes a resource exposed by the MCP server
type ResourceDef struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	URI         string `json:"uri" validate:"required"`
	MimeType    string `json:"mimeType"`
}

// PromptDef describes a prompt template exposed by the MCP server
type PromptDef struct {
	Name        string      `json:"name" validate:"required"`
	Description string      `json:"description"`
	Arguments   []PromptArg `json:"arguments,omitempty"`
}

// PromptArg describes an argument for a prompt template
type PromptArg struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// EngineRequirements specifies minimum runtime version requirements
type EngineRequirements struct {
	Node   string `json:"node,omitempty"`   // e.g., ">=18.0.0"
	Python string `json:"python,omitempty"` // e.g., ">=3.9"
	Docker string `json:"docker,omitempty"` // e.g., ">=24.0"
}
