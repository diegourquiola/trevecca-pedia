package store

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Role represents a role in the system
type Role struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// UserWithRoles represents a user with their roles
type UserWithRoles struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}
