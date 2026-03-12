package interfaces

import (
	"central_server/internal/sinkManager/domain"
	"context"
)

type SinkRepository interface {
	Save(ctx context.Context, sink *domain.Sink) error
	FindByID(ctx context.Context, id string) (*domain.Sink, error)
	FindByEmail(ctx context.Context, email string) (*domain.Sink, error)
	FindAll(ctx context.Context) ([]*domain.Sink, error)
	Update(ctx context.Context, sink *domain.Sink) error
	Delete(ctx context.Context, id string) error
}

type TaskRepository interface {
	Save(ctx context.Context, task *domain.Task) error
	FindByExecutionID(ctx context.Context, executionID string) (*domain.Task, error)
	FindBySinkID(ctx context.Context, sinkID string) ([]*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, executionID string) error
}

type TaskResultRepository interface {
	Save(ctx context.Context, result *domain.TaskResult) error
	FindByExecutionID(ctx context.Context, executionID string) (*domain.TaskResult, error)
}
