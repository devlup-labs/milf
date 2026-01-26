package core

import (
	"context"
	"time"

	"central_server/internal/gateway/domain"
	"central_server/internal/gateway/interfaces"

	"github.com/google/uuid"
)

type LambdaService struct {
	lambdaRepo       interfaces.LambdaRepository
	execRepo         interfaces.ExecutionRepository
	compilationQueue interfaces.CompilationQueueService
	orchestrator     interfaces.OrchestratorService
}

func NewLambdaService(
	lambdaRepo interfaces.LambdaRepository,
	execRepo interfaces.ExecutionRepository,
	compilationQueue interfaces.CompilationQueueService,
	orchestrator interfaces.OrchestratorService,
) *LambdaService {
	return &LambdaService{
		lambdaRepo:       lambdaRepo,
		execRepo:         execRepo,
		compilationQueue: compilationQueue,
		orchestrator:     orchestrator,
	}
}

func (s *LambdaService) StoreLambda(ctx context.Context, req *domain.LambdaStoreRequest) (*domain.LambdaStoreResponse, error) {
	if err := domain.ValidateStoreRequest(req); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	lambdaID := uuid.New().String()
	jobID := uuid.New().String()

	// Create the lambda record with pending compilation status
	lambda := &domain.Lambda{
		ID:         lambdaID,
		Name:       req.Name,
		SourceCode: req.SourceCode,
		Runtime:    req.Runtime,
		MemoryMB:   req.MemoryMB,
		RunType:    req.RunType,
		WasmRef:    "", // Will be populated after compilation completes
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.lambdaRepo.Save(ctx, lambda); err != nil {
		return nil, domain.ErrInternalServer
	}

	// Create and enqueue the compilation job
	compilationJob := &domain.CompilationJob{
		ID:         jobID,
		LambdaID:   lambdaID,
		SourceCode: req.SourceCode,
		Runtime:    req.Runtime,
		Priority:   0, // Default priority
		CreatedAt:  now,
	}

	if err := s.compilationQueue.Enqueue(ctx, compilationJob); err != nil {
		// Rollback: delete the lambda if we can't enqueue the job
		_ = s.lambdaRepo.Delete(ctx, lambdaID)
		return nil, domain.ErrQueueFailed
	}

	return &domain.LambdaStoreResponse{
		ID:                lambdaID,
		Name:              lambda.Name,
		CompilationJobID:  jobID,
		CompilationStatus: domain.CompilationStatusQueued,
		Message:           "Lambda stored successfully. Compilation job has been queued.",
	}, nil
}

func (s *LambdaService) ExecuteLambda(ctx context.Context, req *domain.LambdaExecRequest) (*domain.LambdaExecResponse, error) {
	if err := domain.ValidateExecRequest(req); err != nil {
		return nil, err
	}

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

	if err := s.execRepo.Save(ctx, execution); err != nil {
		return nil, domain.ErrInternalServer
	}

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

func (s *LambdaService) GetLambda(ctx context.Context, lambdaID string) (*domain.Lambda, error) {
	lambda, err := s.lambdaRepo.FindByID(ctx, lambdaID)
	if err != nil {
		return nil, domain.ErrLambdaNotFound
	}
	return lambda, nil
}

func (s *LambdaService) GetExecution(ctx context.Context, executionID string) (*domain.Execution, error) {
	execution, err := s.execRepo.FindByID(ctx, executionID)
	if err != nil {
		return nil, err
	}
	return execution, nil
}

func (s *LambdaService) GetCompilationStatus(ctx context.Context, jobID string) (*domain.CompilationStatusResponse, error) {
	if jobID == "" {
		return nil, domain.ErrInvalidRequest
	}

	status, err := s.compilationQueue.GetJobStatus(ctx, jobID)
	if err != nil {
		return nil, domain.ErrJobNotFound
	}

	return &domain.CompilationStatusResponse{
		JobID:       status.JobID,
		LambdaID:    status.LambdaID,
		Status:      status.Status,
		WasmRef:     status.WasmRef,
		Error:       status.Error,
		QueuedAt:    status.QueuedAt,
		StartedAt:   status.StartedAt,
		CompletedAt: status.CompletedAt,
	}, nil
}
