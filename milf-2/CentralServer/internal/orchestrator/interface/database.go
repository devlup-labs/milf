package interfaces

import (
	"context"
)

type Database interface {
	GetLambdaMetadata(
		ctx context.Context,
		funcID string,
	) (map[string]string, error)
}
