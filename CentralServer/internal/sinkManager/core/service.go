package core

import (
	"context"
	"sync"
	"time"

	"central_server/internal/sinkManager/domain"
	"central_server/internal/sinkManager/interfaces"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SinkManagerService struct {
	sinkRepo       interfaces.SinkRepository
	taskRepo       interfaces.TaskRepository
	resultRepo     interfaces.TaskResultRepository
	sinkClient     interfaces.SinkClient
	resultCallback domain.ResultCallback
	jwtSecret      []byte

	// Heartbeat monitoring
	heartbeatCtx    context.Context
	heartbeatCancel context.CancelFunc
	heartbeatWg     sync.WaitGroup
	mu              sync.Mutex
}

func NewSinkManagerService(
	sinkRepo interfaces.SinkRepository,
	taskRepo interfaces.TaskRepository,
	resultRepo interfaces.TaskResultRepository,
	sinkClient interfaces.SinkClient,
	resultCallback domain.ResultCallback,
	jwtSecret string,
) *SinkManagerService {
	return &SinkManagerService{
		sinkRepo:       sinkRepo,
		taskRepo:       taskRepo,
		resultRepo:     resultRepo,
		sinkClient:     sinkClient,
		resultCallback: resultCallback,
		jwtSecret:      []byte(jwtSecret),
	}
}

// RegisterSink registers a new sink with email/password
func (s *SinkManagerService) RegisterSink(ctx context.Context, req *domain.SinkRegisterRequest) (*domain.SinkRegisterResponse, error) {
	if err := domain.ValidateRegisterRequest(req); err != nil {
		return nil, err
	}

	// Check if sink with same email already exists
	existing, _ := s.sinkRepo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, domain.ErrSinkAlreadyExists
	}

	// Hash password
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
		Status:             domain.SinkStatusOffline, // Offline until first heartbeat
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

// LoginSink authenticates a sink and returns a token
func (s *SinkManagerService) LoginSink(ctx context.Context, req *domain.SinkLoginRequest) (*domain.SinkLoginResponse, error) {
	if err := domain.ValidateLoginRequest(req); err != nil {
		return nil, err
	}

	sink, err := s.sinkRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(sink.Password), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Generate JWT token
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

// UnregisterSink removes a sink from the registry
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

// GetSink retrieves a sink by ID
func (s *SinkManagerService) GetSink(ctx context.Context, sinkID string) (*domain.Sink, error) {
	sink, err := s.sinkRepo.FindByID(ctx, sinkID)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}
	return sink, nil
}

// GetSinkByEmail retrieves a sink by email
func (s *SinkManagerService) GetSinkByEmail(ctx context.Context, email string) (*domain.Sink, error) {
	sink, err := s.sinkRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}
	return sink, nil
}

// ListSinks returns all registered sinks
func (s *SinkManagerService) ListSinks(ctx context.Context) ([]*domain.Sink, error) {
	sinks, err := s.sinkRepo.FindAll(ctx)
	if err != nil {
		return nil, domain.ErrInternalServer
	}
	return sinks, nil
}

// ProcessHeartbeat handles heartbeat from a sink (called every 10 seconds by the sink)
func (s *SinkManagerService) ProcessHeartbeat(ctx context.Context, req *domain.HeartbeatRequest) (*domain.HeartbeatResponse, error) {
	if err := domain.ValidateHeartbeatRequest(req); err != nil {
		return nil, err
	}

	sink, err := s.sinkRepo.FindByID(ctx, req.SinkID)
	if err != nil {
		return nil, domain.ErrSinkNotFound
	}

	// Update sink status and resources
	sink.RAMAvailableMB = req.RAMAvailableMB
	sink.StorageAvailableMB = req.StorageAvailableMB
	sink.Status = domain.SinkStatusOnline
	sink.LastHeartbeat = time.Now().UTC()

	if err := s.sinkRepo.Update(ctx, sink); err != nil {
		return nil, domain.ErrInternalServer
	}

	return &domain.HeartbeatResponse{
		Acknowledged: true,
		Message:      "Heartbeat acknowledged",
	}, nil
}

