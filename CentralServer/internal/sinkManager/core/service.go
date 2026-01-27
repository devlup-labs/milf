package core

import (
	"central_server/internal/sinkManager/domain"
	"central_server/internal/sinkManager/interfaces"
	"context"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SinkManagerService struct {
	sinkRepo       interfaces.SinkRepository
	taskRepo       interfaces.TaskRepository
	resultRepo     interfaces.TaskResultRepository
	sinkClient     interfaces.SinkClient
	queueService   interfaces.QueueService
	resultCallback domain.ResultCallback
	jwtSecret      []byte

	staleCtx    context.Context
	staleCancel context.CancelFunc
	staleWg     sync.WaitGroup
	mu          sync.Mutex
}

func NewSinkManagerService(
	sinkRepo interfaces.SinkRepository,
	taskRepo interfaces.TaskRepository,
	resultRepo interfaces.TaskResultRepository,
	sinkClient interfaces.SinkClient,
	queueService interfaces.QueueService,
	resultCallback domain.ResultCallback,
	jwtSecret string,
) *SinkManagerService {
	return &SinkManagerService{
		sinkRepo:       sinkRepo,
		taskRepo:       taskRepo,
		resultRepo:     resultRepo,
		sinkClient:     sinkClient,
		queueService:   queueService,
		resultCallback: resultCallback,
		jwtSecret:      []byte(jwtSecret),
	}
}

func (s *SinkManagerService) RegisterSink(ctx context.Context, req *domain.SinkRegisterRequest) (*domain.SinkRegisterResponse, error) {
	if err := domain.ValidateRegisterRequest(req); err != nil {
		return nil, err
	}

	existing, _ := s.sinkRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, domain.ErrSinkAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	now := time.Now().UTC()
	sink := &domain.Sink{
		ID:                 uuid.New().String(),
		Email:              req.Email,
		Password:           string(hashedPassword),
		Endpoint:           req.Endpoint,
		RAMAvailableMB:     0,
		StorageAvailableMB: 0,
		Status:             domain.SinkStatusOffline,
		LastHeartbeat:      now,
		RegisteredAt:       now,
	}

	if err := s.sinkRepo.Save(ctx, sink); err != nil {
		return nil, domain.ErrInternalServer
	}

	return &domain.SinkRegisterResponse{
		SinkID:  sink.ID,
		Message: "Sink registered successfully",
	}, nil
}

func (s *SinkManagerService) LoginSink(ctx context.Context, req *domain.SinkLoginRequest) (*domain.SinkLoginResponse, error) {
	if err := domain.ValidateLoginRequest(req); err != nil {
		return nil, err
	}

	sink, err := s.sinkRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(sink.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	claims := jwt.MapClaims{
		"sink_id": sink.ID,
		"email":   sink.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	return &domain.SinkLoginResponse{
		SinkID:  sink.ID,
		Token:   tokenString,
		Message: "Login successful",
	}, nil
}

func (s *SinkManagerService) UnregisterSink(ctx context.Context, sinkID string) error {
	_, err := s.sinkRepo.FindByID(ctx, sinkID)
	if err != nil {
		return domain.ErrSinkNotFound
	}

	if err := s.sinkRepo.Delete(ctx, sinkID); err != nil {
		return domain.ErrInternalServer
	}

	return nil
}

func (s *SinkManagerService) GetSink(ctx context.Context, sinkID string) (*domain.Sink, error) {
	sink, err := s.sinkRepo.FindByID(ctx, sinkID)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}
	return sink, nil
}

func (s *SinkManagerService) GetSinkByEmail(ctx context.Context, email string) (*domain.Sink, error) {
	sink, err := s.sinkRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}
	return sink, nil
}

func (s *SinkManagerService) ListSinks(ctx context.Context) ([]*domain.Sink, error) {
	sinks, err := s.sinkRepo.FindAll(ctx)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return sinks, nil
}

func (s *SinkManagerService) ProcessHeartbeat(ctx context.Context, req *domain.HeartbeatRequest) (*domain.HeartbeatResponse, error) {
	if err := domain.ValidateHeartbeatRequest(req); err != nil {
		return nil, err
	}

	sink, err := s.sinkRepo.FindByID(ctx, req.SinkID)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}

	sink.RAMAvailableMB = req.RAMAvailableMB
	sink.StorageAvailableMB = req.StorageAvailableMB
	sink.Status = domain.SinkStatusOnline
	sink.LastHeartbeat = time.Now().UTC()

	if err := s.sinkRepo.Update(ctx, sink); err != nil {
		return nil, domain.ErrInternalServer
	}

	go s.tryDispatchToSink(context.Background(), sink)

	return &domain.HeartbeatResponse{
		Acknowledged: true,
		Message:      "Heartbeat acknowledged",
	}, nil
}

