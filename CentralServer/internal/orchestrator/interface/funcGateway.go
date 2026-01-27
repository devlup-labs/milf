package interfaces

import "context"

type FuncGateway interface {
	ActivateJob(ctx context.Context, funcID string, userID string) (bool, error)
	DeactivateJob(ctx context.Context, funcID string, userID string) (bool, error)      
}
