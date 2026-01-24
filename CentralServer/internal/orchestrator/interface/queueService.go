package interfaces

import "context"

type QueueService interface {
	Enqueue(
		ctx context.Context,
		jobID string,
		funcID string,
		metaData map[string]string,
	) error
}