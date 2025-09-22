package tracing

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Tracer provides OpenTelemetry tracing capabilities
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
	logger   *zap.Logger
}

// Config for the tracer
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Enabled        bool
}

// NewTracer creates and initializes a new Tracer
func NewTracer(cfg Config, logger *zap.Logger) (*Tracer, error) {
	if !cfg.Enabled {
		logger.Info("Distributed tracing is disabled")
		return &Tracer{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
			logger: logger,
		}, nil
	}

	logger.Info("Initializing distributed tracing",
		zap.String("service", cfg.ServiceName),
		zap.String("otlp_endpoint", cfg.OTLPEndpoint),
	)

	// Create OTLP exporter
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Set global TracerProvider
	otel.SetTracerProvider(provider)

	tracer := provider.Tracer(cfg.ServiceName)

	return &Tracer{
		provider: provider,
		tracer:   tracer,
		logger:   logger,
	}, nil
}

// Shutdown stops the tracer, flushing any remaining spans
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider == nil {
		return nil
	}

	t.logger.Info("Shutting down tracer")
	return t.provider.Shutdown(ctx)
}

// StartSpan starts a new span with the given name
func (t *Tracer) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, spanName)
}

// HTTPMiddleware returns middleware for tracing HTTP requests
func (t *Tracer) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract context from request headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Start a new span for the request
		ctx, span := t.tracer.Start(ctx, fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path))
		defer span.End()

		// Add basic span attributes
		span.SetAttributes(
			semconv.HTTPMethodKey.String(r.Method),
			semconv.HTTPURLKey.String(r.URL.String()),
			semconv.HTTPUserAgentKey.String(r.UserAgent()),
		)

		// Serve the request with the span context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Close shuts down the tracer
func (t *Tracer) Close() error {
	if t.provider == nil {
		return nil
	}

	return t.provider.Shutdown(context.Background())
}
