package interfaces

import (
	"context"
)

type Database interface {
	GetLambda(
		ctx context.Context,
		funcID string,
	) (map[string]string, error)
}
