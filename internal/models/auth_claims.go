package models

import "github.com/golang-jwt/jwt/v5"

// JwtCustomClaims defines the structure of the JWT claims your application uses.
type JwtCustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"` // e.g., RoleAdmin, RoleNormalUser
	jwt.RegisteredClaims
}
