package types

import (
	"encoding/json"
	"time"
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
	Type string	`json:"type" binding:"required"`
	Payload json.RawMessage `json:"payload" binding:"required"`
	MaxRetries int `json:"max_retries,omitempty"`
}

// Job Response Struct
type JobResponse struct {
	JobID string `json:"job_id"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Enum to represent the stage of the job
type JobStatus string

const (
	StatusPending JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted JobStatus = "completed"
	StatusFailed JobStatus = "failed"
	StatusRetrying JobStatus = "retrying"
)