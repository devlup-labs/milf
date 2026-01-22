package domain

// LambdaStoreRequest represents the request payload for storing a lambda
// Only accepts: source code, runtime, RAM constraint, and run type
type LambdaStoreRequest struct {
	Name       string             `json:"name" validate:"required,min=1,max=128"`
	SourceCode string             `json:"source_code" validate:"required"`
	Runtime    RuntimeEnvironment `json:"runtime" validate:"required"`
	MemoryMB   int                `json:"memory_mb" validate:"required,min=64,max=4096"`
	RunType    RunType            `json:"run_type" validate:"required"`
}

// LambdaStoreResponse represents the response after storing a lambda
type LambdaStoreResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	WasmRef string `json:"wasm_ref"`
	Message string `json:"message"`
}

// LambdaExecRequest represents the request payload for executing a lambda
// Accepts the reference id returned from store and input payload
type LambdaExecRequest struct {
	ReferenceID string                 `json:"reference_id" validate:"required"`
	Input       map[string]interface{} `json:"input"`
}

// LambdaExecResponse represents the response after executing a lambda
type LambdaExecResponse struct {
	ExecutionID string          `json:"execution_id"`
	Status      ExecutionStatus `json:"status"`
	Message     string          `json:"message"`
	Result      interface{}     `json:"result,omitempty"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
