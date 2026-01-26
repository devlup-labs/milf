package interfaces

import (
	"central_server/internal/gateway/domain"
	"context"
)

type LambdaRepository interface {
	Save(ctx context.Context, lambda *domain.Lambda) error
	FindByID(ctx context.Context, id string) (*domain.Lambda, error)
	FindByWasmRef(ctx context.Context, wasmRef string) (*domain.Lambda, error)
	Delete(ctx context.Context, id string) error
}

type ExecutionRepository interface {
	Save(ctx context.Context, execution *domain.Execution) error
	FindByID(ctx context.Context, id string) (*domain.Execution, error)
	Update(ctx context.Context, execution *domain.Execution) error
}
