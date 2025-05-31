package models

import "time" // if you have CreatedAt, UpdatedAt

// Role constants for user roles
const (
	RoleAdmin      = "admin"
	RoleNormalUser = "normal_user"
	RoleGuest      = "guest" // Though guest is usually implied by lack of auth
)

// User struct (you'll have more fields from your DB schema)
type User struct {
	ID        string    `json:"id" db:"id"` // Assuming UUID string from DB
	Nickname  string    `json:"nickname,omitempty" db:"nickname"`
	Email     string    `json:"email,omitempty" db:"email"`
	Role      string    `json:"role" db:"role"`
	AvatarURL string    `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// Add other fields as per your DB schema
	// PasswordHash string `json:"-" db:"password_hash"` // Typically not sent in JSON response
}
