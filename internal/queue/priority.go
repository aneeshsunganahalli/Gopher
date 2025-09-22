package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"github.com/go-redis/redis/v8"
)

// Priority levels
const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// Queue keys by priority
const (
	highPriorityQueueKey   = "queue:high"
	normalPriorityQueueKey = "queue:normal"
	lowPriorityQueueKey    = "queue:low"
)

// PriorityQueue implements Queue interface with priority levels
type PriorityQueue struct {
	client        redis.Cmdable
	opts          RedisOptions
	priorityRatio map[string]int // Processing ratio for different priority levels
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(opts RedisOptions) (*PriorityQueue, error) {
	// Parse URL to create new client
	redisOpts, err := redis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	redisOpts.Password = opts.Password
	redisOpts.DB = opts.DB
	redisOpts.DialTimeout = opts.ConnectTimeout
	redisOpts.ReadTimeout = opts.CommandTimeout
	redisOpts.WriteTimeout = opts.CommandTimeout

	client := redis.NewClient(redisOpts)

	ctx, cancel := context.WithTimeout(context.Background(), opts.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Default processing ratio: process 5 high, 3 normal, 1 low priority jobs
	priorityRatio := map[string]int{
		PriorityHigh:   5,
		PriorityNormal: 3,
		PriorityLow:    1,
	}

	return &PriorityQueue{
		client:        client,
		opts:          opts,
		priorityRatio: priorityRatio,
	}, nil
}

// SetPriorityRatio configures the ratio for processing jobs of different priorities
func (p *PriorityQueue) SetPriorityRatio(high, normal, low int) {
	p.priorityRatio = map[string]int{
		PriorityHigh:   high,
		PriorityNormal: normal,
		PriorityLow:    low,
	}
}

// Enqueue adds a job to the queue with the specified priority
func (p *PriorityQueue) Enqueue(ctx context.Context, job *types.Job) error {
	if err := job.Validate(); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Get priority from job metadata or default to normal
	priority := PriorityNormal
	if job.Metadata != nil {
		if priorityVal, ok := job.Metadata["priority"]; ok {
			if priorityStr, ok := priorityVal.(string); ok {
				if priorityStr == PriorityHigh || priorityStr == PriorityLow {
					priority = priorityStr
				}
			}
		}
	}

	// Serialize job to JSON
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Select queue key based on priority
	queueKey := normalPriorityQueueKey
	switch priority {
	case PriorityHigh:
		queueKey = highPriorityQueueKey
	case PriorityLow:
		queueKey = lowPriorityQueueKey
	}

	pipe := p.client.Pipeline()

	// Add job to the appropriate priority queue
	pipe.LPush(ctx, queueKey, jobData)

	// Update stats
	pipe.HIncrBy(ctx, statsKey, "total_enqueued", 1)
	pipe.HIncrBy(ctx, statsKey, fmt.Sprintf("enqueued:%s", priority), 1)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// Dequeue removes and returns a job from the queue, respecting priority ratios
func (p *PriorityQueue) Dequeue(ctx context.Context) (*types.Job, error) {
	// Get current counts to determine which queue to pull from
	counters, err := p.getPriorityCounters(ctx)
	if err != nil {
		return nil, err
	}

	// Determine which queue to pull from based on ratio
	queueKey := p.selectQueueByRatio(counters)

	// Try to get a job from the selected queue
	result := p.client.BRPop(ctx, time.Second, queueKey)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			// No job available, try other queues in priority order
			for _, key := range []string{highPriorityQueueKey, normalPriorityQueueKey, lowPriorityQueueKey} {
				if key == queueKey {
					continue // Already tried this one
				}

				result = p.client.BRPop(ctx, 0, key)
				if err := result.Err(); err != nil {
					if err == redis.Nil {
						continue
					}
					return nil, fmt.Errorf("failed to dequeue job: %w", err)
				}

				// Found a job, break out of loop
				break
			}

			// If still no job after trying all queues
			if result.Err() == redis.Nil {
				return nil, nil
			}
		} else {
			return nil, fmt.Errorf("failed to dequeue job: %w", err)
		}
	}

	values := result.Val()
	if len(values) != 2 {
		return nil, fmt.Errorf("unexpected BRPOP result: %v", values)
	}

	jobData := values[1]

	// Update dequeue stats
	priority := "normal"
	switch values[0] {
	case highPriorityQueueKey:
		priority = PriorityHigh
	case lowPriorityQueueKey:
		priority = PriorityLow
	}

	pipe := p.client.Pipeline()
	pipe.HIncrBy(ctx, statsKey, "total_dequeued", 1)
	pipe.HIncrBy(ctx, statsKey, fmt.Sprintf("dequeued:%s", priority), 1)
	pipe.HIncrBy(ctx, "priority_counters", priority, 1)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update dequeue stats: %w", err)
	}

	// Deserialize job
	var job types.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// selectQueueByRatio determines which queue to pull from based on the priority ratio
func (p *PriorityQueue) selectQueueByRatio(counters map[string]int) string {
	// Calculate ratio of high:normal:low based on configured values and current counts
	highRatio := float64(p.priorityRatio[PriorityHigh]) / float64(counters[PriorityHigh]+1)
	normalRatio := float64(p.priorityRatio[PriorityNormal]) / float64(counters[PriorityNormal]+1)
	lowRatio := float64(p.priorityRatio[PriorityLow]) / float64(counters[PriorityLow]+1)

	// Select queue with highest ratio
	if highRatio >= normalRatio && highRatio >= lowRatio {
		return highPriorityQueueKey
	} else if normalRatio >= highRatio && normalRatio >= lowRatio {
		return normalPriorityQueueKey
	} else {
		return lowPriorityQueueKey
	}
}

// getPriorityCounters gets the current dequeue counters for each priority
func (p *PriorityQueue) getPriorityCounters(ctx context.Context) (map[string]int, error) {
	counters := map[string]int{
		PriorityHigh:   0,
		PriorityNormal: 0,
		PriorityLow:    0,
	}

	// Get current counters
	result := p.client.HGetAll(ctx, "priority_counters")
	if err := result.Err(); err != nil && err != redis.Nil {
		return counters, fmt.Errorf("failed to get priority counters: %w", err)
	}

	// Parse counters
	for k, v := range result.Val() {
		var count int
		if _, err := fmt.Sscanf(v, "%d", &count); err == nil {
			counters[k] = count
		}
	}

	return counters, nil
}

// Size returns the current number of jobs in all priority queues
func (p *PriorityQueue) Size(ctx context.Context) (int, error) {
	pipe := p.client.Pipeline()

	highCmd := pipe.LLen(ctx, highPriorityQueueKey)
	normalCmd := pipe.LLen(ctx, normalPriorityQueueKey)
	lowCmd := pipe.LLen(ctx, lowPriorityQueueKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue sizes: %w", err)
	}

	total := int(highCmd.Val() + normalCmd.Val() + lowCmd.Val())
	return total, nil
}

// SizeByPriority returns the size of each priority queue
func (p *PriorityQueue) SizeByPriority(ctx context.Context) (map[string]int, error) {
	pipe := p.client.Pipeline()

	highCmd := pipe.LLen(ctx, highPriorityQueueKey)
	normalCmd := pipe.LLen(ctx, normalPriorityQueueKey)
	lowCmd := pipe.LLen(ctx, lowPriorityQueueKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue sizes: %w", err)
	}

	return map[string]int{
		PriorityHigh:   int(highCmd.Val()),
		PriorityNormal: int(normalCmd.Val()),
		PriorityLow:    int(lowCmd.Val()),
	}, nil
}

// Health checks if the queue is healthy/reachable
func (p *PriorityQueue) Health(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

// Close closes the queue connection
func (p *PriorityQueue) Close() error {
	if client, ok := p.client.(*redis.Client); ok {
		return client.Close()
	}
	return nil
}
