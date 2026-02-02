package domain

import (
	"context"
	"errors"
	"time"
)

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

type Sink struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	Password           string     `json:"-"`
	Endpoint           string     `json:"endpoint"`
	RAMAvailableMB     int        `json:"ram_available_mb"`
	StorageAvailableMB int        `json:"storage_available_mb"`
	Status             SinkStatus `json:"status"`
	LastHeartbeat      time.Time  `json:"last_heartbeat"`
	RegisteredAt       time.Time  `json:"registered_at"`
}

type Task struct {
	ExecutionID string                 `json:"execution_id"`
	LambdaID    string                 `json:"lambda_id"`
	WasmRef     string                 `json:"wasm_ref"`
	Input       map[string]interface{} `json:"input"`
	SinkID      string                 `json:"sink_id"`
	Status      TaskStatus             `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	DeliveredAt *time.Time             `json:"delivered_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

type TaskResult struct {
	ExecutionID string      `json:"execution_id"`
	Output      interface{} `json:"output"`
	Error       string      `json:"error,omitempty"`
	Success     bool        `json:"success"`
	ReceivedAt  time.Time   `json:"received_at"`
}

type SinkRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Endpoint string `json:"endpoint"`
}

type SinkRegisterResponse struct {
	SinkID  string `json:"sink_id"`
	Message string `json:"message"`
}

type SinkLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SinkLoginResponse struct {
	SinkID  string `json:"sink_id"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

type HeartbeatRequest struct {
	SinkID             string `json:"sink_id"`
	RAMAvailableMB     int    `json:"ram_available_mb"`
	StorageAvailableMB int    `json:"storage_available_mb"`
}

type HeartbeatResponse struct {
	Acknowledged bool   `json:"acknowledged"`
	Message      string `json:"message"`
}

type TaskDeliveryRequest struct {
	ExecutionID string                 `json:"execution_id"`
	WasmRef     string                 `json:"wasm_ref"`
	Input       map[string]interface{} `json:"input"`
}

type TaskDeliveryResponse struct {
	ExecutionID string `json:"execution_id"`
	Accepted    bool   `json:"accepted"`
	Message     string `json:"message"`
}

type TaskResultRequest struct {
	ExecutionID string      `json:"execution_id"`
	Output      interface{} `json:"output"`
	Error       string      `json:"error,omitempty"`
	Success     bool        `json:"success"`
}

type TaskResultResponse struct {
	Received bool   `json:"received"`
	Message  string `json:"message"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

const (
	MinRAMMB          = 0
	MaxRAMMB          = 65536
	MinStorageMB      = 0
	MaxStorageMB      = 1048576
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
	if req.ExecutionID == "" {
		return ErrInvalidSinkRequest
	}
	return nil
}

type SinkManagerService interface {
	RegisterSink(ctx context.Context, req *SinkRegisterRequest) (*SinkRegisterResponse, error)
	LoginSink(ctx context.Context, req *SinkLoginRequest) (*SinkLoginResponse, error)
	UnregisterSink(ctx context.Context, sinkID string) error
	GetSink(ctx context.Context, sinkID string) (*Sink, error)
	GetSinkByEmail(ctx context.Context, email string) (*Sink, error)
	ListSinks(ctx context.Context) ([]*Sink, error)
	ProcessHeartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error)
	DeliverTask(ctx context.Context, task *Task) (*TaskDeliveryResponse, error)
	ProcessTaskResult(ctx context.Context, req *TaskResultRequest) (*TaskResultResponse, error)
	GetTaskResult(ctx context.Context, executionID string) (*TaskResult, error)
	StartStaleDetector(ctx context.Context, staleThreshold time.Duration)
	StopStaleDetector()
	GetSinkForLambda(ctx context.Context, lambdaID string) (string, bool)
}

type ResultCallback func(ctx context.Context, executionID string, result interface{}, err error)
