package interfaces

import (
	"context"
)

type OrchestratorService interface {
	ReceiveTrigger(ctx context.Context, trigID string, funcID string, input string) (bool, error)
	ActivateService(ctx context.Context, funcID string) (bool, error)
	DeactivateService(ctx context.Context, funcID string) (bool, error)
}
