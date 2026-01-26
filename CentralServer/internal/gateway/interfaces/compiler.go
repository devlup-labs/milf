package interfaces

import (
	"central_server/internal/gateway/domain"
	"context"
)

type CompilerService interface {
	Compile(ctx context.Context, sourceCode string, runtime domain.RuntimeEnvironment) (wasmRef string, err error)
}
