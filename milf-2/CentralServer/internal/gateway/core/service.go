package core

import (
	"central_server/internal/gateway/domain"
	"central_server/internal/gateway/interfaces"
	"central_server/utils"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type LambdaService struct {
	gatewayDB     interfaces.FuncGatewayDB
	compilerRepo  interfaces.CompilerDB
	orchestrator  interfaces.OrchestratorService
	compilerQueue *domain.CompilationQueue
	executionRepo interfaces.ExecutionRepository
}

func NewLambdaService(
	gatewayDB interfaces.FuncGatewayDB,
	compilerRepo interfaces.CompilerDB,
	orchestrator interfaces.OrchestratorService,
	compilerQueue *domain.CompilationQueue,
	executionRepo interfaces.ExecutionRepository,
) *LambdaService {
	return &LambdaService{
		gatewayDB:     gatewayDB,
		compilerRepo:  compilerRepo,
		orchestrator:  orchestrator,
		compilerQueue: compilerQueue,
		executionRepo: executionRepo,
	}
}

func (s *LambdaService) SetOrchestrator(orch interfaces.OrchestratorService) {
	s.orchestrator = orch
}

// StoreLambda implements domain.LambdaService
func (s *LambdaService) StoreLambda(ctx context.Context, req *domain.LambdaStoreRequest) (*domain.LambdaStoreResponse, error) {
	// Reusing StoreandQueue logic
	_, err := s.StoreandQueue(ctx, req)
	if err != nil {
		return nil, err
	}
	// Return dummy response or constructed from req
	return &domain.LambdaStoreResponse{
		ID:      req.FuncID,
		Name:    req.FuncID,
		Message: "Stored and Queued",
	}, nil
}

// TriggerLambda implements domain.LambdaService
func (s *LambdaService) TriggerLambda(ctx context.Context, req *domain.LambdaExecRequest) (*domain.LambdaExecResponse, error) {
	trigID := uuid.New().String()

	// Create execution record in database
	execution := &domain.Execution{
		ID:          trigID,
		LambdaID:    req.ReferenceID,
		ReferenceID: req.ReferenceID,
		Input:       req.Input,
		Status:      domain.ExecutionStatusPending,
		StartedAt:   time.Now(),
	}

	if s.executionRepo != nil {
		if err := s.executionRepo.Create(ctx, execution); err != nil {
			utils.Error(fmt.Sprintf("Failed to create execution record: %v", err))
		}
	}

	ack, err := s.SendTriggerWithID(ctx, trigID, req.ReferenceID, fmt.Sprintf("%v", req.Input))
	if err != nil {
		return nil, err
	}

	status := domain.ExecutionStatusPending
	if ack {
		status = domain.ExecutionStatusRunning
	}

	return &domain.LambdaExecResponse{
		ExecutionID: trigID,
		Status:      status,
		Message:     "Trigger sent",
	}, nil
}

// ActivateLambda implements domain.LambdaService
func (s *LambdaService) ActivateLambda(ctx context.Context, req *domain.LambdaExecRequest) (*domain.LambdaExecResponse, error) {
	if req.ReferenceID == "" {
		return nil, domain.ErrInvalidRequest
	}
	ack, err := s.Activate(ctx, req.ReferenceID)
	if err != nil {
		return nil, err
	}
	msg := "Activation requested"
	if ack {
		msg = "Activation successful"
	}
	return &domain.LambdaExecResponse{
		Status:  domain.ExecutionStatusRunning,
		Message: msg,
	}, nil
}

// DeactivateLambda implements domain.LambdaService
func (s *LambdaService) DeactivateLambda(ctx context.Context, req *domain.LambdaExecRequest) (*domain.LambdaExecResponse, error) {
	if req.ReferenceID == "" {
		return nil, domain.ErrInvalidRequest
	}

	ack, err := s.DeactivateJob(ctx, req.ReferenceID, "")
	if err != nil {
		return nil, err
	}
	msg := "Deactivation requested"
	if ack {
		msg = "Deactivation successful"
	}
	return &domain.LambdaExecResponse{
		Status:  domain.ExecutionStatusCompleted,
		Message: msg,
	}, nil
}

func (s *LambdaService) GetLambda(ctx context.Context, lambdaID string) (*domain.Lambda, error) {
	return s.gatewayDB.FindByID(ctx, lambdaID)
}

func (s *LambdaService) GetExecution(ctx context.Context, executionID string) (*domain.Execution, error) {
	if s.executionRepo != nil {
		return s.executionRepo.GetByID(ctx, executionID)
	}
	
	// Fallback if no repo configured
	return &domain.Execution{
		ID:        executionID,
		Status:    domain.ExecutionStatusPending,
		StartedAt: time.Now(),
		Input:     make(map[string]interface{}),
	}, nil
}

// func to store the lambda in database of func gateway(one with low TTL) anmd add a job to compilation queue
func (s *LambdaService) StoreandQueue(ctx context.Context, req *domain.LambdaStoreRequest) (bool, error) {
	utils.Info(fmt.Sprintf("[Gateway] Received StoreAndQueue request for FuncID: %s, UserID: %s", req.FuncID, req.UserID))

	if err := domain.ValidateStoreRequest(req); err != nil {
		utils.Error(fmt.Sprintf("[Gateway] Validation failed for FuncID: %s. Error: %v", req.FuncID, err))
		return false, err
	}

	lambda := &domain.Lambda{
		ID:         req.FuncID,
		UserID:     req.UserID,
		Name:       req.FuncID,
		SourceCode: req.SourceCode,
		Runtime:    req.Runtime,
		MemoryMB:   req.MemoryMB,
		RunType:    req.RunType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := s.gatewayDB.Save(ctx, lambda)
	if err != nil {
		errMsg := fmt.Sprintf("[Gateway] Error saving lambda to database: %v", err)
		utils.Error(errMsg)
		return false, domain.ErrInternalServer
	}
	utils.Info(fmt.Sprintf("[Gateway] Lambda %s saved to DB successfully", req.FuncID))

	if s.compilerQueue.JobsMap[req.FuncID] != nil {
		utils.Info(fmt.Sprintf("[Gateway] Job for %s already exists in queue, skipping enqueue", req.FuncID))
		return true, nil
	}

	err = s.compilerQueue.AddJob(&domain.CompilationQueueObject{
		FuncID: req.FuncID,
	})
	if err != nil {
		errMsg := fmt.Sprintf("[Gateway] Error adding job to compilation queue: %v", err)
		utils.Error(errMsg)
		return false, err
	}
	utils.Info(fmt.Sprintf("[Gateway] Job for %s added to CompilationQueue", req.FuncID))
	return true, nil
}

func (s *LambdaService) Activate(ctx context.Context, funcID string) (bool, error) {
	if funcID == "" {
		return false, domain.ErrExecutionFailed
	}
	// Check if orchestrator is set
	if s.orchestrator == nil {
		return false, errors.New("orchestrator not initialized")
	}
	ack, err := s.orchestrator.ActivateService(ctx, funcID)
	if err != nil {
		errMsg := fmt.Sprintf("Error activating the service: %v", err)
		utils.Error(errMsg)
		return false, err
	}
	return ack, nil
}

// func to send a deactivate service req to orchestrator
func (s *LambdaService) DeactivateJob(ctx context.Context, funcID string, userID string) (bool, error) {
	if funcID == "" {
		return false, domain.ErrExecutionFailed
	}
	ack, err := s.orchestrator.DeactivateService(ctx, funcID)
	if err != nil {
		errMsg := fmt.Sprintf("Error deactivating the service: %v", err)
		utils.Error(errMsg)
		return false, err
	}
	return ack, nil
}

// func to send a trigger for a service to orchestrator
func (s *LambdaService) SendTrigger(ctx context.Context, funcID string, input string) (bool, error) {
	if funcID == "" {
		return false, domain.ErrExecutionFailed
	}
	trigID := uuid.New().String()
	ack, err := s.orchestrator.ReceiveTrigger(ctx, trigID, funcID, input)
	if err != nil {
		errMsg := fmt.Sprintf("Error sending trigger to orchestrator: %v", err)
		utils.Error(errMsg)
		return false, err
	}
	return ack, nil
}

func (s *LambdaService) SendTriggerWithID(ctx context.Context, trigID string, funcID string, input string) (bool, error) {
	if funcID == "" {
		return false, domain.ErrExecutionFailed
	}
	ack, err := s.orchestrator.ReceiveTrigger(ctx, trigID, funcID, input)
	if err != nil {
		errMsg := fmt.Sprintf("Error sending trigger to orchestrator: %v", err)
		utils.Error(errMsg)
		return false, err
	}
	return ack, nil
}

func (s *LambdaService) GetStatus(ctx context.Context, funcID string) (string, error) {
	if funcID == "" {
		return "", domain.ErrExecutionFailed
	}
	exist, _ := s.compilerQueue.JobsMap[funcID]
	if exist != nil {
		return "In Queue", nil
	} else if status, err := s.compilerRepo.GetStatus(ctx, funcID); err == nil {
		return status, nil
	} else {
		return "still compiling", nil
	}
}

func (s *LambdaService) ActivateJob(ctx context.Context, funcID string, userID string) (bool, error) {
	// This method is called by Orchestrator during activation.
	// We can use this to set up any gateway-specific routing or logging.
	utils.Info(fmt.Sprintf("[Gateway] Service activated for FuncID: %s, UserID: %s", funcID, userID))
	return true, nil
}

func (s *LambdaService) ExecuteJob(ctx context.Context, funcID string, input string) (bool, error) {
	if funcID == "" {
		return false, domain.ErrInvalidRequest
	}

	trigID := uuid.New().String()
	ack, err := s.orchestrator.ReceiveTrigger(ctx, trigID, funcID, input)
	if err != nil {
		utils.Error(fmt.Sprintf("[Gateway] Error sending trigger for %s: %v", funcID, err))
		return false, err
	}

	return ack, nil
}

// ListLambdas returns all lambdas for a user
func (s *LambdaService) ListLambdas(ctx context.Context, userID string) ([]*domain.Lambda, error) {
	return s.gatewayDB.List(ctx, userID)
}

// ListExecutions returns all executions for a user
func (s *LambdaService) ListExecutions(ctx context.Context, userID string) ([]*domain.Execution, error) {
	// Query all executions for the user via their lambdas
	// For now, get all lambdas for the user first
	lambdas, err := s.gatewayDB.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(lambdas) == 0 {
		return []*domain.Execution{}, nil
	}

	// Collect all executions from all lambdas
	var allExecutions []*domain.Execution
	if s.executionRepo != nil {
		for _, lambda := range lambdas {
			executions, err := s.executionRepo.ListByLambda(ctx, lambda.ID)
			if err != nil {
				// Log error but continue
				continue
			}
			allExecutions = append(allExecutions, executions...)
		}
	}

	return allExecutions, nil
}

// DeleteLambda deletes a lambda function
func (s *LambdaService) DeleteLambda(ctx context.Context, lambdaID string) error {
	return s.gatewayDB.Delete(ctx, lambdaID)
}
