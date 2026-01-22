package domain

import "time"

// RuntimeEnvironment represents the supported runtime environments for lambdas
type RuntimeEnvironment string

const (
	RuntimeGo         RuntimeEnvironment = "go"
	RuntimeRust       RuntimeEnvironment = "rust"
	RuntimePython     RuntimeEnvironment = "python"
	RuntimeJavaScript RuntimeEnvironment = "javascript"
)

// RunType defines how the lambda should be executed
// on-command: invoked on demand; periodic: invoked on a schedule managed elsewhere
type RunType string

const (
	RunTypeOnCommand RunType = "on_command"
	RunTypePeriodic  RunType = "periodic"
)

// Lambda represents a stored lambda function
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

// Execution represents a lambda execution request
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

// ExecutionStatus represents the current state of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)
