package interfaces

import (
	"context"
	"net/http"

	"central_server/internal/gateway/domain"
	"central_server/internal/gateway/handler"
)

// --- External Service Interfaces (Ports) ---

// CompilationQueueService defines the interface for enqueueing compilation jobs.
type CompilationQueueService interface {
	// Enqueue adds a compilation job to the queue and returns a job ID.
	Enqueue(ctx context.Context, job *domain.CompilationJob) error
	// GetJobStatus retrieves the current status of a compilation job.
	GetJobStatus(ctx context.Context, jobID string) (*domain.CompilationJobStatus, error)
}

// CompilerService defines the interface for the compiler module (synchronous compilation).
type CompilerService interface {
	Compile(ctx context.Context, sourceCode string, runtime domain.RuntimeEnvironment) (wasmRef string, err error)
}

// OrchestratorService defines the interface for the orchestrator module.
type OrchestratorService interface {
	Execute(ctx context.Context, execution *domain.Execution) (interface{}, error)
}

// LambdaRepository defines the interface for lambda persistence.
type LambdaRepository interface {
	Save(ctx context.Context, lambda *domain.Lambda) error
	FindByID(ctx context.Context, id string) (*domain.Lambda, error)
	FindByWasmRef(ctx context.Context, wasmRef string) (*domain.Lambda, error)
	Delete(ctx context.Context, id string) error
}

// ExecutionRepository defines the interface for execution persistence.
type ExecutionRepository interface {
	Save(ctx context.Context, execution *domain.Execution) error
	FindByID(ctx context.Context, id string) (*domain.Execution, error)
	Update(ctx context.Context, execution *domain.Execution) error
}

// --- Router ---

type Router struct {
	mux            *http.ServeMux
	handler        *handler.LambdaHandler
	authMiddleware func(http.Handler) http.Handler
}

func NewRouter(h *handler.LambdaHandler, authMiddleware func(http.Handler) http.Handler) *Router {
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

	r.mux.HandleFunc("POST /api/v1/lambdas", wrap(r.handler.Store))
	r.mux.HandleFunc("GET /api/v1/lambdas/{id}", wrap(r.handler.Get))
	r.mux.HandleFunc("POST /api/v1/lambdas/execute", wrap(r.handler.Execute))
	r.mux.HandleFunc("GET /api/v1/executions/{id}", wrap(r.handler.GetExecution))
	r.mux.HandleFunc("GET /api/v1/compilations/{jobId}", wrap(r.handler.GetCompilationStatus))

	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	return r.mux
}
