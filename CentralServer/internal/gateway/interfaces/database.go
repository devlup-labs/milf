package interfaces

import (
	"central_server/internal/gateway/domain"
	"context"
)

type FuncGatewayDB interface {
	Save(ctx context.Context, lambda *domain.Lambda) error
	FindByID(ctx context.Context, id string) (*domain.Lambda, error)
	FindByWasmRef(ctx context.Context, wasmRef string) (*domain.Lambda, error)
	Delete(ctx context.Context, id string) error
}

type CompilerDB interface {
	GetStatus(ctx context.Context, funcID string) (string, error) 
}