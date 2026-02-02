package interfaces

import (
	"central_server/internal/queueService/domain"
	"context"
)

type QueueService interface {
	Enqueue(ctx context.Context, jobID string, funcID string, metaData map[string]string) (error, bool)
	DispatchOrEnqueue(ctx context.Context, jobID string, funcID string, metaData map[string]string) (error, bool)
	ClaimNextJob(allowedRam int) (*CandidateJob, error)
}

type CandidateJob struct {
	Job     domain.Job
	QueueID string
}
