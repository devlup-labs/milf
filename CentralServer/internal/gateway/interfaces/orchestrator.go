package interfaces

import (
	"context"

	"central_server/internal/gateway/domain"
)

type OrchestratorService interface {
	Execute(ctx context.Context, execution *domain.Execution) (interface{}, error)
}
