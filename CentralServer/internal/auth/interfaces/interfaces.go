package interfaces

import (
	"central_server/internal/auth/domain"
	"context"
)

// UserRepository defines methods for user data persistence.
// UserRepository defines methods for user data persistence
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

// AuthService defines the business logic for authentication.
// SessionRepository defines methods for session management (optional/extensible)
type SessionRepository interface {
	// For now, we might just log creation or validate existence if we were stateful
	SaveSession(ctx context.Context, token string, userID string) error
}

// AuthService defines the business logic for authentication
type AuthService interface {
	Register(ctx context.Context, username, password string) error
	Login(ctx context.Context, username, password string) (string, error)
	VerifyToken(ctx context.Context, tokenString string) (*domain.Claims, error)
}
