package core

import (
	"central_server/internal/queueService/domain"
	"central_server/internal/queueService/interfaces"
	sinkDomain "central_server/internal/sinkManager/domain"
	"central_server/utils"
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type QueueService struct {
	pool        *domain.QueuePool
	selector    QueueSelector
	jobIndex    map[string]string
	sinkManager sinkDomain.SinkManagerService
	mu          sync.Mutex
}

func NewQueueService() *QueueService {
	queues := []*domain.Queue{}
	q1, _ := domain.NewQueue("queue-1", 0, 1024)
	q2, _ := domain.NewQueue("queue-2", 1025, 4096)
	q3, _ := domain.NewQueue("queue-3", 4097, 8192)
	queues = append(queues, q1, q2, q3)
	pool, _ := domain.NewQueuePool(queues)
	return &QueueService{
		pool:     pool,
		jobIndex: make(map[string]string),
	}
}

func (s *QueueService) SetSinkManager(sm sinkDomain.SinkManagerService) {
	s.sinkManager = sm
}

func (s *QueueService) Enqueue(ctx context.Context, jobID string, funcID string, metaData map[string]string) (error, bool) {
	maxRam, err := strconv.Atoi(metaData["maxRam"])
	if err != nil {
		utils.Error("Failed to parse maxRam: " + err.Error())
		return err, false
	}

	job, err := domain.NewJob(jobID, funcID, metaData)
	if err != nil {
		utils.Error("Failed to create job: " + err.Error())
		return err, false
	}

	queue := s.selector.SelectQueue(*s.pool, maxRam)
	if queue == nil {
		utils.Error("No suitable queue found for jobID: " + jobID)
		return fmt.Errorf("No suitable queue found"), false
	}

	err = queue.AddJob(job)
	if err != nil {
		utils.Error("Failed to add job to queue: " + err.Error())
		return err, false
	}
	utils.Info("Job enqueued successfully: " + jobID + " in queue: " + queue.QueueID)
	err = job.UpdateStatus(domain.JobQueued)
	if err != nil {
		fmt.Println(err)
		return err, false
	}
	s.mu.Lock()
	s.jobIndex[jobID] = queue.QueueID
	s.mu.Unlock()
	return nil, true
}

func (s *QueueService) CandidateJobs(allowedRam int) []interfaces.CandidateJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	candidates := make([]interfaces.CandidateJob, 0)
	for _, queue := range s.pool.All() {
		if queue.ResourceRange.MinRam <= allowedRam &&
			queue.ResourceRange.MaxRam >= allowedRam {

			job, ok := queue.Peek()
			if ok {
				candidates = append(candidates, interfaces.CandidateJob{
					Job:     *job,
					QueueID: queue.QueueID,
				})
			}
		}
	}
	return candidates
}

func (s *QueueService) ClaimNextJob(allowedRam int) (*interfaces.CandidateJob, error) {
	candidates := s.CandidateJobs(allowedRam)
	if len(candidates) == 0 {
		return nil, nil
	}

	chosen := candidates[0]

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, queue := range s.pool.All() {
		if queue.QueueID == chosen.QueueID {
			queue.PopJob()
			delete(s.jobIndex, chosen.Job.JobID)
			break
		}
	}

	return &chosen, nil
}

func (s *QueueService) DispatchOrEnqueue(ctx context.Context, jobID string, funcID string, metaData map[string]string) (error, bool) {
	sinkID, active := s.sinkManager.GetSinkForLambda(ctx, funcID)
	if active {
		utils.Info(fmt.Sprintf("Lambda %s is active in sink %s. Dispatching directly.", funcID, sinkID))
		input := make(map[string]interface{})
		for k, v := range metaData {
			input[k] = v
		}

		task := &sinkDomain.Task{
			ExecutionID: jobID,
			LambdaID:    funcID,
			WasmRef:     metaData["wasmRef"],
			Input:       input,
			SinkID:      sinkID,
			Status:      sinkDomain.TaskStatusPending,
			CreatedAt:   time.Now().UTC(),
		}

		_, err := s.sinkManager.DeliverTask(ctx, task)
		if err != nil {
			utils.Error(fmt.Sprintf("Failed to deliver task to active sink %s: %v. Enqueuing instead.", sinkID, err))
			// Fallback to enqueue
			return s.Enqueue(ctx, jobID, funcID, metaData)
		}
		return nil, true
	}
	return s.Enqueue(ctx, jobID, funcID, metaData)
}
