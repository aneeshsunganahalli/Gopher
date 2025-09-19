package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aneeshsunganahalli/Gopher/examples/handlers"
	"github.com/aneeshsunganahalli/Gopher/internal/config"
	"github.com/aneeshsunganahalli/Gopher/internal/job"
	"github.com/aneeshsunganahalli/Gopher/internal/queue"
	"github.com/aneeshsunganahalli/Gopher/internal/worker"
	"go.uber.org/zap"
)


func main() {
cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger, err := initLogger(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting job queue worker",
		zap.String("version", "1.0.0"),
		zap.Int("concurrency", cfg.Worker.Concurrency),
	)

	// Initialize Redis queue
	redisConfig := queue.RedisOptions{
		URL:             cfg.Redis.URL,
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.DB,
		ConnectTimeout:  cfg.Redis.Timeout,
		CommandTimeout:  cfg.Redis.Timeout,
	}

	jobQueue, err := queue.NewRedisQueue(redisConfig)
	if err != nil {
		logger.Fatal("Failed to initialize Redis queue", zap.Error(err))
	}
	defer jobQueue.Close()

	// Initialize job registry
	registry := job.NewRegistry(logger)

	// Register job handlers
	if err := registerJobHandlers(registry, logger); err != nil {
		logger.Fatal("Failed to register job handlers", zap.Error(err))
	}

// Initialize worker pool
	poolConfig := worker.PoolConfig{
		Concurrency:     cfg.Worker.Concurrency,
		ShutdownTimeout: cfg.Worker.ShutdownTimeout,
		PollInterval:    cfg.Worker.PollInterval,
	}	

	pool := worker.NewPool(poolConfig, jobQueue, registry, logger)

	// Start worker pool
	if err := pool.Start(); err != nil {
		logger.Fatal("Failed to start worker pool", zap.Error(err))
	}

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker pool...")

	// Stop worker pool gracefully
	if err := pool.Stop(); err != nil {
		logger.Error("Failed to shutdown worker pool gracefully", zap.Error(err))
	}

	logger.Info("Worker pool shutdown complete")
}

func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	if cfg.Format == "console" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return zapConfig.Build()
}

func registerJobHandlers(registry *job.Registry, logger *zap.Logger) error {
	// Register email handler
	emailHandler := handlers.NewEmailJobHandler(logger)
	if err := registry.Register(emailHandler); err != nil {
		return err
	}

	// Register image handler
	imageHandler := handlers.NewImageJobHandler(logger)
	if err := registry.Register(imageHandler); err != nil {
		return err
	}

	// Register math handler
	mathHandler := handlers.NewMathJobHandler(logger)
	if err := registry.Register(mathHandler); err != nil {
		return err
	}

	logger.Info("All job handlers registered successfully")
	return nil
}