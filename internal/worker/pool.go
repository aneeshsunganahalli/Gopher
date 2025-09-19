package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aneeshsunganahalli/Gopher/internal/job"
	"github.com/aneeshsunganahalli/Gopher/internal/queue"
	"go.uber.org/zap"
)

// Pool manages collection of workers
type Pool struct {

	// Config
	concurrency int
	registry    *job.Registry
	queue       queue.Queue
	logger      *zap.Logger

	// Runtime state
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	workers []*Worker

	// Metrics
	mu             sync.RWMutex
	totalProcessed int64
	totalFailed    int64
	totalRetried   int64

	// Shutdown
	shutdownTimeout time.Duration
}

// PoolConfig holds configuration for the worker pool
type PoolConfig struct {
	Concurrency     int
	ShutdownTimeout time.Duration
	PollInterval    time.Duration
}

// PoolStats holds statistics about the worker pool
type PoolStats struct {
	TotalWorkers   int   `json:"total_workers"`
	ActiveWorkers  int   `json:"active_workers"`
	TotalProcessed int64 `json:"total_processed"`
	TotalFailed    int64 `json:"total_failed"`
	TotalRetried   int64 `json:"total_retried"`
}

// NewPool creates a new worker pool
func NewPool(config PoolConfig, queue queue.Queue, registry *job.Registry, logger *zap.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		concurrency:     config.Concurrency,
		registry:        registry,
		queue:           queue,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		workers:         make([]*Worker, config.Concurrency),
		shutdownTimeout: config.ShutdownTimeout,
	}
}

func (p *Pool) Start() error {
	p.logger.Info("Starting worker pool", zap.Int("concurrency", p.concurrency))

	// Start workers
	for i := 0; i < p.concurrency; i++ {
		workerConfig := WorkerConfig{
			ID:           fmt.Sprintf("worker-%d", i+1),
			PollInterval: time.Second,
		}

		worker := NewWorker(workerConfig, p.queue, p.registry, p.logger)
		p.workers[i] = worker

		// Start worker in goroutine
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()

			if err := w.Start(p.ctx); err != nil {
				p.logger.Error("Worker stopped with error",
					zap.String("worker_id", w.config.ID),
					zap.Error(err),
				)
			}
		}(worker)
	}

	// Start metrics collection
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.collectMetrics()
	}()

	p.logger.Info("Worker pool started successfully")
	return nil

}

func (p *Pool) Stop() error {
	p.logger.Info("Stopping worker pool", zap.Duration("timeout", p.shutdownTimeout))

	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker pool stopped gracefully")
		return nil
	case <-time.After(p.shutdownTimeout):
		p.logger.Warn("Worker pool shutdown timeout exceeded")
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

func (p *Pool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Counting active workers
	activeWorkers := 0
	for _,worker := range p.workers {
		if worker.IsActive() {
			activeWorkers++
		}
	}
	return PoolStats{
		TotalWorkers:   p.concurrency,
		ActiveWorkers:  activeWorkers,
		TotalProcessed: p.totalProcessed,
		TotalFailed:    p.totalFailed,
		TotalRetried:   p.totalRetried,
	}
}

// collectMetrics periodically collects metrics from workers
func (p *Pool) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.updateMetrics()
		}
	}
}

// updateMetrics aggregates metrics from all workers
func (p *Pool) updateMetrics() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	var totalProcessed, totalFailed, totalRetried int64
	
	for _, worker := range p.workers {
		stats := worker.GetStats()
		totalProcessed += stats.JobsProcessed
		totalFailed += stats.JobsFailed
		totalRetried += stats.JobsRetried
	}
	
	p.totalProcessed = totalProcessed
	p.totalFailed = totalFailed
	p.totalRetried = totalRetried
	
	// Log metrics periodically
	p.logger.Info("Worker pool metrics",
		zap.Int64("processed", totalProcessed),
		zap.Int64("failed", totalFailed),
		zap.Int64("retried", totalRetried),
		zap.Int("active_workers", p.getActiveWorkerCount()),
	)
}

func (p *Pool) getActiveWorkerCount() int {
	count := 0
	for _, worker := range p.workers {
		if worker.IsActive() {
			count++
		}
	}
	return count
}