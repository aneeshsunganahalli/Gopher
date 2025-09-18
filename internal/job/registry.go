package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"go.uber.org/zap"
)

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]types.JobHandler
	logger   *zap.Logger
}

// NewRegistry creates a new job handler registry
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		handlers: make(map[string]types.JobHandler),
		logger:   logger,
	}
}

// Register adds a job handler to the registry
func (r *Registry) Register(handler types.JobHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	jobType := handler.Type()
	if jobType == "" {
		return fmt.Errorf("handler type cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[jobType]; exists {
		return fmt.Errorf("handler for type '%s' already exists", jobType)
	}

	r.handlers[jobType] = handler
	r.logger.Info("Registered job handler",
		zap.String("type", jobType),
		zap.String("description", handler.Description()),
	)

	return nil
}

// Get retrieves a handler for the given job type
func (r *Registry) Get(jobType string) (types.JobHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[jobType]
	if !exists {
		return nil, fmt.Errorf("no handler registeed for job type %s", jobType)
	}

	return handler, nil
}

// Types returns all registered job types
func (r *Registry) Type() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}

	return types
}

// Process executes a job using appropriate handler
func (r *Registry) Process(ctx context.Context, job *types.Job) *types.JobResult {
	startTime := ctx.Value("start_time").(int64)

	result := &types.JobResult{
		JobID:       job.ID,
		CompletedAt: time.Now().UTC(),
	}

	// Calculate duration
	duration := time.Since(time.Unix(0, startTime))
	result.Duration = duration.String()

	// Get handler
	handler, err := r.Get(job.Type)
	if err != nil {
		result.Status = types.StatusFailed
		result.Error = err.Error()
		r.logger.Error("No handler found for job",
			zap.String("job_id", job.ID),
			zap.String("job_type", job.Type),
			zap.Error(err),
		)
		return result
	}

	// Execute job
	r.logger.Info("Processing job",
		zap.String("job_id", job.ID),
		zap.String("job_type", job.Type),
		zap.Int("attempt", job.Attempts+1),
	)

	if err := handler.Handle(ctx, job); err != nil {
		result.Status = types.StatusFailed
		result.Error = err.Error()

		r.logger.Error("Job processing failed",
			zap.String("job_id", job.ID),
			zap.String("job_type", job.Type),
			zap.Error(err),
			zap.Duration("duration", duration),
		)

		return result
	}

	result.Status = types.StatusCompleted
	r.logger.Info("Job completed successfully",
		zap.String("job_id", job.ID),
		zap.String("job_type", job.Type),
		zap.Duration("duration", duration),
	)

	return result
}

func (r *Registry) ListHandlers() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make(map[string]string)
	for t, h := range r.handlers {
		handlers[t] = h.Description()
	}
	return handlers
}
