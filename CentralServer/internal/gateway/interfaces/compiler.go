package interfaces

import (
	"central_server/internal/gateway/domain"
	"context"
)

type CompilerService interface {
	Compile(ctx context.Context, sourceCode []byte, runtime domain.RuntimeEnvironment, funcID string, metadata map[string]string) (bool, error)
}
