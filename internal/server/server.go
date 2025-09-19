package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aneeshsunganahalli/Gopher/internal/config"
	"github.com/aneeshsunganahalli/Gopher/internal/job"
	"github.com/aneeshsunganahalli/Gopher/internal/queue"
	"github.com/aneeshsunganahalli/Gopher/pkg/types"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Represents HTTP Server
type Server struct {
	config   *config.Config
	queue    queue.Queue
	registry *job.Registry
	logger   *zap.Logger
	router   *gin.Engine
	server   *http.Server
}

func NewServer(cfg *config.Config, queue queue.Queue, registry *job.Registry, logger *zap.Logger) *Server {
	s := &Server{
		config:   cfg,
		queue:    queue,
		registry: registry,
		logger:   logger,
	}

	s.setupRouter()
	s.setupServer()

	return s
}

func (s *Server) setupRouter() {

	if s.config.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	// Middleware
	s.router.Use(gin.Recovery())
	s.router.Use(s.loggingMiddleware())
	s.router.Use(s.corsMiddleware())

	s.router.GET("/health", s.healthHandler)

	v1 := s.router.Group("/api/v1")
	{
		v1.POST("/jobs", s.enqueueJobHandler)
		v1.GET("/jobs/types", s.listJobTypesHandler)
		v1.GET("/queue/stats", s.queueStatsHandler)
	}
}

func (s *Server) setupServer() {
	s.server = &http.Server{
		Addr:         s.config.Server.Address(),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.String("address", s.server.Addr),
	)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop the server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP Server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop server gracefully: %w", err)
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

func (s *Server) healthHandler(c *gin.Context) {

	if err := s.queue.Health(c.Request.Context()); err != nil {
		s.logger.Error("Health Check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

func (s *Server) enqueueJobHandler(c *gin.Context) {
	var request types.JobRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		s.logger.Error("Invalid job request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate job type is supported
	if _, err := s.registry.Get(request.Type); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Unsupported job type",
			"details": fmt.Sprintf("Job type '%s' is not registered", request.Type),
		})
		return
	}

	// Set default max retries if not specified
	maxRetries := s.config.Worker.MaxRetries
	if request.MaxRetries != nil {
		maxRetries = *request.MaxRetries
	}

	// Create job
	job := types.NewJob(request.Type, request.Payload, maxRetries)

	// Enqueue job
	if err := s.queue.Enqueue(c.Request.Context(), job); err != nil {
		s.logger.Error("Failed to enqueue job",
			zap.String("job_id", job.ID),
			zap.String("job_type", job.Type),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to enqueue job",
			"details": err.Error(),
		})
		return
	}

	s.logger.Info("Job enqueued successfully",
		zap.String("job_id", job.ID),
		zap.String("job_type", job.Type),
	)

	response := types.JobResponse{
		JobID:     job.ID,
		Status:    string(types.StatusPending),
		CreatedAt: job.CreatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// List job types handler
func (s *Server) listJobTypesHandler(c *gin.Context) {
	handlers := s.registry.ListHandlers()

	c.JSON(http.StatusOK, gin.H{
		"job_types": handlers,
	})
}

// Queue stats handler
func (s *Server) queueStatsHandler(c *gin.Context) {
	// Get queue stats if supported
	if redisQueue, ok := s.queue.(*queue.RedisQueue); ok {
		stats, err := redisQueue.GetStats(c.Request.Context())
		if err != nil {
			s.logger.Error("Failed to get queue stats", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get queue statistics",
			})
			return
		}

		c.JSON(http.StatusOK, stats)
		return
	}

	// Fallback to basic queue size
	size, err := s.queue.Size(c.Request.Context())
	if err != nil {
		s.logger.Error("Failed to get queue size", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get queue size",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"queue_size": size,
	})
}
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		duration := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Info("HTTP request",
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.Int("size", c.Writer.Size()),
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