// DeliverTask delivers a task to a specific sink
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

	// Save task before delivery
	task.Status = domain.TaskStatusPending
	task.CreatedAt = time.Now().UTC()
	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	if err := s.taskRepo.Save(ctx, task); err != nil {
		return nil, domain.ErrInternalServer
	}

	// Prepare delivery request
	deliveryReq := &domain.TaskDeliveryRequest{
		TaskID:      task.ID,
		ExecutionID: task.ExecutionID,
		WasmRef:     task.WasmRef,
		Input:       task.Input,
	}

	// Send task to sink
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

		// Mark sink as busy
		sink.Status = domain.SinkStatusBusy
		_ = s.sinkRepo.Update(ctx, sink)
	} else {
		task.Status = domain.TaskStatusFailed
		_ = s.taskRepo.Update(ctx, task)
		return nil, domain.ErrTaskDeliveryFailed
	}

	return resp, nil
}

// ProcessTaskResult handles the result from a sink
func (s *SinkManagerService) ProcessTaskResult(ctx context.Context, req *domain.TaskResultRequest) (*domain.TaskResultResponse, error) {
	if err := domain.ValidateTaskResultRequest(req); err != nil {
		return nil, err
	}

	// Find the task
	task, err := s.taskRepo.FindByID(ctx, req.TaskID)
	if err != nil {
		return nil, domain.ErrResultNotFound
	}

	// Update task status
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
		ID:          uuid.New().String(),
		TaskID:      req.TaskID,
		ExecutionID: req.ExecutionID,
		Output:      req.Output,
		Error:       req.Error,
		Success:     req.Success,
		ReceivedAt:  now,
	}

	if err := s.resultRepo.Save(ctx, result); err != nil {
		return nil, domain.ErrInternalServer
	}

	// Mark sink as online again
	if task.SinkID != "" {
		if sink, err := s.sinkRepo.FindByID(ctx, task.SinkID); err == nil {
			sink.Status = domain.SinkStatusOnline
			_ = s.sinkRepo.Update(ctx, sink)
		}
	}

	// Notify gateway via callback
	if s.resultCallback != nil {
		var resultErr error
		if !req.Success && req.Error != "" {
			resultErr = domain.ErrResultNotFound // Use as placeholder, real error message is in req.Error
		}
		go s.resultCallback(context.Background(), req.ExecutionID, req.Output, resultErr)
	}

	return &domain.TaskResultResponse{
		Received: true,
		Message:  "Result received successfully",
	}, nil
}

// GetTaskResult retrieves a task result by task ID
func (s *SinkManagerService) GetTaskResult(ctx context.Context, taskID string) (*domain.TaskResult, error) {
	result, err := s.resultRepo.FindByTaskID(ctx, taskID)
	if err != nil {
		return nil, domain.ErrResultNotFound
	}
	return result, nil
}

// StartHeartbeatMonitor starts the background heartbeat monitoring
func (s *SinkManagerService) StartHeartbeatMonitor(ctx context.Context, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.heartbeatCancel != nil {
		return // Already running
	}

	s.heartbeatCtx, s.heartbeatCancel = context.WithCancel(ctx)

	s.heartbeatWg.Add(1)
	go s.heartbeatLoop(interval)
}

// StopHeartbeatMonitor stops the background heartbeat monitoring
func (s *SinkManagerService) StopHeartbeatMonitor() {
	s.mu.Lock()
	if s.heartbeatCancel != nil {
		s.heartbeatCancel()
		s.heartbeatCancel = nil
	}
	s.mu.Unlock()

	s.heartbeatWg.Wait()
}

func (s *SinkManagerService) heartbeatLoop(interval time.Duration) {
	defer s.heartbeatWg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.heartbeatCtx.Done():
			return
		case <-ticker.C:
			s.checkAllSinks()
		}
	}
}

func (s *SinkManagerService) checkAllSinks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sinks, err := s.sinkRepo.FindAll(ctx)
	if err != nil {
		return
	}

	for _, sink := range sinks {
		if sink.Status == domain.SinkStatusOffline {
			continue // Skip offline sinks
		}

		go s.checkSink(sink)
	}
}

func (s *SinkManagerService) checkSink(sink *domain.Sink) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.sinkClient.SendHeartbeat(ctx, sink)
	if err != nil {
		sink.Status = domain.SinkStatusOffline
		_ = s.sinkRepo.Update(ctx, sink)
		return
	}

	// Update sink with latest resource info
	sink.RAMAvailableMB = resp.RAMAvailableMB
	sink.StorageAvailableMB = resp.StorageAvailableMB
	sink.LastHeartbeat = time.Now().UTC()

	// Only set to online if not busy
	if sink.Status != domain.SinkStatusBusy {
		sink.Status = domain.SinkStatusOnline
	}

	_ = s.sinkRepo.Update(ctx, sink)
}
