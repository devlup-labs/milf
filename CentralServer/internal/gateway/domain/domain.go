package domain

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"time"
)

// --- Errors ---

var (
	ErrLambdaNotFound    = errors.New("lambda not found")
	ErrInvalidRuntime    = errors.New("invalid runtime environment")
	ErrInvalidRunType    = errors.New("invalid run type")
	ErrCompilationFailed = errors.New("compilation failed")
	ErrExecutionFailed   = errors.New("execution failed")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInternalServer    = errors.New("internal server error")
)

// --- Enums ---

type RuntimeEnvironment string

const (
	RuntimeGo         RuntimeEnvironment = "go"
	RuntimeRust       RuntimeEnvironment = "rust"
	RuntimePython     RuntimeEnvironment = "python"
	RuntimeJavaScript RuntimeEnvironment = "javascript"
	RuntimeCpp        RuntimeEnvironment = "cpp"
	RuntimeC          RuntimeEnvironment = "c"
)

type RunType string

const (
	RunTypeOnCommand RunType = "on_command"
	RunTypePeriodic  RunType = "periodic"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

// --- Entities ---

type Lambda struct {
	ID         string             `json:"id"`
	UserID     string             `json:"user_id"`
	Name       string             `json:"name"`
	SourceCode []byte             `json:"source_code"`
	Runtime    RuntimeEnvironment `json:"runtime"`
	MemoryMB   int                `json:"memory_mb"`
	RunType    RunType            `json:"run_type"`
	WasmRef    string             `json:"wasm_ref"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type Execution struct {
	ID          string                 `json:"id"`
	LambdaID    string                 `json:"lambda_id"`
	ReferenceID string                 `json:"reference_id"`
	Input       map[string]interface{} `json:"input"`
	Status      ExecutionStatus        `json:"status"`
	Output      interface{}            `json:"output"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	FinishedAt  *time.Time             `json:"finished_at,omitempty"`
}

type CompilationQueueObject struct {
	FuncID string
}

type CompilationQueue struct {
	Jobs    *list.List
	JobsMap map[string]*list.Element
	mu      sync.Mutex
	cond    *sync.Cond
}

func (c *CompilationQueue) AddJob(job *CompilationQueueObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.JobsMap[job.FuncID]; exists {
		return errors.New("job already exists in queue")
	}
	e := c.Jobs.PushBack(*job)
	c.JobsMap[job.FuncID] = e

	// Signal waiting consumers
	c.cond.Signal()
	return nil
}

func (c *CompilationQueue) Dequeue() *CompilationQueueObject {
	c.mu.Lock()
	defer c.mu.Unlock()

	for c.Jobs.Len() == 0 {
		c.cond.Wait()
	}

	e := c.Jobs.Front()
	c.Jobs.Remove(e)
	job := e.Value.(CompilationQueueObject)
	delete(c.JobsMap, job.FuncID)

	return &job
}

func NewCompilationQueue() *CompilationQueue {
	q := &CompilationQueue{
		Jobs:    list.New(),
		JobsMap: make(map[string]*list.Element),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// --- DTOs ---

type LambdaStoreRequest struct {
	UserID     string             `json:"user_id"`
	FuncID     string             `json:"func_id"`
	SourceCode []byte             `json:"source_code"`
	Runtime    RuntimeEnvironment `json:"runtime"`
	MemoryMB   int                `json:"memory_mb"`
	RunType    RunType            `json:"run_type"`
	MetaData   map[string]string  `json:"metadata,omitempty"`
}

type LambdaStoreResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	WasmRef string `json:"wasm_ref"`
	Message string `json:"message"`
}

type LambdaExecRequest struct {
	ReferenceID string                 `json:"reference_id"`
	Input       map[string]interface{} `json:"input"`
}

type LambdaExecResponse struct {
	ExecutionID string          `json:"execution_id"`
	Status      ExecutionStatus `json:"status"`
	Message     string          `json:"message"`
	Result      interface{}     `json:"result,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// --- Validation ---

const (
	MinMemoryMB   = 64
	MaxMemoryMB   = 4096
	MinNameLength = 1
	MaxNameLength = 128
)

func ValidateStoreRequest(req *LambdaStoreRequest) error {
	if req.FuncID == "" || len(req.SourceCode) == 0 || req.UserID == "" {
		return ErrInvalidRequest
	}
	if len(req.FuncID) < MinNameLength || len(req.FuncID) > MaxNameLength {
		return ErrInvalidRequest
	}
	if !IsValidRuntime(req.Runtime) {
		return ErrInvalidRuntime
	}
	if !IsValidRunType(req.RunType) {
		return ErrInvalidRunType
	}
	if req.MemoryMB < MinMemoryMB || req.MemoryMB > MaxMemoryMB {
		return ErrInvalidRequest
	}
	return nil
}

func ValidateExecRequest(req *LambdaExecRequest) error {
	if req.ReferenceID == "" {
		return ErrInvalidRequest
	}
	return nil
}

func IsValidRuntime(rt RuntimeEnvironment) bool {
	switch rt {
	case RuntimeGo, RuntimeRust, RuntimePython, RuntimeJavaScript:
		return true
	default:
		return false
	}
}

func IsValidRunType(rt RunType) bool {
	switch rt {
	case RunTypeOnCommand, RunTypePeriodic:
		return true
	default:
		return false
	}
}

// --- Service Interface (implemented by core) ---

type LambdaService interface {
	StoreLambda(ctx context.Context, req *LambdaStoreRequest) (*LambdaStoreResponse, error)
	TriggerLambda(ctx context.Context, req *LambdaExecRequest) (*LambdaExecResponse, error)
	ActivateLambda(ctx context.Context, req *LambdaExecRequest) (*LambdaExecResponse, error)
	DeactivateLambda(ctx context.Context, req *LambdaExecRequest) (*LambdaExecResponse, error)
	GetLambda(ctx context.Context, lambdaID string) (*Lambda, error)
	GetExecution(ctx context.Context, executionID string) (*Execution, error)
	ActivateJob(ctx context.Context, funcID string, userID string) (bool, error)
	ExecuteJob(ctx context.Context, funcID string, input string) (bool, error)
}
