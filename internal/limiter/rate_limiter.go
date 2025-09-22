package limiter

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter interface for rate limiting job processing
type RateLimiter interface {
	// Allow checks if a job of the given type can be processed
	Allow(ctx context.Context, jobType string) (bool, error)

	// Done marks the completion of a job processing
	Done(ctx context.Context, jobType string) error

	// SetLimit sets the rate limit for a job type
	SetLimit(ctx context.Context, jobType string, limit float64, burst int) error
} // LocalRateLimiter implements in-memory rate limiting
type LocalRateLimiter struct {
	mu           sync.RWMutex
	limits       map[string]float64 // requests per second
	bursts       map[string]int
	lastAllowed  map[string]time.Time
	tokenBuckets map[string]float64
	defaults     float64
	defaultBurst int
}

// NewLocalRateLimiter creates a new in-memory rate limiter
func NewLocalRateLimiter(defaultLimit float64, defaultBurst int) *LocalRateLimiter {
	return &LocalRateLimiter{
		limits:       make(map[string]float64),
		bursts:       make(map[string]int),
		lastAllowed:  make(map[string]time.Time),
		tokenBuckets: make(map[string]float64),
		defaults:     defaultLimit,
		defaultBurst: defaultBurst,
	}
}

// Allow checks if a job can be processed under rate limits
func (l *LocalRateLimiter) Allow(ctx context.Context, jobType string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Get or initialize limiter for this job type
	limit, ok := l.limits[jobType]
	if !ok {
		limit = l.defaults
		l.limits[jobType] = limit
	}

	burst, ok := l.bursts[jobType]
	if !ok {
		burst = l.defaultBurst
		l.bursts[jobType] = burst
	}

	lastTime, ok := l.lastAllowed[jobType]
	if !ok {
		lastTime = time.Now().Add(-24 * time.Hour) // Default to a day ago
		l.lastAllowed[jobType] = lastTime
	}

	tokens, ok := l.tokenBuckets[jobType]
	if !ok {
		tokens = float64(burst)
		l.tokenBuckets[jobType] = tokens
	}

	// Calculate token refill based on time elapsed
	now := time.Now()
	elapsed := now.Sub(lastTime)
	refill := elapsed.Seconds() * limit
	newTokens := tokens + refill
	if newTokens > float64(burst) {
		newTokens = float64(burst)
	}

	// Try to take a token
	if newTokens < 1 {
		return false, nil
	}

	// Take a token and update state
	newTokens--
	l.tokenBuckets[jobType] = newTokens
	l.lastAllowed[jobType] = now

	return true, nil
}

// Done is a no-op for the local limiter
func (l *LocalRateLimiter) Done(ctx context.Context, jobType string) error {
	// No-op for local limiter
	return nil
}

// SetLimit updates the rate limit for a job type
func (l *LocalRateLimiter) SetLimit(ctx context.Context, jobType string, limit float64, burst int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.limits[jobType] = limit
	l.bursts[jobType] = burst
	return nil
}

// RedisRateLimiter implements distributed rate limiting using Redis
type RedisRateLimiter struct {
	client       redis.Cmdable
	prefix       string
	defaults     float64
	defaultBurst int
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(client redis.Cmdable, prefix string, defaultLimit float64, defaultBurst int) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:       client,
		prefix:       prefix,
		defaults:     defaultLimit,
		defaultBurst: defaultBurst,
	}
}

// Allow checks if a job can be processed using Redis-based token bucket
func (r *RedisRateLimiter) Allow(ctx context.Context, jobType string) (bool, error) {
	limitsKey := fmt.Sprintf("%s:limits:%s", r.prefix, jobType)
	tokensKey := fmt.Sprintf("%s:tokens:%s", r.prefix, jobType)

	// Get current limits for this job type
	pipe := r.client.Pipeline()
	limitCmd := pipe.HGet(ctx, limitsKey, "limit")
	burstCmd := pipe.HGet(ctx, limitsKey, "burst")
	lastUpdatedCmd := pipe.HGet(ctx, limitsKey, "last_updated")
	currentTokensCmd := pipe.Get(ctx, tokensKey)
	_, err := pipe.Exec(ctx)

	// Parse values with defaults
	limit := r.defaults
	burst := r.defaultBurst
	var lastUpdated time.Time
	currentTokens := float64(burst)

	if limitVal, err := limitCmd.Result(); err == nil {
		if l, err := strconv.ParseFloat(limitVal, 64); err == nil {
			limit = l
		}
	}

	if burstVal, err := burstCmd.Result(); err == nil {
		if b, err := strconv.Atoi(burstVal); err == nil {
			burst = b
		}
	}

	if lastUpdatedVal, err := lastUpdatedCmd.Result(); err == nil {
		if t, err := time.Parse(time.RFC3339, lastUpdatedVal); err == nil {
			lastUpdated = t
		}
	} else {
		lastUpdated = time.Now().Add(-24 * time.Hour) // Default to a day ago
	}

	if tokensVal, err := currentTokensCmd.Result(); err == nil {
		if t, err := strconv.ParseFloat(tokensVal, 64); err == nil {
			currentTokens = t
		}
	}

	// Calculate token refill based on time elapsed
	now := time.Now()
	elapsed := now.Sub(lastUpdated)
	refill := float64(elapsed.Seconds()) * float64(limit)
	newTokens := currentTokens + refill
	if newTokens > float64(burst) {
		newTokens = float64(burst)
	}

	// Try to take a token
	if newTokens < 1 {
		return false, nil
	}

	// Take a token and update state
	newTokens--
	pipe = r.client.Pipeline()
	pipe.Set(ctx, tokensKey, fmt.Sprintf("%.6f", newTokens), 0)
	pipe.HSet(ctx, limitsKey, "last_updated", now.Format(time.RFC3339))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to update rate limit tokens: %w", err)
	}

	return true, nil
}

// Done is a no-op for the Redis limiter (token is already taken in Allow)
func (r *RedisRateLimiter) Done(ctx context.Context, jobType string) error {
	// No-op for this implementation
	return nil
}

// SetLimit updates the rate limit for a job type
func (r *RedisRateLimiter) SetLimit(ctx context.Context, jobType string, limit float64, burst int) error {
	limitsKey := fmt.Sprintf("%s:limits:%s", r.prefix, jobType)

	pipe := r.client.Pipeline()
	pipe.HSet(ctx, limitsKey, map[string]interface{}{
		"limit": fmt.Sprintf("%.6f", limit),
		"burst": fmt.Sprintf("%d", burst),
	})
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set rate limit: %w", err)
	}

	return nil
}
