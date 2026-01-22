package domain

import "context"

// LambdaService defines the interface for lambda operations within the gateway
// This is the internal service interface that the core implements
type LambdaService interface {
	StoreLambda(ctx context.Context, req *LambdaStoreRequest) (*LambdaStoreResponse, error)
	ExecuteLambda(ctx context.Context, req *LambdaExecRequest) (*LambdaExecResponse, error)
	GetLambda(ctx context.Context, lambdaID string) (*Lambda, error)
	GetExecution(ctx context.Context, executionID string) (*Execution, error)
}

// CompilerService defines the interface for the compiler module
// This port is implemented by the compiler module
type CompilerService interface {
	// Compile takes source code and runtime, compiles to WASM, and returns the reference
	Compile(ctx context.Context, sourceCode string, runtime RuntimeEnvironment) (wasmRef string, err error)
}

// OrchestratorService defines the interface for the orchestrator module
// This port is implemented by the orchestrator module
type OrchestratorService interface {
	// Execute coordinates execution of the lambda and returns the result
	Execute(ctx context.Context, execution *Execution) (interface{}, error)
}

// LambdaRepository defines the interface for lambda persistence
// This port is implemented by the storage layer
type LambdaRepository interface {
	// Save persists a lambda and returns its ID
	Save(ctx context.Context, lambda *Lambda) error

	// FindByID retrieves a lambda by its ID
	FindByID(ctx context.Context, id string) (*Lambda, error)

	// FindByWasmRef retrieves a lambda by its WASM reference
	FindByWasmRef(ctx context.Context, wasmRef string) (*Lambda, error)

	// Delete removes a lambda by its ID
	Delete(ctx context.Context, id string) error
}

// ExecutionRepository defines the interface for execution persistence
// This port is implemented by the storage layer
type ExecutionRepository interface {
	// Save persists an execution record
	Save(ctx context.Context, execution *Execution) error

	// FindByID retrieves an execution by its ID
	FindByID(ctx context.Context, id string) (*Execution, error)

	// Update updates an existing execution record
	Update(ctx context.Context, execution *Execution) error
}
