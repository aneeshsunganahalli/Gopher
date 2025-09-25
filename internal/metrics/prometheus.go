package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Metrics holds all Prometheus metrics for the job queue
type Metrics struct {
	// Job metrics
	JobsEnqueued      *prometheus.CounterVec
	JobsDequeued      *prometheus.CounterVec
	JobsProcessed     *prometheus.CounterVec
	JobsFailed        *prometheus.CounterVec
	JobsRetried       *prometheus.CounterVec
	JobProcessingTime *prometheus.HistogramVec

	// Queue metrics
	QueueSize          *prometheus.GaugeVec
	ScheduledQueueSize prometheus.Gauge
	DLQSize            prometheus.Gauge

	// Worker metrics
	WorkerCount       prometheus.Gauge
	ActiveWorkers     prometheus.Gauge
	WorkerUtilization prometheus.Gauge

	// System metrics
	APIRequestCount    *prometheus.CounterVec
	APIRequestDuration *prometheus.HistogramVec

	logger *zap.Logger
	server *http.Server
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(logger *zap.Logger) *Metrics {
	m := &Metrics{
		logger: logger,

		// Job metrics
		JobsEnqueued: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gopher_jobs_enqueued_total",
			Help: "Total number of jobs added to the queue",
		}, []string{"job_type", "priority"}),

		JobsDequeued: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gopher_jobs_dequeued_total",
			Help: "Total number of jobs removed from the queue",
		}, []string{"job_type", "priority"}),

		JobsProcessed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gopher_jobs_processed_total",
			Help: "Total number of jobs processed successfully",
		}, []string{"job_type"}),

		JobsFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gopher_jobs_failed_total",
			Help: "Total number of jobs that failed processing",
		}, []string{"job_type", "error_type"}),

		JobsRetried: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gopher_jobs_retried_total",
			Help: "Total number of jobs that were retried",
		}, []string{"job_type"}),

		JobProcessingTime: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gopher_job_processing_duration_seconds",
			Help:    "Time taken to process jobs",
			Buckets: prometheus.DefBuckets,
		}, []string{"job_type"}),

		// Queue metrics
		QueueSize: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gopher_queue_size",
			Help: "Current number of jobs in the queue",
		}, []string{"priority"}),

		ScheduledQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gopher_scheduled_queue_size",
			Help: "Current number of jobs in the scheduled queue",
		}),

		DLQSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gopher_dlq_size",
			Help: "Current number of jobs in the dead letter queue",
		}),

		// Worker metrics
		WorkerCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gopher_worker_count",
			Help: "Total number of workers in the pool",
		}),

		ActiveWorkers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gopher_active_workers",
			Help: "Number of workers currently processing jobs",
		}),

		WorkerUtilization: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gopher_worker_utilization",
			Help: "Percentage of workers currently active (0-100)",
		}),

		// API metrics
		APIRequestCount: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gopher_api_requests_total",
			Help: "Total number of API requests",
		}, []string{"method", "path", "status"}),

		APIRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gopher_api_request_duration_seconds",
			Help:    "Duration of API requests",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
	}

	logger.Info("Prometheus metrics initialized")
	return m
}

// StartServer starts the Prometheus metrics HTTP server
func (m *Metrics) StartServer(address string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	m.server = &http.Server{
		Addr:    address,
		Handler: mux,
	}

	m.logger.Info("Starting Prometheus metrics server", zap.String("address", address))
	return m.server.ListenAndServe()
}

// StopServer stops the Prometheus metrics HTTP server
func (m *Metrics) StopServer(ctx context.Context) error {
	m.logger.Info("Stopping Prometheus metrics server")
	return m.server.Shutdown(ctx)
}

// PrometheusMiddleware returns a middleware for collecting HTTP metrics
func (m *Metrics) PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status code
		rw := newResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := rw.statusCode

		m.APIRequestCount.WithLabelValues(r.Method, r.URL.Path, string(rune(status))).Inc()
		m.APIRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
