package worker

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/aneeshsunganahalli/Gopher/internal/job"
	"github.com/aneeshsunganahalli/Gopher/internal/queue"
	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"go.uber.org/zap"
)

type contextKey string

const startTimeKey contextKey = "start_time"

type Worker struct {
	config   WorkerConfig
	queue    queue.Queue
	registry *job.Registry
	logger   *zap.Logger

	jobsProcessed int64
	jobsFailed    int64
	jobsRetried   int64
	isActive      int32 // 0 = inactive, 1 = active

	// Current job context (for cancellation)
	currentJobCtx    context.Context
	currentJobCancel context.CancelFunc
}

// WorkerConfig holds configuration for a worker
type WorkerConfig struct {
	ID           string
	PollInterval time.Duration
}

// WorkerStats holds statistics for a single worker
type WorkerStats struct {
	WorkerID       string `json:"worker_id"`
	JobsProcessed  int64  `json:"jobs_processed"`
	JobsFailed     int64  `json:"jobs_failed"`
	JobsRetried    int64  `json:"jobs_retried"`
	IsActive       bool   `json:"is_active"`
}

func NewWorker(config WorkerConfig, queue queue.Queue, registry *job.Registry, logger *zap.Logger) *Worker {
	return &Worker{
		config:   config,
		queue:    queue,
		registry: registry,
		logger:   logger.With(zap.String("worker_id", config.ID)),
	}
}

// Start starts the worker's main processing loop
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Worker starting")

	atomic.StoreInt32(&w.isActive, 1)
	defer atomic.StoreInt32(&w.isActive, 0)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker stopping due to context cancellation")
			w.cancelCurrentJob()
			return ctx.Err()

		default:
			// Process next job
			if err := w.processNextJob(ctx); err != nil {
				w.logger.Error("Error processing job", zap.Error(err))
				// Continue processing other jobs even if one fails
			}
		}
	}
}


func (w *Worker) processNextJob(ctx context.Context) error {
	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	w.currentJobCtx = jobCtx
	w.currentJobCancel = cancel
	defer func() {
		w.currentJobCtx = nil
		w.currentJobCancel = nil
	}()

	// Fetch job from queue
	job, err := w.queue.Dequeue(jobCtx)
	if err != nil {
		return err
	}

	// No job available
	if job == nil {
		// Short sleep to prevent tight polling
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.config.PollInterval):
			return nil
		}
	}

	// Process the job
	return w.executeJob(jobCtx, job)
}

// GetStats returns current worker statistics
func (w *Worker) GetStats() WorkerStats {
	return WorkerStats{
		WorkerID:       w.config.ID,
		JobsProcessed:  atomic.LoadInt64(&w.jobsProcessed),
		JobsFailed:     atomic.LoadInt64(&w.jobsFailed),
		JobsRetried:    atomic.LoadInt64(&w.jobsRetried),
		IsActive:       w.IsActive(),
	}
}

// executes a single job
func (w *Worker) executeJob(ctx context.Context, job *types.Job) error {
	startTime := time.Now()

	// Add start time to context for duration calculation
	ctx = context.WithValue(ctx, startTimeKey, startTime.UnixNano())

	w.logger.Info("Starting job execution",
		zap.String("job_id", job.ID),
		zap.String("job_type", job.Type),
		zap.Int("attempt", job.Attempts+1),
		zap.Int("max_retries", job.MaxRetries),
	)

	// Increment attempt counter
	job.IncrementAttempts()
	
	// Process job using registry
	result := w.registry.Process(ctx, job)

	switch result.Status {
	case types.StatusCompleted:
		atomic.AddInt64(&w.jobsProcessed, 1)
		w.logger.Info("Job completed successfully",
			zap.String("job_id", job.ID),
			zap.String("duration", result.Duration),
		)
		
	case types.StatusFailed:
		atomic.AddInt64(&w.jobsFailed, 1)
		
		// Check if we should retry
		if job.ShouldRetry() {
			atomic.AddInt64(&w.jobsRetried, 1)
			w.logger.Warn("Job failed, retrying",
				zap.String("job_id", job.ID),
				zap.String("error", result.Error),
				zap.Int("attempt", job.Attempts),
				zap.Int("max_retries", job.MaxRetries),
			)
			
			// Re-enqueue job for retry with exponential backoff
			if err := w.requeueJobWithDelay(ctx, job); err != nil {
				w.logger.Error("Failed to requeue job for retry",
					zap.String("job_id", job.ID),
					zap.Error(err),
				)
			}
		} else {
			w.logger.Error("Job failed permanently",
				zap.String("job_id", job.ID),
				zap.String("error", result.Error),
				zap.Int("attempts", job.Attempts),
			)
			
		}
	}
	
	return nil
}

func (w *Worker) requeueJobWithDelay(ctx context.Context, job *types.Job) error {

	delay := time.Duration(1<<uint(job.Attempts-1)) * time.Second

	// Delay cap at 5 min
	if delay > 5*time.Minute {
		delay = 5*time.Minute
	}

	w.logger.Info("Scheduling job retry",
	zap.String("job_id", job.ID),
	zap.Duration("delay", delay),)

	go func(){
		time.Sleep(delay)

		retryCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := w.queue.Enqueue(retryCtx, job); err != nil {
			w.logger.Error("Failed to enqueue retry job",
				zap.String("job_id", job.ID),
				zap.Error(err),
			)
		}
	}()

	return nil
}


// IsActive returns true if the worker is currently active
func (w *Worker) IsActive() bool {
	return atomic.LoadInt32(&w.isActive) == 1
}

// cancelCurrentJob cancels the currently running job
func (w *Worker) cancelCurrentJob() {
	if w.currentJobCancel != nil {
		w.logger.Info("Cancelling current job")
		w.currentJobCancel()
	}
}