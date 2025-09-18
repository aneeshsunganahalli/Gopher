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
	jobQueueKey = "job_queue"   //  Redis list storing jobs.
	statsKey    = "queue_stats" //  Redis hash storing counters like total enqueued/dequeued
)

type RedisOptions struct {
	URL            string
	Password       string
	DB             int
	ConnectTimeout time.Duration
	CommandTimeout time.Duration
}

type RedisQueue struct {
	client redis.Cmdable // Client used to talk to Redis
	opts   RedisOptions
}

func NewRedisQueue(opts RedisOptions) (*RedisQueue, error) {
	// Parse URl to create new client
	redisOpts, err := redis.ParseURL(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	redisOpts.Password = opts.Password
	redisOpts.DB = opts.DB
	redisOpts.DialTimeout = opts.ConnectTimeout
	redisOpts.ReadTimeout = opts.CommandTimeout
	redisOpts.WriteTimeout = opts.CommandTimeout

	client := redis.NewClient(redisOpts) // creates actual connection pool to redis

	ctx, cancel := context.WithTimeout(context.Background(), opts.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisQueue{
		client: client,
		opts:   opts,
	}, nil
}

func (r *RedisQueue) Enqueue(ctx context.Context, job *types.Job) error {
	if err := job.Validate(); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Serialize job to JSON
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	pipe := r.client.Pipeline() // used for atomic operations

	pipe.LPush(ctx, jobQueueKey, jobData) // adding job to queue

	pipe.HIncrBy(ctx, statsKey, "total_enqueued", 1)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

func (r *RedisQueue) Dequeue(ctx context.Context) (*types.Job, error) {
	result := r.client.BRPop(ctx, time.Second, jobQueueKey)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			// No job available, this is normal
			return nil, nil
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	values := result.Val()
	if len(values) != 2 {
		return nil, fmt.Errorf("unexpected BRPOP result: %w", values)
	}

	jobData := values[1]

	var job types.Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	go func() {
		// Use background context to avoid cancellation affecting stats
		statsCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r.client.HIncrBy(statsCtx, statsKey, "total_dequeued", 1)
	}()

	return &job, nil
}

func (r *RedisQueue) Size(ctx context.Context) (int, error) {
	result := r.client.LLen(ctx, jobQueueKey)
	if err := result.Err(); err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}
	return int(result.Val()), nil
}

func (r *RedisQueue) Health(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (r *RedisQueue) Close() error {
	if client, ok := r.client.(*redis.Client); ok {
		return client.Close()
	}
	return nil
}

func (r *RedisQueue) GetStats(ctx context.Context) (*QueueStats, error) {
	pipe := r.client.Pipeline()

	sizeCmd := pipe.LLen(ctx, jobQueueKey)
	statsCmd := pipe.HGetAll(ctx, statsKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := &QueueStats{
		QueueSize: int(sizeCmd.Val()),
	}

	// Parse statistics if they exist
	if statsData := statsCmd.Val(); len(statsData) > 0 {
		if enqueued, exists := statsData["total_enqueued"]; exists {
			fmt.Sscanf(enqueued, "%d", &stats.TotalEnqueued)
		}
		if dequeued, exists := statsData["total_dequeued"]; exists {
			fmt.Sscanf(dequeued, "%d", &stats.TotalDequeued)
		}
	}

	return stats, nil
}
