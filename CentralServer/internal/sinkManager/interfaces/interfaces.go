package interfaces

import (
	"context"
	"net/http"

	"central_server/internal/sinkManager/domain"
	"central_server/internal/sinkManager/handler"
)

// --- Repository Interfaces ---

// SinkRepository defines the interface for sink persistence
type SinkRepository interface {
	Save(ctx context.Context, sink *domain.Sink) error
	FindByID(ctx context.Context, id string) (*domain.Sink, error)
	FindByEmail(ctx context.Context, email string) (*domain.Sink, error)
	FindAll(ctx context.Context) ([]*domain.Sink, error)
	Update(ctx context.Context, sink *domain.Sink) error
	Delete(ctx context.Context, id string) error
}

// TaskRepository defines the interface for task persistence
type TaskRepository interface {
	Save(ctx context.Context, task *domain.Task) error
	FindByID(ctx context.Context, id string) (*domain.Task, error)
	FindByExecutionID(ctx context.Context, executionID string) (*domain.Task, error)
	FindBySinkID(ctx context.Context, sinkID string) ([]*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id string) error
}

// TaskResultRepository defines the interface for task result persistence
type TaskResultRepository interface {
	Save(ctx context.Context, result *domain.TaskResult) error
	FindByTaskID(ctx context.Context, taskID string) (*domain.TaskResult, error)
	FindByExecutionID(ctx context.Context, executionID string) (*domain.TaskResult, error)
}

// --- External Service Interfaces (Ports) ---

// SinkClient defines the interface for communicating with sink devices
type SinkClient interface {
	// SendHeartbeat sends a heartbeat request to the sink and validates response
	SendHeartbeat(ctx context.Context, sink *domain.Sink) (*domain.HeartbeatRequest, error)

	// DeliverTask sends a task to the sink for execution
	DeliverTask(ctx context.Context, sink *domain.Sink, task *domain.TaskDeliveryRequest) (*domain.TaskDeliveryResponse, error)
}

// QueueService defines the interface to interact with the queue service module
type QueueService interface {
	// GetNextTask retrieves the next task to be executed
	GetNextTask(ctx context.Context) (*domain.Task, error)

	// AcknowledgeTask marks a task as delivered
	AcknowledgeTask(ctx context.Context, taskID string) error
}

// --- Router ---

type Router struct {
	mux            *http.ServeMux
	handler        *handler.SinkHandler
	authMiddleware func(http.Handler) http.Handler
}

func NewRouter(h *handler.SinkHandler, authMiddleware func(http.Handler) http.Handler) *Router {
	return &Router{
		mux:            http.NewServeMux(),
		handler:        h,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) Setup() http.Handler {
	wrap := func(fn http.HandlerFunc) http.HandlerFunc {
		if r.authMiddleware == nil {
			return fn
		}
		return func(w http.ResponseWriter, req *http.Request) {
			r.authMiddleware(http.HandlerFunc(fn)).ServeHTTP(w, req)
		}
	}

	// Sink auth endpoints (no server auth required)
	r.mux.HandleFunc("POST /api/v1/sinks/register", r.handler.Register)
	r.mux.HandleFunc("POST /api/v1/sinks/login", r.handler.Login)

	// Sink operational endpoints (no server auth, sink uses its own token)
	r.mux.HandleFunc("POST /api/v1/sinks/heartbeat", r.handler.Heartbeat)
	r.mux.HandleFunc("POST /api/v1/sinks/result", r.handler.SubmitResult)

	// Server-only endpoints (auth required)
	r.mux.HandleFunc("GET /api/v1/sinks", wrap(r.handler.ListSinks))
	r.mux.HandleFunc("GET /api/v1/sinks/{id}", wrap(r.handler.GetSink))
	r.mux.HandleFunc("DELETE /api/v1/sinks/{id}", wrap(r.handler.UnregisterSink))
	r.mux.HandleFunc("GET /api/v1/tasks/{id}/result", wrap(r.handler.GetTaskResult))

	return r.mux
}
