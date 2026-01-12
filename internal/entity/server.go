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
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
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
