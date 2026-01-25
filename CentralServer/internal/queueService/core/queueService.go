package core

import (
	"central_server/internal/queueService/domain"
	"central_server/utils"
	"context"
	"fmt"
	"strconv"
	"sync"
)

type QueueService struct {
	pool     *domain.QueuePool
	selector QueueSelector
	jobIndex map[string]string//this is made so that later when the sinkManager wants to add a job, we will send it a jobs from all the queues and then it will claim one of the job and then we will remove the job from the queue
	mu       sync.Mutex // so that if multiple workers in the orchastrator later that appends to thw queues, race conditittions are solved
}

type CandidateJob struct{
	job domain.Job
	queueID string
}

func NewQueueService() *QueueService {
		queues := []*domain.Queue{}
		q1, _ := domain.NewQueue("queue-1", 0, 1024)
		q2, _ := domain.NewQueue("queue-2", 1025, 4096)
		q3, _ := domain.NewQueue("queue-3", 4097, 8192)
		queues = append(queues, q1, q2, q3)
		pool, _ := domain.NewQueuePool(queues)
		return &QueueService{
			pool: pool,
			jobIndex: make(map[string]string),
		}
}

func (s *QueueService)Enqueue(ctx context.Context, jobID string, funcID string, metaData map[string]string) (error, bool){
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


func (s *QueueService) CandidateJobs(allowedRam int) []CandidateJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	candidates := make([]CandidateJob, 0)
	for _, queue := range s.pool.All() {
		// queue-level resource compatibility
		if queue.ResourceRange.MinRam <= allowedRam &&
			queue.ResourceRange.MaxRam >= allowedRam {

			job, ok := queue.Peek()
			if ok {
				candidates = append(candidates, CandidateJob{
					job:     *job,
					queueID: queue.QueueID,
				})
			}
		}
	}
	return candidates
}

func (s *QueueService)ClaimJob(queueID string){
	s.mu.Lock()
	defer s.mu.Unlock()
	queues := s.pool.All()
	queue := queues[queueID]//correct this later
	queue.PopJob()
}