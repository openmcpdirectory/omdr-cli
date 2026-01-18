package entity

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account in the registry
type User struct {
	ID             uuid.UUID `json:"id" db:"id"`
	FullName       string    `json:"full_name" db:"full_name" validate:"required"`
	Username       string    `json:"username" db:"username" validate:"required"`
	Email          string    `json:"email" db:"email"`
	AvatarURL      string    `json:"avatar_url" db:"avatar_url"`
	Provider       string    `json:"provider" db:"provider" validate:"required"`
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id" validate:"required"`
	EmailVerified  bool      `json:"email_verified" db:"email_verified"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Namespace represents a reserved namespace for server publishing
type Namespace struct {
	Name      string    `json:"name" db:"name" validate:"required"`
	OwnerID   uuid.UUID `json:"owner_id" db:"owner_id" validate:"required"`
	Verified  bool      `json:"verified" db:"verified"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
