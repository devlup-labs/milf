package storage

import (
	"context"
	"errors"
	"sync"

	authdomain "central_server/internal/auth/domain"
	gwdomain "central_server/internal/gateway/domain"

	"github.com/google/uuid"
)

// MemoryLambdaRepo is an in-memory implementation of LambdaRepository.
type MemoryLambdaRepo struct {
	mu     sync.RWMutex
	byID   map[string]*gwdomain.Lambda
	byWasm map[string]*gwdomain.Lambda
}

func NewMemoryLambdaRepo() *MemoryLambdaRepo {
	return &MemoryLambdaRepo{
		byID:   make(map[string]*gwdomain.Lambda),
		byWasm: make(map[string]*gwdomain.Lambda),
	}
}

func (r *MemoryLambdaRepo) Save(ctx context.Context, lambda *gwdomain.Lambda) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[lambda.ID] = lambda
	r.byWasm[lambda.WasmRef] = lambda
	return nil
}

func (r *MemoryLambdaRepo) FindByID(ctx context.Context, id string) (*gwdomain.Lambda, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.byID[id]
	if !ok {
		return nil, gwdomain.ErrLambdaNotFound
	}
	return l, nil
}

func (r *MemoryLambdaRepo) FindByWasmRef(ctx context.Context, wasmRef string) (*gwdomain.Lambda, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.byWasm[wasmRef]
	if !ok {
		return nil, gwdomain.ErrLambdaNotFound
	}
	return l, nil
}

func (r *MemoryLambdaRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.byID[id]; ok {
		delete(r.byWasm, l.WasmRef)
	}
	delete(r.byID, id)
	return nil
}

// MemoryExecutionRepo is an in-memory implementation of ExecutionRepository.
type MemoryExecutionRepo struct {
	mu   sync.RWMutex
	data map[string]*gwdomain.Execution
}

func NewMemoryExecutionRepo() *MemoryExecutionRepo {
	return &MemoryExecutionRepo{data: make(map[string]*gwdomain.Execution)}
}

func (r *MemoryExecutionRepo) Save(ctx context.Context, e *gwdomain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[e.ID] = e
	return nil
}

func (r *MemoryExecutionRepo) FindByID(ctx context.Context, id string) (*gwdomain.Execution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.data[id]
	if !ok {
		return nil, gwdomain.ErrExecutionFailed
	}
	return e, nil
}

func (r *MemoryExecutionRepo) Update(ctx context.Context, e *gwdomain.Execution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[e.ID] = e
	return nil
}

// MemoryUserRepo is an in-memory implementation of UserRepository.
type MemoryUserRepo struct {
	mu         sync.RWMutex
	byUsername map[string]*authdomain.User
}

func NewMemoryUserRepo() *MemoryUserRepo {
	return &MemoryUserRepo{byUsername: make(map[string]*authdomain.User)}
}

func (r *MemoryUserRepo) Create(ctx context.Context, user *authdomain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byUsername[user.Username]; exists {
		return errors.New("user already exists")
	}
	r.byUsername[user.Username] = user
	return nil
}

func (r *MemoryUserRepo) GetByUsername(ctx context.Context, username string) (*authdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byUsername[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// DummyCompiler is a stub compiler for testing.
type DummyCompiler struct{}

func (DummyCompiler) Compile(ctx context.Context, sourceCode string, runtime gwdomain.RuntimeEnvironment) (string, error) {
	return uuid.New().String(), nil
}

// DummyOrchestrator is a stub orchestrator for testing.
type DummyOrchestrator struct{}

func (DummyOrchestrator) Execute(ctx context.Context, execution *gwdomain.Execution) (interface{}, error) {
	return map[string]interface{}{
		"lambda_id":    execution.LambdaID,
		"reference_id": execution.ReferenceID,
		"input":        execution.Input,
	}, nil
}