func (s *SinkManagerService) DeliverTask(ctx context.Context, task *domain.Task) (*domain.TaskDeliveryResponse, error) {
	if task.SinkID == "" {
		return nil, domain.ErrInvalidSinkRequest
	}

	sink, err := s.sinkRepo.FindByID(ctx, task.SinkID)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}

	if sink.Status == domain.SinkStatusOffline {
		return nil, domain.ErrSinkUnreachable
	}

	task.Status = domain.TaskStatusPending
	task.CreatedAt = time.Now().UTC()

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, domain.ErrInternalServer
	}

	deliveryReq := &domain.TaskDeliveryRequest{
		ExecutionID: task.ExecutionID,
		WasmRef:     task.WasmRef,
		Input:       task.Input,
	}

	resp, err := s.sinkClient.DeliverTask(ctx, sink, deliveryReq)
	if err != nil {
		task.Status = domain.TaskStatusFailed
		_ = s.taskRepo.Update(ctx, task)
		return nil, domain.ErrTaskDeliveryFailed
	}

	if resp.Accepted {
		now := time.Now().UTC()
		task.Status = domain.TaskStatusDelivered
		task.DeliveredAt = &now
		_ = s.taskRepo.Update(ctx, task)

		sink.Status = domain.SinkStatusBusy
		_ = s.sinkRepo.Update(ctx, sink)
	} else {
		task.Status = domain.TaskStatusFailed
		_ = s.taskRepo.Update(ctx, task)
		return nil, domain.ErrTaskDeliveryFailed
	}

	return resp, nil
}

func (s *SinkManagerService) ProcessTaskResult(ctx context.Context, req *domain.TaskResultRequest) (*domain.TaskResultResponse, error) {
	if err := domain.ValidateTaskResultRequest(req); err != nil {
		return nil, err
	}

	task, err := s.taskRepo.FindByExecutionID(ctx, req.ExecutionID)
	if err != nil {
		return nil, domain.ErrResultNotFound
	}

	now := time.Now().UTC()
	if req.Success {
		task.Status = domain.TaskStatusCompleted
	} else {
		task.Status = domain.TaskStatusFailed
	}
	task.CompletedAt = &now

	_ = s.taskRepo.Update(ctx, task)

	// Save result
	result := &domain.TaskResult{
		ExecutionID: req.ExecutionID,
		Output:      req.Output,
		Error:       req.Error,
		Success:     req.Success,
		ReceivedAt:  now,
	}

	if err := s.resultRepo.Save(ctx, result); err != nil {
		return nil, domain.ErrInternalServer
	}

	if task.SinkID != "" {
		if sink, err := s.sinkRepo.FindByID(ctx, task.SinkID); err == nil {
			sink.Status = domain.SinkStatusOnline
			_ = s.sinkRepo.Update(ctx, sink)
			go s.tryDispatchToSink(context.Background(), sink)
		}
	}

	if s.resultCallback != nil {
		var resultErr error
		if !req.Success && req.Error != "" {
			resultErr = domain.ErrResultNotFound
		}
		go s.resultCallback(context.Background(), req.ExecutionID, req.Output, resultErr)
	}

	return &domain.TaskResultResponse{
		Received: true,
		Message:  "Result received successfully",
	}, nil
}

func (s *SinkManagerService) GetTaskResult(ctx context.Context, executionID string) (*domain.TaskResult, error) {
	result, err := s.resultRepo.FindByExecutionID(ctx, executionID)
	if err != nil {
		return nil, domain.ErrResultNotFound
	}
	return result, nil
}

func (s *SinkManagerService) tryDispatchToSink(ctx context.Context, sink *domain.Sink) {
	if sink.Status != domain.SinkStatusOnline {
		return
	}

	if s.queueService == nil {
		return
	}

	candidate, err := s.queueService.ClaimNextJob(sink.RAMAvailableMB)
	if err != nil || candidate == nil {
		return
	}

	input := make(map[string]interface{})
	for k, v := range candidate.Job.MetaData {
		input[k] = v
	}

	task := &domain.Task{
		ExecutionID: candidate.Job.JobID,
		LambdaID:    candidate.Job.FuncID,
		WasmRef:     candidate.Job.MetaData["wasmRef"],
		Input:       input,
		SinkID:      sink.ID,
		Status:      domain.TaskStatusPending,
		CreatedAt:   time.Now().UTC(),
	}

	_, _ = s.DeliverTask(ctx, task)
}

func (s *SinkManagerService) StartStaleDetector(ctx context.Context, staleThreshold time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.staleCancel != nil {
		return
	}

	s.staleCtx, s.staleCancel = context.WithCancel(ctx)

	s.staleWg.Add(1)
	go s.staleDetectorLoop(staleThreshold)
}

func (s *SinkManagerService) StopStaleDetector() {
	s.mu.Lock()
	if s.staleCancel != nil {
		s.staleCancel()
		s.staleCancel = nil
	}
	s.mu.Unlock()

	s.staleWg.Wait()
}

func (s *SinkManagerService) staleDetectorLoop(staleThreshold time.Duration) {
	defer s.staleWg.Done()

	ticker := time.NewTicker(staleThreshold / 2)
	defer ticker.Stop()

	for {
		select {
		case <-s.staleCtx.Done():
			return
		case <-ticker.C:
			s.markStaleSinksOffline(staleThreshold)
		}
	}
}

func (s *SinkManagerService) markStaleSinksOffline(staleThreshold time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sinks, err := s.sinkRepo.FindAll(ctx)
	if err != nil {
		return
	}

	now := time.Now().UTC()
	for _, sink := range sinks {
		if sink.Status == domain.SinkStatusOffline {
			continue
		}

		if now.Sub(sink.LastHeartbeat) > staleThreshold {
			sink.Status = domain.SinkStatusOffline
			_ = s.sinkRepo.Update(ctx, sink)
		}
	}
}
