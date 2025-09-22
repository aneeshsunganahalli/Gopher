package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"github.com/go-redis/redis/v8"
)

const (
	scheduledJobsKey      = "scheduled_jobs"  // Redis sorted set storing scheduled jobs
	scheduledJobsStatsKey = "scheduled_stats" // Redis hash storing scheduled job stats
)

// ScheduledQueue manages delayed and recurring jobs
type ScheduledQueue struct {
	client redis.Cmdable
	queue  Queue // Reference to the main queue for moving due jobs
}

// NewScheduledQueue creates a new scheduled job queue
func NewScheduledQueue(client redis.Cmdable, queue Queue) *ScheduledQueue {
	return &ScheduledQueue{
		client: client,
		queue:  queue,
	}
}

// Schedule adds a job to be processed at a future time
func (s *ScheduledQueue) Schedule(ctx context.Context, job *types.Job, executeAt time.Time) error {
	if err := job.Validate(); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Create scheduled job wrapper
	scheduledJob := &types.ScheduledJob{
		Job:       job,
		ExecuteAt: executeAt,
		Recurring: false,
	}

	return s.addScheduledJob(ctx, scheduledJob)
}

// ScheduleRecurring adds a recurring job with the specified cron expression
func (s *ScheduledQueue) ScheduleRecurring(ctx context.Context, job *types.Job, cronExpr string) error {
	if err := job.Validate(); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Validate cron expression
	schedule, err := parseCronExpression(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next execution time
	nextExec := schedule.Next(time.Now())

	// Create scheduled job wrapper
	scheduledJob := &types.ScheduledJob{
		Job:            job,
		ExecuteAt:      nextExec,
		Recurring:      true,
		CronExpression: cronExpr,
	}

	return s.addScheduledJob(ctx, scheduledJob)
}

// addScheduledJob adds a job to the scheduled queue
func (s *ScheduledQueue) addScheduledJob(ctx context.Context, scheduledJob *types.ScheduledJob) error {
	// Serialize job
	jobData, err := json.Marshal(scheduledJob)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled job: %w", err)
	}

	// Add to sorted set with score as Unix timestamp
	score := float64(scheduledJob.ExecuteAt.Unix())
	err = s.client.ZAdd(ctx, scheduledJobsKey, &redis.Z{
		Score:  score,
		Member: jobData,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	// Update stats
	pipe := s.client.Pipeline()
	pipe.HIncrBy(ctx, scheduledJobsStatsKey, "total", 1)
	if scheduledJob.Recurring {
		pipe.HIncrBy(ctx, scheduledJobsStatsKey, "recurring", 1)
	} else {
		pipe.HIncrBy(ctx, scheduledJobsStatsKey, "one_time", 1)
	}
	pipe.HIncrBy(ctx, scheduledJobsStatsKey, fmt.Sprintf("type:%s", scheduledJob.Job.Type), 1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update scheduled job stats: %w", err)
	}

	return nil
}

// ProcessDueJobs moves jobs that are due to the main queue
func (s *ScheduledQueue) ProcessDueJobs(ctx context.Context) (int, error) {
	now := time.Now().Unix()

	// Get all jobs that are due
	result := s.client.ZRangeByScore(ctx, scheduledJobsKey, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now),
	})

	if err := result.Err(); err != nil {
		return 0, fmt.Errorf("failed to get due jobs: %w", err)
	}

	jobs := result.Val()
	processedCount := 0

	for _, jobData := range jobs {
		var scheduledJob types.ScheduledJob
		if err := json.Unmarshal([]byte(jobData), &scheduledJob); err != nil {
			continue
		}

		// Move to main queue
		if err := s.queue.Enqueue(ctx, scheduledJob.Job); err != nil {
			continue
		}

		// Remove from scheduled queue
		s.client.ZRem(ctx, scheduledJobsKey, jobData)

		// If recurring, schedule next execution
		if scheduledJob.Recurring {
			schedule, err := parseCronExpression(scheduledJob.CronExpression)
			if err == nil {
				// Calculate next execution time
				nextExec := schedule.Next(time.Now())

				// Create new job for next execution
				nextJob := *scheduledJob.Job // Clone the job
				nextJob.ID = generateJobID() // Generate a new ID
				nextJob.Attempts = 0         // Reset attempts
				nextJob.CreatedAt = time.Now().UTC()
				nextJob.UpdatedAt = time.Now().UTC()

				// Schedule next execution
				nextScheduledJob := types.ScheduledJob{
					Job:            &nextJob,
					ExecuteAt:      nextExec,
					Recurring:      true,
					CronExpression: scheduledJob.CronExpression,
				}

				s.addScheduledJob(ctx, &nextScheduledJob)
			}
		} else {
			// Update stats for one-time jobs
			s.client.HIncrBy(ctx, scheduledJobsStatsKey, "one_time", -1)
		}

		processedCount++
	}

	return processedCount, nil
}

// Size returns the number of scheduled jobs
func (s *ScheduledQueue) Size(ctx context.Context) (int, error) {
	result := s.client.ZCard(ctx, scheduledJobsKey)
	if err := result.Err(); err != nil {
		return 0, fmt.Errorf("failed to get scheduled queue size: %w", err)
	}

	return int(result.Val()), nil
}

// parseCronExpression parses a cron expression (stub - would use a cron library)
func parseCronExpression(expr string) (CronSchedule, error) {
	// This is a simplified stub - in a real implementation, you'd use a proper cron library
	// such as github.com/robfig/cron

	// For now, just return a simple implementation that schedules for 1 minute in the future
	return &simpleCronSchedule{}, nil
}

// CronSchedule interface for calculating next execution time
type CronSchedule interface {
	Next(time.Time) time.Time
}

// Simple implementation for the stub
type simpleCronSchedule struct{}

func (s *simpleCronSchedule) Next(t time.Time) time.Time {
	return t.Add(1 * time.Minute)
}

// Helper function to generate a job ID (temporary implementation)
func generateJobID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}
