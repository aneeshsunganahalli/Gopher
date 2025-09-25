package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aneeshsunganahalli/Gopher/internal/config"
	"github.com/aneeshsunganahalli/Gopher/internal/queue"
	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "gopher",
	Short: "Gopher is a distributed task queue for Go",
	Long: `A distributed task queue built in Go with Redis backend.
Complete documentation is available at https://github.com/aneeshsunganahalli/Gopher`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func init() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize Redis connection
	redisOpts := queue.RedisOptions{
		URL:            cfg.Redis.URL,
		Password:       cfg.Redis.Password,
		DB:             cfg.Redis.DB,
		ConnectTimeout: cfg.Redis.Timeout,
		CommandTimeout: cfg.Redis.Timeout,
	}

	// Setup commands
	setupCommands(redisOpts, logger)
}

func setupCommands(redisOpts queue.RedisOptions, logger *zap.Logger) {
	// Queue stats command
	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show queue statistics",
		Run: func(cmd *cobra.Command, args []string) {
			printQueueStats(redisOpts, logger)
		},
	}

	// Submit job command
	var jobType, payload string
	var maxRetries int
	var submitCmd = &cobra.Command{
		Use:   "submit",
		Short: "Submit a job to the queue",
		Run: func(cmd *cobra.Command, args []string) {
			submitJob(redisOpts, logger, jobType, payload, maxRetries)
		},
	}
	submitCmd.Flags().StringVarP(&jobType, "type", "t", "", "Job type (required)")
	submitCmd.Flags().StringVarP(&payload, "payload", "p", "{}", "Job payload as JSON")
	submitCmd.Flags().IntVarP(&maxRetries, "retries", "r", 3, "Maximum number of retries")
	submitCmd.MarkFlagRequired("type")

	// List failed jobs command
	var listFailedCmd = &cobra.Command{
		Use:   "list-failed",
		Short: "List failed jobs in the dead letter queue",
		Run: func(cmd *cobra.Command, args []string) {
			listFailedJobs(redisOpts, logger)
		},
	}

	// Retry failed job command
	var jobID string
	var retryCmd = &cobra.Command{
		Use:   "retry",
		Short: "Retry a failed job from the dead letter queue",
		Run: func(cmd *cobra.Command, args []string) {
			retryFailedJob(redisOpts, logger, jobID)
		},
	}
	retryCmd.Flags().StringVarP(&jobID, "id", "i", "", "Job ID to retry (required)")
	retryCmd.MarkFlagRequired("id")

	// Retry all failed jobs command
	var retryAllCmd = &cobra.Command{
		Use:   "retry-all",
		Short: "Retry all failed jobs in the dead letter queue",
		Run: func(cmd *cobra.Command, args []string) {
			retryAllFailedJobs(redisOpts, logger)
		},
	}

	// Purge queue command
	var queueName string
	var purgeCmd = &cobra.Command{
		Use:   "purge",
		Short: "Purge a queue",
		Run: func(cmd *cobra.Command, args []string) {
			purgeQueue(redisOpts, logger, queueName)
		},
	}
	purgeCmd.Flags().StringVarP(&queueName, "queue", "q", "main", "Queue to purge (main, scheduled, failed)")

	// Health check command
	var healthCmd = &cobra.Command{
		Use:   "health",
		Short: "Check system health",
		Run: func(cmd *cobra.Command, args []string) {
			checkHealth(redisOpts, logger)
		},
	}

	// Add all commands to root
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(submitCmd)
	rootCmd.AddCommand(listFailedCmd)
	rootCmd.AddCommand(retryCmd)
	rootCmd.AddCommand(retryAllCmd)
	rootCmd.AddCommand(purgeCmd)
	rootCmd.AddCommand(healthCmd)
}

func printQueueStats(redisOpts queue.RedisOptions, logger *zap.Logger) {
	q, err := queue.NewRedisQueue(redisOpts)
	if err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return
	}
	defer q.Close()

	ctx := context.Background()
	size, err := q.Size(ctx)
	if err != nil {
		logger.Error("Failed to get queue size", zap.Error(err))
		return
	}

	fmt.Printf("Queue Statistics:\n")
	fmt.Printf("----------------\n")
	fmt.Printf("Current queue size: %d\n", size)

	// TODO: Add more statistics
}

func submitJob(redisOpts queue.RedisOptions, logger *zap.Logger, jobType, payload string, maxRetries int) {
	q, err := queue.NewRedisQueue(redisOpts)
	if err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return
	}
	defer q.Close()

	// Parse payload
	var rawPayload json.RawMessage
	if err := json.Unmarshal([]byte(payload), &rawPayload); err != nil {
		logger.Error("Invalid JSON payload", zap.Error(err))
		return
	}

	// Create job
	job := types.NewJob(jobType, rawPayload, maxRetries)

	// Enqueue job
	ctx := context.Background()
	if err := q.Enqueue(ctx, job); err != nil {
		logger.Error("Failed to enqueue job", zap.Error(err))
		return
	}

	fmt.Printf("Job enqueued successfully:\n")
	fmt.Printf("  ID: %s\n", job.ID)
	fmt.Printf("  Type: %s\n", job.Type)
	fmt.Printf("  Max retries: %d\n", job.MaxRetries)
}

func listFailedJobs(redisOpts queue.RedisOptions, logger *zap.Logger) {
	// Implementation will depend on DLQ
	fmt.Println("List of failed jobs:")
	fmt.Println("-------------------")
	// TODO: Implement when DLQ is available
}

func retryFailedJob(redisOpts queue.RedisOptions, logger *zap.Logger, jobID string) {
	// Implementation will depend on DLQ
	fmt.Printf("Retrying job %s...\n", jobID)
	// TODO: Implement when DLQ is available
}

func retryAllFailedJobs(redisOpts queue.RedisOptions, logger *zap.Logger) {
	// Implementation will depend on DLQ
	fmt.Println("Retrying all failed jobs...")
	// TODO: Implement when DLQ is available
}

func purgeQueue(redisOpts queue.RedisOptions, logger *zap.Logger, queueName string) {
	// This would require implementing a purge method on the queue
	fmt.Printf("Purging %s queue...\n", queueName)
	// TODO: Implement queue purge functionality
}

func checkHealth(redisOpts queue.RedisOptions, logger *zap.Logger) {
	q, err := queue.NewRedisQueue(redisOpts)
	if err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		fmt.Println("❌ System health check failed: Redis connection error")
		return
	}
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := q.Health(ctx); err != nil {
		logger.Error("Redis health check failed", zap.Error(err))
		fmt.Println("❌ System health check failed: Redis unhealthy")
		return
	}

	fmt.Println("✅ System health check passed")
	fmt.Println("  Redis: Connected and healthy")
}
