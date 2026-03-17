package core

import (
	"central_server/internal/gateway/domain"
	"central_server/internal/gateway/interfaces"
	"central_server/utils"
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	gatewayDB    interfaces.FuncGatewayDB
	lambdaExec   interfaces.OrchestratorService
	jobs         map[string]cron.EntryID
	mu           sync.Mutex
}

func NewScheduler(db interfaces.FuncGatewayDB, exec interfaces.OrchestratorService) *Scheduler {
	return &Scheduler{
		cron:       cron.New(),
		gatewayDB:  db,
		lambdaExec: exec,
		jobs:       make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.cron.Start()
	utils.Info("[Scheduler] CRON scheduler started")
	
	// Initial sync of periodic jobs
	s.SyncJobs(ctx)
}

func (s *Scheduler) SyncJobs(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// List all lambdas (pass empty string for all users)
	lambdas, err := s.gatewayDB.List(ctx, "")
	if err != nil {
		utils.Error(fmt.Sprintf("[Scheduler] Failed to list lambdas for sync: %v", err))
		return
	}
	utils.Info(fmt.Sprintf("[Scheduler] Found %d total lambdas in DB", len(lambdas)))

	for _, lambda := range lambdas {
		utils.Info(fmt.Sprintf("[Scheduler] Sync checking: %s (type=%s, cron=%s)", lambda.ID, lambda.RunType, lambda.CronExpression))
		if lambda.RunType == domain.RunTypePeriodic && lambda.CronExpression != "" {
			s.addOrUpdateJob(lambda)
		} else {
			s.removeJob(lambda.ID)
		}
	}
}

func (s *Scheduler) addOrUpdateJob(lambda *domain.Lambda) {
	// If already exists, check if cron changed (simplified: always re-register for now or check map)
	if _, exists := s.jobs[lambda.ID]; exists {
		s.cron.Remove(s.jobs[lambda.ID])
	}

	id, err := s.cron.AddFunc(lambda.CronExpression, func() {
		utils.Info(fmt.Sprintf("[Scheduler] Triggering periodic job: %s", lambda.ID))
		// Use a background context for the scheduled execution
		ctx := context.Background()
		// Manual input for now, could be configurable
		input := `{"source": "scheduler", "time": "periodic"}`
		
		// Need a unique trigger ID for every execution
		trigID := uuid.New().String()
		// Actually uses generic receive trigger
		_, err := s.lambdaExec.ReceiveTrigger(ctx, trigID, lambda.ID, input)
		if err != nil {
			utils.Error(fmt.Sprintf("[Scheduler] Failed to trigger job %s: %v", lambda.ID, err))
		}
	})

	if err != nil {
		utils.Error(fmt.Sprintf("[Scheduler] Failed to add cron for job %s: %v", lambda.ID, err))
		return
	}

	s.jobs[lambda.ID] = id
	utils.Info(fmt.Sprintf("[Scheduler] Registered job %s with cron: %s", lambda.ID, lambda.CronExpression))
}

func (s *Scheduler) removeJob(lambdaID string) {
	if id, exists := s.jobs[lambdaID]; exists {
		s.cron.Remove(id)
		delete(s.jobs, lambdaID)
		utils.Info(fmt.Sprintf("[Scheduler] Removed job %s from scheduler", lambdaID))
	}
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}
