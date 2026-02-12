package interfaces

import (
	"central_server/internal/sinkManager/domain"
	"context"
)

type SinkClient interface {
	DeliverTask(ctx context.Context, sink *domain.Sink, task *domain.TaskDeliveryRequest) (*domain.TaskDeliveryResponse, error)
}
