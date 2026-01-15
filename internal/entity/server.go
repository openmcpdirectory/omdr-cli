package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Server represents an MCP server in the registry
type Server struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Namespace       string    `json:"namespace" db:"namespace" validate:"required"`
	Name            string    `json:"name" db:"name" validate:"required"`
	Description     string    `json:"description" db:"description"`
	SourceURL       string    `json:"source_url" db:"source_url"`
	Verified        bool      `json:"verified" db:"verified"`
	TrustScore      int       `json:"trust_score" db:"trust_score" validate:"min=0,max=100"`
	InstallCount    int64     `json:"install_count" db:"install_count"`
	OwnerID         uuid.UUID `json:"owner_id" db:"owner_id"`
	IsSponsored     bool      `json:"is_sponsored" db:"is_sponsored"`
	SponsorTier     *string   `json:"sponsor_tier,omitempty" db:"sponsor_tier"`
	SponsorPriority int       `json:"sponsor_priority" db:"sponsor_priority"`
	AuthMethod      *string   `json:"auth_method,omitempty" db:"auth_method"`
	IsFork          bool      `json:"is_fork" db:"is_fork"`
	ParentRepoURL   *string   `json:"parent_repo_url,omitempty" db:"parent_repo_url"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// PaidService represents a detected paid third-party service
type PaidService struct {
	ID            uuid.UUID `json:"id"`
	ServerID      uuid.UUID `json:"server_id"`
	ServiceName   string    `json:"service_name"`
	ServiceType   string    `json:"service_type"`
	Location      string    `json:"location"`
	EstimatedCost *string   `json:"estimated_cost,omitempty"`
	DetectedAt    time.Time `json:"detected_at"`
}

// ForkInfo represents fork relationship information
type ForkInfo struct {
	IsFork           bool     `json:"is_fork"`
	ParentRepoURL    string   `json:"parent_repo_url,omitempty"`
	ParentNamespace  string   `json:"parent_namespace,omitempty"`
	ParentName       string   `json:"parent_name,omitempty"`
	ParentTrustScore *float64 `json:"parent_trust_score,omitempty"`
}

// ServerVersion represents a specific version of an MCP server
type ServerVersion struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ServerID     uuid.UUID       `json:"server_id" db:"server_id" validate:"required"`
	Version      string          `json:"version" db:"version" validate:"required,semver"`
	Manifest     json.RawMessage `json:"manifest" db:"manifest" validate:"required"`
	Capabilities Capabilities    `json:"capabilities" db:"capabilities"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// Capabilities represents the capabilities exposed by an MCP server
type Capabilities struct {
	HasTools     bool `json:"has_tools"`
	HasResources bool `json:"has_resources"`
	HasPrompts   bool `json:"has_prompts"`
}
