package entity

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account in the registry
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	GitHubID  int64     `json:"github_id" db:"github_id" validate:"required"`
	Username  string    `json:"username" db:"username" validate:"required"`
	Email     string    `json:"email" db:"email"`
	AvatarURL string    `json:"avatar_url" db:"avatar_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Namespace represents a reserved namespace for server publishing
type Namespace struct {
	Name      string    `json:"name" db:"name" validate:"required"`
	OwnerID   uuid.UUID `json:"owner_id" db:"owner_id" validate:"required"`
	Verified  bool      `json:"verified" db:"verified"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
