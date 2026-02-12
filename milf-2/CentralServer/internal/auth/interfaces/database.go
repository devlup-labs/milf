package interfaces

import (
	"central_server/internal/auth/domain"
	"context"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

type SessionRepository interface {
	SaveSession(ctx context.Context, token string, userID string) error
}

type AuthService interface {
	Register(ctx context.Context, username, password string) error
	Login(ctx context.Context, username, password string) (string, error)
	VerifyToken(ctx context.Context, token string) (*domain.Claims, error)
}
