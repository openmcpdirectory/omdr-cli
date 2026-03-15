package mcpspec

import "encoding/json"

// MCPManifest represents the omdr.json / mcp.json manifest file.
// It is a superset of the standard MCP manifest: standard MCP fields live at
// the top level and the optional "omdr" key carries OMDR-specific hosting,
// deployment, pricing and secret configuration.
type MCPManifest struct {
	// --- Standard MCP fields ---
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

	// --- OMDR extension ---
	OMDR *OMDRExtension `json:"omdr,omitempty" toml:"omdr,omitempty"`

	// SourceFile records which file the manifest was loaded from (not serialised).
	SourceFile string `json:"-" toml:"-"`
}

// StripOMDR returns a shallow copy of the manifest with the OMDR extension and
// SourceFile removed, suitable for clients that only understand standard MCP.
func (m *MCPManifest) StripOMDR() *MCPManifest {
	cp := *m
	cp.OMDR = nil
	cp.SourceFile = ""
	return &cp
}

// EngineRequirements specifies minimum runtime version requirements.
type EngineRequirements struct {
	Node   string `json:"node,omitempty" toml:"node,omitempty"`
	Python string `json:"python,omitempty" toml:"python,omitempty"`
	Docker string `json:"docker,omitempty" toml:"docker,omitempty"`
}

// RuntimeConfig specifies how to execute the MCP server.
type RuntimeConfig struct {
	Type    string            `json:"type" validate:"required,oneof=node python docker go" toml:"type"`
	Command string            `json:"command" validate:"required" toml:"command"`
	Args    []string          `json:"args,omitempty" toml:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty" toml:"env,omitempty"`
}

// ToolDefinition describes a tool exposed by the MCP server.
type ToolDefinition struct {
	Name        string          `json:"name" validate:"required"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ResourceDef describes a resource exposed by the MCP server.
type ResourceDef struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	URI         string `json:"uri" validate:"required"`
	MimeType    string `json:"mimeType"`
}

// PromptDef describes a prompt template exposed by the MCP server.
type PromptDef struct {
	Name        string      `json:"name" validate:"required"`
	Description string      `json:"description"`
	Arguments   []PromptArg `json:"arguments,omitempty"`
}

// PromptArg describes an argument for a prompt template.
type PromptArg struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ---------------------------------------------------------------------------
// OMDR Extension types
// ---------------------------------------------------------------------------

// DeploymentMode describes where the MCP server runs.
type DeploymentMode string

const (
	DeployLocal      DeploymentMode = "local"
	DeployHosted     DeploymentMode = "hosted"
	DeploySelfHosted DeploymentMode = "self_hosted"
)

// ArtifactKind describes the artefact packaging format.
type ArtifactKind string

const (
	ArtifactDocker ArtifactKind = "docker"
	ArtifactWASM   ArtifactKind = "wasm"
	ArtifactNPM    ArtifactKind = "npm"
	ArtifactPython ArtifactKind = "python"
)

// PricingModel describes how the server is priced.
type PricingModel string

const (
	PricingFree         PricingModel = "free"
	PricingPerCall      PricingModel = "per_call"
	PricingSubscription PricingModel = "subscription"
)

// AuthMethod describes the authentication mechanism for self-hosted servers.
type AuthMethod string

const (
	AuthBearer AuthMethod = "bearer"
	AuthAPIKey AuthMethod = "api_key"
	AuthOAuth2 AuthMethod = "oauth2"
)

// SecretManagedBy indicates who is responsible for providing a secret value.
type SecretManagedBy string

const (
	SecretManagedByUser    SecretManagedBy = "user"
	SecretManagedByCreator SecretManagedBy = "creator"
)

// OMDRExtension carries OMDR-specific configuration inside omdr.json.
type OMDRExtension struct {
	Version    string         `json:"version" toml:"version"` // schema version, currently "1"
	Deployment DeploymentMode `json:"deployment" toml:"deployment" validate:"omitempty,oneof=local hosted self_hosted"`
	Hosting    *HostingConfig `json:"hosting,omitempty" toml:"hosting,omitempty"`
	Secrets    []SecretConfig `json:"secrets,omitempty" toml:"secrets,omitempty"`
	Scaling    *ScalingConfig `json:"scaling,omitempty" toml:"scaling,omitempty"`
	Pricing    *PricingConfig `json:"pricing,omitempty" toml:"pricing,omitempty"`
	Auth       *AuthConfig    `json:"auth,omitempty" toml:"auth,omitempty"`
	Categories []string       `json:"categories,omitempty" toml:"categories,omitempty"`
}

// HostingConfig describes how the server is built and where it runs.
type HostingConfig struct {
	ArtifactType ArtifactKind `json:"artifact_type,omitempty" toml:"artifact_type,omitempty" validate:"omitempty,oneof=docker wasm npm python"`
	Dockerfile   string       `json:"dockerfile,omitempty" toml:"dockerfile,omitempty"`
	GitHubURL    string       `json:"github_url,omitempty" toml:"github_url,omitempty"`
	EndpointURL  string       `json:"endpoint_url,omitempty" toml:"endpoint_url,omitempty"`
	HealthCheck  string       `json:"health_check,omitempty" toml:"health_check,omitempty"`
}

// SecretConfig describes a secret required by the MCP server.
type SecretConfig struct {
	Name        string          `json:"name" toml:"name" validate:"required"`
	Description string          `json:"description,omitempty" toml:"description,omitempty"`
	Required    bool            `json:"required" toml:"required"`
	ManagedBy   SecretManagedBy `json:"managed_by" toml:"managed_by" validate:"required,oneof=user creator"`
}

// ScalingConfig controls hosted instance scaling.
type ScalingConfig struct {
	MinInstances int `json:"min_instances,omitempty" toml:"min_instances,omitempty"`
	MaxInstances int `json:"max_instances,omitempty" toml:"max_instances,omitempty"`
}

// PricingConfig describes creator-set pricing.
type PricingConfig struct {
	Model        PricingModel `json:"model" toml:"model" validate:"required,oneof=free per_call subscription"`
	PerCallCents int          `json:"per_call_cents,omitempty" toml:"per_call_cents,omitempty"`
	MonthlyCents int          `json:"monthly_cents,omitempty" toml:"monthly_cents,omitempty"`
}

// AuthConfig describes how a self-hosted server authenticates callers.
type AuthConfig struct {
	Method AuthMethod      `json:"method" toml:"method" validate:"required,oneof=bearer api_key oauth2"`
	Config json.RawMessage `json:"config,omitempty" toml:"-"`
}
