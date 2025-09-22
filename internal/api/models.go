package api

import (
	"time"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
)

// API Request/Response Types

// EnqueueJobRequest represents a request to add a job to the queue
type EnqueueJobRequest struct {
	Type       string           `json:"type" binding:"required"`
	Payload    interface{}      `json:"payload" binding:"required"`
	MaxRetries *int             `json:"max_retries,omitempty"`
	Priority   string           `json:"priority,omitempty"` // high, normal, low
	ExecuteAt  *time.Time       `json:"execute_at,omitempty"`
	Recurring  *RecurringConfig `json:"recurring,omitempty"`
}

// RecurringConfig holds configuration for a recurring job
type RecurringConfig struct {
	CronExpression string `json:"cron_expression" binding:"required"`
}

// EnqueueJobResponse represents the response after enqueuing a job
type EnqueueJobResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// JobStatusRequest represents a request to get job status
type JobStatusRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

// JobStatusResponse represents the response with job status information
type JobStatusResponse struct {
	JobID       string          `json:"job_id"`
	Type        string          `json:"type"`
	Status      types.JobStatus `json:"status"`
	EnqueuedAt  time.Time       `json:"enqueued_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Attempts    int             `json:"attempts"`
	MaxRetries  int             `json:"max_retries"`
	Error       string          `json:"error,omitempty"`
}

// BatchEnqueueRequest represents a request to enqueue multiple jobs at once
type BatchEnqueueRequest struct {
	Jobs []EnqueueJobRequest `json:"jobs" binding:"required,min=1,dive"`
}

// BatchEnqueueResponse represents the response after enqueuing multiple jobs
type BatchEnqueueResponse struct {
	Jobs []EnqueueJobResponse `json:"jobs"`
}

// CancelJobRequest represents a request to cancel a job
type CancelJobRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

// CancelJobResponse represents the response after canceling a job
type CancelJobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// QueueStatsResponse represents the response with queue statistics
type QueueStatsResponse struct {
	Queues map[string]QueueInfo `json:"queues"`
	Jobs   JobStats             `json:"jobs"`
}

// QueueInfo holds information about a specific queue
type QueueInfo struct {
	Size          int `json:"size"`
	TotalEnqueued int `json:"total_enqueued"`
	TotalDequeued int `json:"total_dequeued"`
}

// JobStats holds statistics about jobs
type JobStats struct {
	TotalProcessed int            `json:"total_processed"`
	TotalFailed    int            `json:"total_failed"`
	TotalRetried   int            `json:"total_retried"`
	ByType         map[string]int `json:"by_type"`
}

// WorkerStatsResponse represents the response with worker statistics
type WorkerStatsResponse struct {
	TotalWorkers  int          `json:"total_workers"`
	ActiveWorkers int          `json:"active_workers"`
	IdleWorkers   int          `json:"idle_workers"`
	WorkersInfo   []WorkerInfo `json:"workers"`
}

// WorkerInfo holds information about a specific worker
type WorkerInfo struct {
	ID            string `json:"id"`
	Status        string `json:"status"` // idle, processing
	JobsProcessed int    `json:"jobs_processed"`
	JobsFailed    int    `json:"jobs_failed"`
	JobsRetried   int    `json:"jobs_retried"`
	CurrentJobID  string `json:"current_job_id,omitempty"`
}

// ListFailedJobsResponse represents the response with failed jobs
type ListFailedJobsResponse struct {
	Jobs       []FailedJobInfo `json:"jobs"`
	TotalCount int             `json:"total_count"`
}

// FailedJobInfo holds information about a failed job
type FailedJobInfo struct {
	JobID      string    `json:"job_id"`
	Type       string    `json:"type"`
	Payload    string    `json:"payload"`
	Error      string    `json:"error"`
	Attempts   int       `json:"attempts"`
	MaxRetries int       `json:"max_retries"`
	FailedAt   time.Time `json:"failed_at"`
}

// RetryFailedJobRequest represents a request to retry a failed job
type RetryFailedJobRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

// RetryFailedJobResponse represents the response after retrying a failed job
type RetryFailedJobResponse struct {
	JobID      string    `json:"job_id"`
	Status     string    `json:"status"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

// RetryAllFailedJobsResponse represents the response after retrying all failed jobs
type RetryAllFailedJobsResponse struct {
	Count  int      `json:"count"`
	JobIDs []string `json:"job_ids"`
	Status string   `json:"status"`
}

// HealthResponse represents the response for health check
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Redis     string `json:"redis"`
	Workers   string `json:"workers"`
}
