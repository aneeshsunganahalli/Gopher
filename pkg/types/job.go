package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Queued Job Struct
type Job struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Attempts   int             `json:"attempts"`
	MaxRetries int             `json:"max_retries"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Job Submission Request
type JobRequest struct {
	Type       string          `json:"type" binding:"required"`
	Payload    json.RawMessage `json:"payload" binding:"required"`
	MaxRetries *int             `json:"max_retries,omitempty"`
}

// Job Response Struct
type JobResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Enum to represent the stage of the job
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
)

type JobHandler interface {

	// Handle processes the job with the given context
	Handle(ctx context.Context, job *Job) error

	// Type returns the job type this handler processes
	Type() string

	// Description returns a human-readable description of what this handler does
	Description() string
}

type JobResult struct {
	JobID       string    `json:"job_id"`
	Status      JobStatus `json:"status"`
	Error       string    `json:"error,omitempty"`
	Duration    string    `json:"duration"`
	CompletedAt time.Time `json:"completed_at"`
}

func NewJob(jobType string, payload json.RawMessage, maxRetries int) *Job {
	return &Job{
		ID:         generateJobID(),
		Type:       jobType,
		Payload:    payload,
		Attempts:   0,
		MaxRetries: maxRetries,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
}

func (j *Job) ShouldRetry() bool {
	return j.Attempts < j.MaxRetries
}

func (j *Job) IncrementAttempts() {
	j.Attempts++
	j.UpdatedAt = time.Now().UTC()
}

func (j *Job) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if j.Type == "" {
		return fmt.Errorf("job type cannot be empty")
	}
	if len(j.Payload) == 0 {
		return fmt.Errorf("job payload cannot be empty")
	}
	if j.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be empty")
	}
	return nil
}

func generateJobID() string {
	id := uuid.NewString()
	return "job_" + id
}
