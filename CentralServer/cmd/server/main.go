package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"central_server/internal/gateway"
	"central_server/internal/gateway/domain"

	"github.com/google/uuid"
)

// --- In-memory implementations for testing ---

type inMemoryLambdaRepo struct {
	mu     sync.RWMutex
	byID   map[string]*domain.Lambda
	byWasm map[string]*domain.Lambda
}

func newInMemoryLambdaRepo() *inMemoryLambdaRepo {
	return &inMemoryLambdaRepo{byID: make(map[string]*domain.Lambda), byWasm: make(map[string]*domain.Lambda)}
}

func (r *inMemoryLambdaRepo) Save(ctx context.Context, lambda *domain.Lambda) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[lambda.ID] = lambda
	r.byWasm[lambda.WasmRef] = lambda
	return nil
}

func (r *inMemoryLambdaRepo) FindByID(ctx context.Context, id string) (*domain.Lambda, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrLambdaNotFound
	}
	return l, nil
}

func (r *inMemoryLambdaRepo) FindByWasmRef(ctx context.Context, wasmRef string) (*domain.Lambda, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.byWasm[wasmRef]
	if !ok {
		return nil, domain.ErrLambdaNotFound
	}
	return l, nil
}

func (r *inMemoryLambdaRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.byID[id]; ok {
		delete(r.byWasm, l.WasmRef)
	}
	delete(r.byID, id)
	return nil
}

type inMemoryExecutionRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.Execution
}

func newInMemoryExecutionRepo() *inMemoryExecutionRepo {
	return &inMemoryExecutionRepo{data: make(map[string]*domain.Execution)}
}

func (r *inMemoryExecutionRepo) Save(ctx context.Context, e *domain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[e.ID] = e
	return nil
}

func (r *inMemoryExecutionRepo) FindByID(ctx context.Context, id string) (*domain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.data[id]
	if !ok {
		return nil, domain.ErrExecutionFailed
	}
	return e, nil
}

func (r *inMemoryExecutionRepo) Update(ctx context.Context, e *domain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[e.ID] = e
	return nil
}

type dummyCompiler struct{}

func (dummyCompiler) Compile(ctx context.Context, sourceCode string, runtime domain.RuntimeEnvironment) (string, error) {
	return uuid.New().String(), nil
}

type dummyOrchestrator struct{}

func (dummyOrchestrator) Execute(ctx context.Context, execution *domain.Execution) (interface{}, error) {
	// Echo input as result for testing
	return map[string]interface{}{
		"lambda_id":    execution.LambdaID,
		"reference_id": execution.ReferenceID,
		"input":        execution.Input,
		"executed_at":  time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func main() {
	lambdaRepo := newInMemoryLambdaRepo()
	execRepo := newInMemoryExecutionRepo()
	compiler := dummyCompiler{}
	orchestrator := dummyOrchestrator{}

	gw := gateway.New(lambdaRepo, execRepo, compiler, orchestrator)
	handler := gw.Handler()

	addr := ":8080"
	log.Printf("gateway test server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
