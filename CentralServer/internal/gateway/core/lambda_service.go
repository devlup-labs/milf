package core

import (
	"context"
	"time"

	"central_server/internal/gateway/domain"

	"github.com/google/uuid"
)

// LambdaService handles lambda storage and execution business logic
type LambdaService struct {
	lambdaRepo   domain.LambdaRepository
	execRepo     domain.ExecutionRepository
	compiler     domain.CompilerService
	orchestrator domain.OrchestratorService
}

// NewLambdaService creates a new LambdaService with the provided dependencies
func NewLambdaService(
	lambdaRepo domain.LambdaRepository,
	execRepo domain.ExecutionRepository,
	compiler domain.CompilerService,
	orchestrator domain.OrchestratorService,
) *LambdaService {
	return &LambdaService{
		lambdaRepo:   lambdaRepo,
		execRepo:     execRepo,
		compiler:     compiler,
		orchestrator: orchestrator,
	}
}

// StoreLambda processes and stores a new lambda function
func (s *LambdaService) StoreLambda(ctx context.Context, req *domain.LambdaStoreRequest) (*domain.LambdaStoreResponse, error) {
	if err := domain.ValidateStoreRequest(req); err != nil {
		return nil, err
	}

	// Compile source code to WASM via compiler module
	wasmRef, err := s.compiler.Compile(ctx, req.SourceCode, req.Runtime)
	if err != nil {
		return nil, domain.ErrCompilationFailed
	}

	now := time.Now().UTC()
	lambda := &domain.Lambda{
		ID:         uuid.New().String(),
		Name:       req.Name,
		SourceCode: req.SourceCode,
		Runtime:    req.Runtime,
		MemoryMB:   req.MemoryMB,
		RunType:    req.RunType,
		WasmRef:    wasmRef,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.lambdaRepo.Save(ctx, lambda); err != nil {
		return nil, domain.ErrInternalServer
	}

	return &domain.LambdaStoreResponse{
		ID:      lambda.ID,
		Name:    lambda.Name,
		WasmRef: lambda.WasmRef,
		Message: "Lambda stored successfully",
	}, nil
}

// ExecuteLambda initiates execution of a stored lambda
func (s *LambdaService) ExecuteLambda(ctx context.Context, req *domain.LambdaExecRequest) (*domain.LambdaExecResponse, error) {
	if err := domain.ValidateExecRequest(req); err != nil {
		return nil, err
	}

	// Verify lambda exists using the reference id (wasm reference)
	lambda, err := s.lambdaRepo.FindByWasmRef(ctx, req.ReferenceID)
	if err != nil {
		return nil, domain.ErrLambdaNotFound
	}

	now := time.Now().UTC()
	execution := &domain.Execution{
		ID:          uuid.New().String(),
		LambdaID:    lambda.ID,
		ReferenceID: req.ReferenceID,
		Input:       req.Input,
		Status:      domain.ExecutionStatusPending,
		StartedAt:   now,
	}

	// Persist execution record
	if err := s.execRepo.Save(ctx, execution); err != nil {
		return nil, domain.ErrInternalServer
	}

	// Delegate execution to orchestrator and capture result
	result, err := s.orchestrator.Execute(ctx, execution)
	if err != nil {
		execution.Status = domain.ExecutionStatusFailed
		execution.Error = err.Error()
		_ = s.execRepo.Update(ctx, execution)
		return nil, domain.ErrExecutionFailed
	}

	execution.Status = domain.ExecutionStatusCompleted
	execution.Output = result
	_ = s.execRepo.Update(ctx, execution)

	return &domain.LambdaExecResponse{
		ExecutionID: execution.ID,
		Status:      execution.Status,
		Message:     "Execution completed successfully",
		Result:      result,
	}, nil
}

// GetLambda retrieves a lambda by its ID
func (s *LambdaService) GetLambda(ctx context.Context, lambdaID string) (*domain.Lambda, error) {
	lambda, err := s.lambdaRepo.FindByID(ctx, lambdaID)
	if err != nil {
		return nil, domain.ErrLambdaNotFound
	}
	return lambda, nil
}

// GetExecution retrieves an execution by its ID
func (s *LambdaService) GetExecution(ctx context.Context, executionID string) (*domain.Execution, error) {
	execution, err := s.execRepo.FindByID(ctx, executionID)
	if err != nil {
		return nil, err
	}
	return execution, nil
}
