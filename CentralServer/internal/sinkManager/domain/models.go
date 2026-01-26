package domain

import (
	"context"
	"errors"
	"time"
)

// --- Errors ---

var (
	ErrSinkNotFound       = errors.New("sink not found")
	ErrSinkAlreadyExists  = errors.New("sink already registered")
	ErrSinkUnreachable    = errors.New("sink is unreachable")
	ErrInvalidSinkRequest = errors.New("invalid sink request")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTaskDeliveryFailed = errors.New("failed to deliver task to sink")
	ErrResultNotFound     = errors.New("result not found")
	ErrInternalServer     = errors.New("internal server error")
)

// --- Enums ---

type SinkStatus string

const (
	SinkStatusOnline  SinkStatus = "online"
	SinkStatusOffline SinkStatus = "offline"
	SinkStatusBusy    SinkStatus = "busy"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusDelivered TaskStatus = "delivered"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// --- Entities ---

// Sink represents a consumer device that can execute lambda functions
type Sink struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	Password           string     `json:"-"`        // Stored as bcrypt hash, never sent in JSON
	Endpoint           string     `json:"endpoint"` // URL to reach the sink
	RAMAvailableMB     int        `json:"ram_available_mb"`
	StorageAvailableMB int        `json:"storage_available_mb"`
	Status             SinkStatus `json:"status"`
	LastHeartbeat      time.Time  `json:"last_heartbeat"`
	RegisteredAt       time.Time  `json:"registered_at"`
}

// Task represents a lambda function to be executed on a sink
type Task struct {
	ID          string                 `json:"id"`
	ExecutionID string                 `json:"execution_id"` // Links to gateway execution
	LambdaID    string                 `json:"lambda_id"`
	WasmRef     string                 `json:"wasm_ref"`
	Input       map[string]interface{} `json:"input"`
	SinkID      string                 `json:"sink_id"`
	Status      TaskStatus             `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// TaskResult represents the result of an executed task
type TaskResult struct {
	ID          string      `json:"id"`
	TaskID      string      `json:"task_id"`
	ExecutionID string      `json:"execution_id"`
	Output      interface{} `json:"output"`
	Error       string      `json:"error,omitempty"`
	Success     bool        `json:"success"`
	ReceivedAt  time.Time   `json:"received_at"`
}

// --- DTOs ---

// SinkRegisterRequest is the payload for sink registration (like user signup)
type SinkRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Endpoint string `json:"endpoint"`
}

// SinkRegisterResponse is the response after successful registration
type SinkRegisterResponse struct {
	SinkID  string `json:"sink_id"`
	Message string `json:"message"`
}

// SinkLoginRequest is the payload for sink login
type SinkLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SinkLoginResponse is the response after successful login
type SinkLoginResponse struct {
	SinkID  string `json:"sink_id"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

// HeartbeatRequest is sent by the sink every 10 seconds with current stats
type HeartbeatRequest struct {
	SinkID             string `json:"sink_id"`
	RAMAvailableMB     int    `json:"ram_available_mb"`
	StorageAvailableMB int    `json:"storage_available_mb"`
}

// HeartbeatResponse is sent back to the sink
type HeartbeatResponse struct {
	Acknowledged bool   `json:"acknowledged"`
	Message      string `json:"message"`
}

// TaskDeliveryRequest is sent to the sink to execute a task
type TaskDeliveryRequest struct {
	TaskID      string                 `json:"task_id"`
	ExecutionID string                 `json:"execution_id"`
	WasmRef     string                 `json:"wasm_ref"`
	Input       map[string]interface{} `json:"input"`
}

// TaskDeliveryResponse is received from the sink after accepting the task
type TaskDeliveryResponse struct {
	TaskID   string `json:"task_id"`
	Accepted bool   `json:"accepted"`
	Message  string `json:"message"`
}

// TaskResultRequest is received from the sink with execution result
type TaskResultRequest struct {
	TaskID      string      `json:"task_id"`
	ExecutionID string      `json:"execution_id"`
	Output      interface{} `json:"output"`
	Error       string      `json:"error,omitempty"`
	Success     bool        `json:"success"`
}

// TaskResultResponse is sent back to the sink
type TaskResultResponse struct {
	Received bool   `json:"received"`
	Message  string `json:"message"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// --- Validation ---

const (
	MinRAMMB          = 0
	MaxRAMMB          = 65536 // 64GB
	MinStorageMB      = 0
	MaxStorageMB      = 1048576 // 1TB
	MinPasswordLength = 6
	MaxPasswordLength = 128
)

func ValidateRegisterRequest(req *SinkRegisterRequest) error {
	if req.Email == "" {
		return ErrInvalidSinkRequest
	}
	if req.Password == "" || len(req.Password) < MinPasswordLength || len(req.Password) > MaxPasswordLength {
		return ErrInvalidSinkRequest
	}
	if req.Endpoint == "" {
		return ErrInvalidSinkRequest
	}
	return nil
}

func ValidateLoginRequest(req *SinkLoginRequest) error {
	if req.Email == "" || req.Password == "" {
		return ErrInvalidSinkRequest
	}
	return nil
}

func ValidateHeartbeatRequest(req *HeartbeatRequest) error {
	if req.SinkID == "" {
		return ErrInvalidSinkRequest
	}
	if req.RAMAvailableMB < MinRAMMB || req.RAMAvailableMB > MaxRAMMB {
		return ErrInvalidSinkRequest
	}
	if req.StorageAvailableMB < MinStorageMB || req.StorageAvailableMB > MaxStorageMB {
		return ErrInvalidSinkRequest
	}
	return nil
}

func ValidateTaskResultRequest(req *TaskResultRequest) error {
	if req.TaskID == "" || req.ExecutionID == "" {
		return ErrInvalidSinkRequest
	}
	return nil
}

// --- Service Interface (implemented by core) ---

type SinkManagerService interface {
	// RegisterSink registers a new sink with email/password
	RegisterSink(ctx context.Context, req *SinkRegisterRequest) (*SinkRegisterResponse, error)

	// LoginSink authenticates a sink and returns a token
	LoginSink(ctx context.Context, req *SinkLoginRequest) (*SinkLoginResponse, error)

	// UnregisterSink removes a sink from the registry
	UnregisterSink(ctx context.Context, sinkID string) error

	// GetSink retrieves a sink by ID
	GetSink(ctx context.Context, sinkID string) (*Sink, error)

	// GetSinkByEmail retrieves a sink by email
	GetSinkByEmail(ctx context.Context, email string) (*Sink, error)

	// ListSinks returns all registered sinks
	ListSinks(ctx context.Context) ([]*Sink, error)

	// ProcessHeartbeat handles heartbeat from a sink (called every 10 seconds)
	ProcessHeartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error)

	// DeliverTask delivers a task to a specific sink
	DeliverTask(ctx context.Context, task *Task) (*TaskDeliveryResponse, error)

	// ProcessTaskResult handles the result from a sink
	ProcessTaskResult(ctx context.Context, req *TaskResultRequest) (*TaskResultResponse, error)

	// GetTaskResult retrieves a task result by task ID
	GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error)

	// StartHeartbeatMonitor starts the background heartbeat monitoring
	StartHeartbeatMonitor(ctx context.Context, interval time.Duration)

	// StopHeartbeatMonitor stops the background heartbeat monitoring
	StopHeartbeatMonitor()
}

// ResultCallback is used to notify the gateway about task results
type ResultCallback func(ctx context.Context, executionID string, result interface{}, err error)
