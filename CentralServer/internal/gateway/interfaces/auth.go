package interfaces

import (
	"context"

	authDomain "central_server/internal/auth/domain"
)

type AuthService interface {
	VerifyToken(ctx context.Context, tokenString string) (*authDomain.Claims, error)
}
