package domain

import (
	"container/list"
	"errors"
)

type ResourceRange struct {
	MinRam int
	MaxRam int
}


type Queue struct {
	QueueID       string
	ResourceRange ResourceRange

	jobs     *list.List                     
	existMap map[string]*list.Element       
	numJobs  int
}

func NewQueue(queueID string, minRam, maxRam int) (*Queue, error) {
	if queueID == "" {
		return nil, errors.New("queueID cannot be empty")
	}
	if minRam < 0 || maxRam < 0 {
		return nil, errors.New("RAM values cannot be negative")
	}
	if minRam > maxRam {
		return nil, errors.New("minRam cannot exceed maxRam")
	}
	if minRam == 0 && maxRam == 0 {
		return nil, errors.New("at least one of minRam or maxRam must be > 0")
	}

	return &Queue{
		QueueID: queueID,
		ResourceRange: ResourceRange{
			MinRam: minRam,
			MaxRam: maxRam,
		},
		jobs:     list.New(),
		existMap: make(map[string]*list.Element),
		numJobs:  0,
	}, nil
}


func (q *Queue) AddJob(job *Job) error {
	if _, exists := q.existMap[job.JobID]; exists {
		return errors.New("job already exists in queue")
	}

	elem := q.jobs.PushBack(job)
	q.existMap[job.JobID] = elem
	q.numJobs++

	return nil
}

func (q *Queue) PopJob() (*Job, error) {
	front := q.jobs.Front()
	if front == nil {
		return nil, errors.New("queue is empty")
	}

	job := front.Value.(*Job)

	q.jobs.Remove(front)
	delete(q.existMap, job.JobID)
	q.numJobs--

	return job, nil
}

func (q *Queue) RemoveJob(jobID string) error {
	elem, exists := q.existMap[jobID]
	if !exists {
		return errors.New("job not found in queue")
	}

	q.jobs.Remove(elem)
	delete(q.existMap, jobID)
	q.numJobs--

	return nil
}

func (q *Queue) HasJob(jobID string) bool {
	_, exists := q.existMap[jobID]
	return exists
}

func (q *Queue) Peek() (*Job, bool) {
	front := q.jobs.Front()
	if front == nil {
		return nil, false
	}
	return front.Value.(*Job), true
}

func (q *Queue) Size() int {
	return q.numJobs
}

func (q *Queue) IsEmpty() bool {
	return q.numJobs == 0
}
