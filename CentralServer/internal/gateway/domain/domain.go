package domain

import (
	"context"
	"errors"
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
	Name       string             `json:"name"`
	SourceCode string             `json:"source_code"`
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

// --- DTOs ---

type LambdaStoreRequest struct {
	Name       string             `json:"name"`
	SourceCode string             `json:"source_code"`
	Runtime    RuntimeEnvironment `json:"runtime"`
	MemoryMB   int                `json:"memory_mb"`
	RunType    RunType            `json:"run_type"`
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
	if req.Name == "" || req.SourceCode == "" {
		return ErrInvalidRequest
	}
	if len(req.Name) < MinNameLength || len(req.Name) > MaxNameLength {
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
	ExecuteLambda(ctx context.Context, req *LambdaExecRequest) (*LambdaExecResponse, error)
	GetLambda(ctx context.Context, lambdaID string) (*Lambda, error)
	GetExecution(ctx context.Context, executionID string) (*Execution, error)
}
