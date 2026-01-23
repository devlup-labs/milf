package domain

import "github.com/golang-jwt/jwt/v5"

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"` // Stored as bcrypt hash
}

// Claims represents the JWT claims
type Claims struct {
	Username string `json:"username"`
	UserID   string `json:"user_id"`
	jwt.RegisteredClaims
}

// LoginRequest represents the payload for login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents the response containing the token
type AuthResponse struct {
	Token string `json:"token"`
}
