package storage

import (
	"context"
	"errors"
	"sync"

	authdomain "central_server/internal/auth/domain"
	gwdomain "central_server/internal/gateway/domain"
	sinkdomain "central_server/internal/sinkManager/domain"

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

// DummyCompilationQueue is a stub compilation queue for testing.
// It stores jobs in memory and simulates async compilation.
type DummyCompilationQueue struct {
	mu   sync.RWMutex
	jobs map[string]*gwdomain.CompilationJobStatus
}

func NewDummyCompilationQueue() *DummyCompilationQueue {
	return &DummyCompilationQueue{
		jobs: make(map[string]*gwdomain.CompilationJobStatus),
	}
}

func (q *DummyCompilationQueue) Enqueue(ctx context.Context, job *gwdomain.CompilationJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Create a job status entry
	status := &gwdomain.CompilationJobStatus{
		JobID:    job.ID,
		LambdaID: job.LambdaID,
		Status:   gwdomain.CompilationStatusQueued,
		QueuedAt: job.CreatedAt,
	}
	q.jobs[job.ID] = status

	// In a real implementation, this would be processed by a worker.
	// For the dummy, we immediately mark it as completed with a generated wasm ref.
	go func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		if s, ok := q.jobs[job.ID]; ok {
			s.Status = gwdomain.CompilationStatusCompleted
			s.WasmRef = uuid.New().String()
		}
	}()

	return nil
}

func (q *DummyCompilationQueue) GetJobStatus(ctx context.Context, jobID string) (*gwdomain.CompilationJobStatus, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	status, ok := q.jobs[jobID]
	if !ok {
		return nil, gwdomain.ErrJobNotFound
	}
	return status, nil
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

// --- Sink Manager Repositories ---

// MemorySinkRepo is an in-memory implementation of SinkRepository.
type MemorySinkRepo struct {
	mu      sync.RWMutex
	byID    map[string]*sinkdomain.Sink
	byEmail map[string]*sinkdomain.Sink
}

func NewMemorySinkRepo() *MemorySinkRepo {
	return &MemorySinkRepo{
		byID:    make(map[string]*sinkdomain.Sink),
		byEmail: make(map[string]*sinkdomain.Sink),
	}
}

func (r *MemorySinkRepo) Save(ctx context.Context, sink *sinkdomain.Sink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[sink.ID] = sink
	r.byEmail[sink.Email] = sink
	return nil
}

func (r *MemorySinkRepo) FindByID(ctx context.Context, id string) (*sinkdomain.Sink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sink, ok := r.byID[id]
	if !ok {
		return nil, sinkdomain.ErrSinkNotFound
	}
	return sink, nil
}

func (r *MemorySinkRepo) FindByEmail(ctx context.Context, email string) (*sinkdomain.Sink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sink, ok := r.byEmail[email]
	if !ok {
		return nil, sinkdomain.ErrSinkNotFound
	}
	return sink, nil
}

func (r *MemorySinkRepo) FindAll(ctx context.Context) ([]*sinkdomain.Sink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sinks := make([]*sinkdomain.Sink, 0, len(r.byID))
	for _, sink := range r.byID {
		sinks = append(sinks, sink)
	}
	return sinks, nil
}

func (r *MemorySinkRepo) Update(ctx context.Context, sink *sinkdomain.Sink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[sink.ID]; !ok {
		return sinkdomain.ErrSinkNotFound
	}
	r.byID[sink.ID] = sink
	r.byEmail[sink.Email] = sink
	return nil
}

func (r *MemorySinkRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sink, ok := r.byID[id]; ok {
		delete(r.byEmail, sink.Email)
	}
	delete(r.byID, id)
	return nil
}

// MemoryTaskRepo is an in-memory implementation of TaskRepository.
type MemoryTaskRepo struct {
	mu            sync.RWMutex
	byID          map[string]*sinkdomain.Task
	byExecutionID map[string]*sinkdomain.Task
	bySinkID      map[string][]*sinkdomain.Task
}

func NewMemoryTaskRepo() *MemoryTaskRepo {
	return &MemoryTaskRepo{
		byID:          make(map[string]*sinkdomain.Task),
		byExecutionID: make(map[string]*sinkdomain.Task),
		bySinkID:      make(map[string][]*sinkdomain.Task),
	}
}

func (r *MemoryTaskRepo) Save(ctx context.Context, task *sinkdomain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[task.ID] = task
	r.byExecutionID[task.ExecutionID] = task
	r.bySinkID[task.SinkID] = append(r.bySinkID[task.SinkID], task)
	return nil
}

func (r *MemoryTaskRepo) FindByID(ctx context.Context, id string) (*sinkdomain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.byID[id]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (r *MemoryTaskRepo) FindByExecutionID(ctx context.Context, executionID string) (*sinkdomain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.byExecutionID[executionID]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (r *MemoryTaskRepo) FindBySinkID(ctx context.Context, sinkID string) ([]*sinkdomain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tasks, ok := r.bySinkID[sinkID]
	if !ok {
		return []*sinkdomain.Task{}, nil
	}
	return tasks, nil
}

func (r *MemoryTaskRepo) Update(ctx context.Context, task *sinkdomain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[task.ID]; !ok {
		return errors.New("task not found")
	}
	r.byID[task.ID] = task
	r.byExecutionID[task.ExecutionID] = task
	return nil
}

func (r *MemoryTaskRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, id)
	return nil
}

// MemoryTaskResultRepo is an in-memory implementation of TaskResultRepository.
type MemoryTaskResultRepo struct {
	mu            sync.RWMutex
	byTaskID      map[string]*sinkdomain.TaskResult
	byExecutionID map[string]*sinkdomain.TaskResult
}

func NewMemoryTaskResultRepo() *MemoryTaskResultRepo {
	return &MemoryTaskResultRepo{
		byTaskID:      make(map[string]*sinkdomain.TaskResult),
		byExecutionID: make(map[string]*sinkdomain.TaskResult),
	}
}

func (r *MemoryTaskResultRepo) Save(ctx context.Context, result *sinkdomain.TaskResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byTaskID[result.TaskID] = result
	r.byExecutionID[result.ExecutionID] = result
	return nil
}

func (r *MemoryTaskResultRepo) FindByTaskID(ctx context.Context, taskID string) (*sinkdomain.TaskResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.byTaskID[taskID]
	if !ok {
		return nil, sinkdomain.ErrResultNotFound
	}
	return result, nil
}

func (r *MemoryTaskResultRepo) FindByExecutionID(ctx context.Context, executionID string) (*sinkdomain.TaskResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.byExecutionID[executionID]
	if !ok {
		return nil, sinkdomain.ErrResultNotFound
	}
	return result, nil
}

// DummySinkClient is a stub sink client for testing.
type DummySinkClient struct{}

func (DummySinkClient) SendHeartbeat(ctx context.Context, sink *sinkdomain.Sink) (*sinkdomain.HeartbeatRequest, error) {
	return &sinkdomain.HeartbeatRequest{
		SinkID:             sink.ID,
		RAMAvailableMB:     sink.RAMAvailableMB,
		StorageAvailableMB: sink.StorageAvailableMB,
	}, nil
}

func (DummySinkClient) DeliverTask(ctx context.Context, sink *sinkdomain.Sink, task *sinkdomain.TaskDeliveryRequest) (*sinkdomain.TaskDeliveryResponse, error) {
	return &sinkdomain.TaskDeliveryResponse{
		TaskID:   task.TaskID,
		Accepted: true,
		Message:  "Task accepted for execution",
	}, nil
}
