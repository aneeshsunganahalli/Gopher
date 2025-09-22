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
	deadLetterQueueKey = "dlq:jobs"  // Redis list storing dead letter jobs
	dlqStatsKey        = "dlq:stats" // Redis hash storing DLQ stats
)

// DeadLetterQueue handles failed jobs that have exhausted retry attempts
type DeadLetterQueue interface {
	// Send a job to the dead letter queue
	Send(ctx context.Context, job *types.Job, errorMsg string) error

	// Get the number of jobs in the DLQ
	Size(ctx context.Context) (int, error)

	// Reprocess a job from the DLQ by moving it back to the main queue
	Reprocess(ctx context.Context, jobID string) error

	// List jobs in the DLQ with pagination
	List(ctx context.Context, offset, limit int) ([]*types.FailedJobInfo, error)
}

// FailedJobInfo contains information about a failed job in the DLQ
type FailedJobInfo struct {
	Job      *types.Job `json:"job"`
	Error    string     `json:"error"`
	FailedAt time.Time  `json:"failed_at"`
}

// RedisDLQ implements the DeadLetterQueue interface using Redis
type RedisDLQ struct {
	client redis.Cmdable
	queue  Queue // Reference to the main queue for reprocessing
}

// NewRedisDLQ creates a new Redis-backed dead letter queue
func NewRedisDLQ(client redis.Cmdable, queue Queue) *RedisDLQ {
	return &RedisDLQ{
		client: client,
		queue:  queue,
	}
}

// Send puts a failed job into the dead letter queue
func (d *RedisDLQ) Send(ctx context.Context, job *types.Job, errorMsg string) error {
	failedInfo := &types.FailedJobInfo{
		Job:      job,
		Error:    errorMsg,
		FailedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(failedInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal failed job info: %w", err)
	}

	pipe := d.client.Pipeline()

	// Add to DLQ
	pipe.LPush(ctx, deadLetterQueueKey, data)

	// Update stats
	pipe.HIncrBy(ctx, dlqStatsKey, "total", 1)
	pipe.HIncrBy(ctx, dlqStatsKey, fmt.Sprintf("type:%s", job.Type), 1)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to send job to DLQ: %w", err)
	}

	return nil
}

// Size returns the number of jobs in the DLQ
func (d *RedisDLQ) Size(ctx context.Context) (int, error) {
	result := d.client.LLen(ctx, deadLetterQueueKey)
	if err := result.Err(); err != nil {
		return 0, fmt.Errorf("failed to get DLQ size: %w", err)
	}

	return int(result.Val()), nil
}

// Reprocess moves a job from the DLQ back to the main queue
func (d *RedisDLQ) Reprocess(ctx context.Context, jobID string) error {
	// Get all jobs in the DLQ
	result := d.client.LRange(ctx, deadLetterQueueKey, 0, -1)
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to list DLQ jobs: %w", err)
	}

	found := false
	var failedInfo types.FailedJobInfo

	// Find the job with the matching ID
	for _, item := range result.Val() {
		if err := json.Unmarshal([]byte(item), &failedInfo); err != nil {
			continue
		}

		if failedInfo.Job.ID == jobID {
			found = true

			// Reset attempts counter
			failedInfo.Job.Attempts = 0
			failedInfo.Job.UpdatedAt = time.Now().UTC()

			// Remove from DLQ
			d.client.LRem(ctx, deadLetterQueueKey, 1, item)

			// Add to main queue
			if err := d.queue.Enqueue(ctx, failedInfo.Job); err != nil {
				return fmt.Errorf("failed to requeue job: %w", err)
			}

			// Update stats
			pipe := d.client.Pipeline()
			pipe.HIncrBy(ctx, dlqStatsKey, "total", -1)
			pipe.HIncrBy(ctx, dlqStatsKey, fmt.Sprintf("type:%s", failedInfo.Job.Type), -1)
			pipe.HIncrBy(ctx, dlqStatsKey, "reprocessed", 1)
			_, err := pipe.Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update DLQ stats: %w", err)
			}

			break
		}
	}

	if !found {
		return fmt.Errorf("job with ID %s not found in DLQ", jobID)
	}

	return nil
}

// List returns jobs in the DLQ with pagination
func (d *RedisDLQ) List(ctx context.Context, offset, limit int) ([]*types.FailedJobInfo, error) {
	result := d.client.LRange(ctx, deadLetterQueueKey, int64(offset), int64(offset+limit-1))
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to list DLQ jobs: %w", err)
	}

	jobs := make([]*types.FailedJobInfo, 0, len(result.Val()))

	for _, item := range result.Val() {
		var failedInfo types.FailedJobInfo
		if err := json.Unmarshal([]byte(item), &failedInfo); err != nil {
			continue
		}

		jobs = append(jobs, &failedInfo)
	}

	return jobs, nil
}
