package domain

import "errors"

type JobStatus string

const (
	JobNew     JobStatus = "NEW"
	JobQueued  JobStatus = "QUEUED"
	JobPending JobStatus = "PENDING"
	JobRunning JobStatus = "RUNNING"
	JobDone    JobStatus = "DONE"
	JobFailed  JobStatus = "FAILED"
)

type Job struct {
	JobID        string
	FuncID       string
	Status       JobStatus
	MetaData   	 map[string]string `json:"metadata"`
	CyclesWaited int
}


func NewJob(jobID, funcID string, metaData map[string]string) (*Job, error) {
	if jobID == "" {
		return nil, errors.New("jobID cannot be empty")
	}
	if funcID == "" {
		return nil, errors.New("funcID cannot be empty")
	}

	return &Job{
		JobID:  jobID,
		FuncID: funcID,
		Status: JobNew,
		MetaData: metaData,
		CyclesWaited: 0,
	}, nil
}


func (j *Job) UpdateStatus(newStatus JobStatus) error {
	if !isValidTransition(j.Status, newStatus) {
		return errors.New("invalid job status transition")
	}
	j.Status = newStatus
	return nil
}

func (j *Job) IncrementWait() {
	j.CyclesWaited++
}

func (j *Job) ResetWait() {
	j.CyclesWaited = 0
}


func isValidTransition(from, to JobStatus) bool {
	switch from {
	case JobNew:
		return to == JobQueued
	case JobQueued:
		return to == JobPending || to == JobFailed
	case JobPending:
		return to == JobRunning || to == JobQueued
	case JobRunning:
		return to == JobDone || to == JobFailed
	default:
		return false
	}
}
