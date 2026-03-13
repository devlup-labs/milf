package core

import (
	"central_server/internal/sinkManager/domain"
	"central_server/internal/sinkManager/interfaces"
	"context"
	"encoding/base64"
	"fmt"
	"log"
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
	wasmFetcher    func(ctx context.Context, lambdaID string) ([]byte, error) // optional
	jwtSecret      []byte

	staleCtx      context.Context
	staleCancel   context.CancelFunc
	staleWg       sync.WaitGroup
	mu            sync.Mutex
	activeLambdas map[string]string
	wsConns       map[string]interface{} // SinkID -> *websocket.Conn
	wsWriteMu     sync.Mutex             // Protects concurrent writes to ALL websockets (simple fix)

	// Per-sink locks to ensure only one dispatch happens at a time for a worker
	sinkLocks   map[string]*sync.Mutex
	sinkLocksMu sync.Mutex

	// Track consecutive failures for testing automation
	sinkFailures   map[string]int
	sinkFailuresMu sync.Mutex

	// Testing hook: if set, will try to fetch and dispatch the most recent function
	// when a heartbeat is received and no real jobs are waiting.
	recentFuncFetcher func(ctx context.Context) (string, error)
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
		activeLambdas:  make(map[string]string),
		wsConns:        make(map[string]interface{}),
		sinkLocks:      make(map[string]*sync.Mutex),
		sinkFailures:   make(map[string]int),
	}
}

func (s *SinkManagerService) getSinkLock(sinkID string) *sync.Mutex {
	s.sinkLocksMu.Lock()
	defer s.sinkLocksMu.Unlock()
	if lock, ok := s.sinkLocks[sinkID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	s.sinkLocks[sinkID] = lock
	return lock
}

// SetWasmFetcher wires a function to fetch compiled WASM bytes by lambda ID.
// Call this after construction to break circular dependency.
func (s *SinkManagerService) SetWasmFetcher(fn func(ctx context.Context, lambdaID string) ([]byte, error)) {
	s.wasmFetcher = fn
}

// SetRecentFuncFetcher wires a function to fetch the most recent lambda ID for testing.
func (s *SinkManagerService) SetRecentFuncFetcher(fn func(ctx context.Context) (string, error)) {
	s.recentFuncFetcher = fn
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
	sink.LastHeartbeat = time.Now().UTC()

	// CRITICAL: Only mark as Online if it was previously Offline.
	// We MUST NOT overwrite 'Busy' status, or we will hammer the device with tasks.
	if sink.Status == domain.SinkStatusOffline || sink.Status == "" {
		sink.Status = domain.SinkStatusOnline
		log.Printf("[WorkManager] 🟢 Worker %s came ONLINE (RAM: %dMB)", req.SinkID, req.RAMAvailableMB)
	}

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
		log.Printf("[WorkManager] ⚠  Sink %s is OFFLINE, cannot deliver task %s", sink.ID, task.ExecutionID)
		return nil, domain.ErrSinkUnreachable
	}

	// Mark as busy IMMEDIATELY to prevent redundant dispatch from other goroutines
	// We'll roll this back if delivery fails completely.
	originalStatus := sink.Status
	sink.Status = domain.SinkStatusBusy
	_ = s.sinkRepo.Update(ctx, sink)

	task.Status = domain.TaskStatusPending
	task.CreatedAt = time.Now().UTC()

	if err := s.taskRepo.Save(ctx, task); err != nil {
		// Rollback busy status
		sink.Status = originalStatus
		_ = s.sinkRepo.Update(ctx, sink)
		return nil, domain.ErrInternalServer
	}

	// Try WebSocket delivery first (preferred — direct real-time push)
	_, wsConnected := s.wsConns[task.SinkID]
	if wsConnected {
		log.Printf("[WorkManager] ➤ Assigning job [exec=%s func=%s] to worker %s via WebSocket",
			task.ExecutionID, task.LambdaID, sink.ID)
		err := s.SendWebSocketMessage(task.SinkID, domain.MsgTaskAssignment, map[string]interface{}{
			"execution_id": task.ExecutionID,
			"lambda_id":    task.LambdaID,
			"wasm_url":     fmt.Sprintf("/api/v1/lambdas/%s/wasm", task.LambdaID),
			"wasm_base64":  task.WasmRef, // Inline binary to avoid separate download if possible
			"payload":      task.Input,
		})
		if err != nil {
			log.Printf("[WorkManager] ⚠  WS delivery failed for %s: %v — falling back to HTTP", task.ExecutionID, err)
			goto httpFallback
		}
		now := time.Now().UTC()
		task.Status = domain.TaskStatusDelivered
		task.DeliveredAt = &now
		_ = s.taskRepo.Update(ctx, task)
		
		s.mu.Lock()
		s.activeLambdas[task.LambdaID] = sink.ID
		s.mu.Unlock()
		return &domain.TaskDeliveryResponse{ExecutionID: task.ExecutionID, Accepted: true, Message: "Delivered via WebSocket"}, nil
	}

httpFallback:
	// HTTP delivery via sinkClient
	{
		deliveryReq := &domain.TaskDeliveryRequest{
			ExecutionID: task.ExecutionID,
			WasmRef:     task.WasmRef,
			Input:       task.Input,
		}

		resp, err := s.sinkClient.DeliverTask(ctx, sink, deliveryReq)
		if err != nil {
			log.Printf("[WorkManager] ✗ HTTP delivery failed for exec %s to sink %s: %v",
				task.ExecutionID, sink.ID, err)
			task.Status = domain.TaskStatusFailed
			_ = s.taskRepo.Update(ctx, task)
			
			// Rollback busy status
			sink.Status = originalStatus
			_ = s.sinkRepo.Update(ctx, sink)
			
			return nil, domain.ErrTaskDeliveryFailed
		}

		if resp.Accepted {
			log.Printf("[WorkManager] ➤ Job [exec=%s func=%s] accepted by worker %s (HTTP)",
				task.ExecutionID, task.LambdaID, sink.ID)
			now := time.Now().UTC()
			task.Status = domain.TaskStatusDelivered
			task.DeliveredAt = &now
			_ = s.taskRepo.Update(ctx, task)
		} else {
			log.Printf("[WorkManager] ✗ Sink %s rejected task %s", sink.ID, task.ExecutionID)
			task.Status = domain.TaskStatusFailed
			_ = s.taskRepo.Update(ctx, task)
			
			// Rollback busy status
			sink.Status = originalStatus
			_ = s.sinkRepo.Update(ctx, sink)
			
			return nil, domain.ErrTaskDeliveryFailed
		}
		return resp, nil
	}
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
		log.Printf("[WorkManager] ✔ Worker %s completed exec=%s output=%v",
			task.SinkID, req.ExecutionID, req.Output)
		
		// Reset failure count on success
		s.sinkFailuresMu.Lock()
		s.sinkFailures[task.SinkID] = 0
		s.sinkFailuresMu.Unlock()
	} else {
		task.Status = domain.TaskStatusFailed
		log.Printf("[WorkManager] ✗ Worker %s FAILED exec=%s error=%q",
			task.SinkID, req.ExecutionID, req.Error)
		
		// Increment failure count
		s.sinkFailuresMu.Lock()
		s.sinkFailures[task.SinkID]++
		s.sinkFailuresMu.Unlock()
	}
	task.CompletedAt = &now

	_ = s.taskRepo.Update(ctx, task)

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
			log.Printf("[WorkManager] 🟢 Worker %s back ONLINE, checking queue", task.SinkID)
			go s.tryDispatchToSink(context.Background(), sink)
		}
	}

	if s.resultCallback != nil {
		var resultErr error
		if !req.Success && req.Error != "" {
			resultErr = fmt.Errorf("%s", req.Error)
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
	// 1. Per-sink lock to avoid race conditions during dispatch
	lock := s.getSinkLock(sink.ID)
	lock.Lock()
	defer lock.Unlock()

	if sink.Status != domain.SinkStatusOnline {
		return
	}

	if s.queueService == nil {
		return
	}

	candidate, err := s.queueService.ClaimNextJob(sink.RAMAvailableMB)
	if err != nil || candidate == nil {
		// FALLBACK FOR TESTING: If the user requested auto-testing of the recent function
		if s.recentFuncFetcher != nil {
			log.Printf("[WorkManager] ⚡ Testing Mode: No jobs in queue, attempting to dispatch most recent function to sink %s", sink.ID)
			funcID, err := s.recentFuncFetcher(ctx)
			if err == nil && funcID != "" {
				s.dispatchMockJob(ctx, sink, funcID)
			}
		}
		return
	}

	log.Printf("[WorkManager] 📦 Claimed job [func=%s exec=%s] from queue for worker %s",
		candidate.Job.FuncID, candidate.Job.JobID, sink.ID)

	// Build input envelope
	inputPayload := candidate.Job.InputPayload
	if inputPayload == "" {
		inputPayload = `{"type":"null"}`
	}

	task := &domain.Task{
		ExecutionID: candidate.Job.JobID,
		LambdaID:    candidate.Job.FuncID,
		WasmRef:     "", // populated below if wasmFetcher is set
		Input: map[string]interface{}{
			"type": "json",
			"data": inputPayload,
		},
		SinkID:    sink.ID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now().UTC(),
	}

	// Try to inline WASM bytes as base64 if fetcher is wired
	if s.wasmFetcher != nil {
		if wasmBytes, err := s.wasmFetcher(ctx, candidate.Job.FuncID); err == nil && len(wasmBytes) > 0 {
			task.WasmRef = base64.StdEncoding.EncodeToString(wasmBytes)
			log.Printf("[WorkManager] 💾 Inlined %d WASM bytes for func=%s",
				len(wasmBytes), candidate.Job.FuncID)
		} else {
			log.Printf("[WorkManager] ⚠  Could not fetch WASM for func=%s: %v", candidate.Job.FuncID, err)
		}
	}

	_, err = s.DeliverTask(ctx, task)
	if err != nil {
		log.Printf("[WorkManager] ✗ Failed to deliver job [exec=%s] to worker %s: %v",
			task.ExecutionID, sink.ID, err)
	}
}

// dispatchMockJob sends a one-off mock job to a sink for testing.
func (s *SinkManagerService) dispatchMockJob(ctx context.Context, sink *domain.Sink, funcID string) {
	mockID := "test-" + uuid.New().String()[:8]
	task := &domain.Task{
		ExecutionID: mockID,
		LambdaID:    funcID,
		Input: map[string]interface{}{
			"type": "json",
			"data": `{"a": 15, "b": 35}`,
		},
		SinkID:    sink.ID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now().UTC(),
	}

	if s.wasmFetcher != nil {
		if wasmBytes, err := s.wasmFetcher(ctx, funcID); err == nil && len(wasmBytes) > 0 {
			task.WasmRef = base64.StdEncoding.EncodeToString(wasmBytes)
			log.Printf("[WorkManager] 💾 Inlined %d WASM bytes for test job %s", len(wasmBytes), mockID)
		}
	}

	_, err := s.DeliverTask(ctx, task)
	if err != nil {
		log.Printf("[WorkManager] ✗ Failed to deliver mock job to worker %s: %v", sink.ID, err)
	}
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

func (s *SinkManagerService) GetSinkForLambda(ctx context.Context, lambdaID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sinkID, ok := s.activeLambdas[lambdaID]
	return sinkID, ok
}

func (s *SinkManagerService) NotifyNewJob(ctx context.Context) {
	// Look for any online sink and try to dispatch
	sinks, err := s.sinkRepo.FindAll(ctx)
	if err != nil {
		return
	}
	for _, sink := range sinks {
		if sink.Status == domain.SinkStatusOnline {
			go s.tryDispatchToSink(context.Background(), sink)
		}
	}
}

// --- WebSocket Methods ---

func (s *SinkManagerService) RegisterWebSocket(sinkID string, conn interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wsConns[sinkID] = conn
}

func (s *SinkManagerService) UnregisterWebSocket(sinkID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.wsConns, sinkID)
}

func (s *SinkManagerService) SendWebSocketMessage(sinkID string, msgType domain.WsMessageType, payload interface{}) error {
	s.wsWriteMu.Lock()
	defer s.wsWriteMu.Unlock()

	s.mu.Lock()
	connInterface, exists := s.wsConns[sinkID]
	s.mu.Unlock()
	
	if !exists {
		return domain.ErrSinkUnreachable
	}
	
	// We use reflection/type assertion in the handler package that knows about gorilla/websocket
	// Here we just hold the interface or we can type-assert if we import it.
	// We'll define an interface locally to avoid tight coupling if preferred, 
	// but simplest is casting to an interface with WriteJSON.
	
	type jsonWriter interface {
		WriteJSON(v interface{}) error
	}
	
	if wsConn, ok := connInterface.(jsonWriter); ok {
		return wsConn.WriteJSON(domain.WebSocketMessage{
			Type:    msgType,
			Payload: payload,
		})
	}
	
	return domain.ErrInternalServer
}
