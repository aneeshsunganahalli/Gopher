# Gopher - Distributed Task Queue for Go

Gopher is a robust, distributed task queue built in Go with Redis as the backend. It provides a reliable way to execute asynchronous, distributed tasks with features like prioritization, scheduling, retries, and dead-letter queues.

## Features

- **Simple HTTP API**: RESTful interface for job submission and management
- **Priority Queues**: Support for high, normal, and low priority jobs
- **Scheduled Jobs**: Execute jobs at a future time or on a recurring schedule
- **Robust Error Handling**: Automatic retries with configurable limits
- **Dead Letter Queue**: Failed jobs go to a DLQ for inspection and potential retry
- **Rate Limiting**: Control job processing rates by job type
- **Monitoring**: Prometheus metrics and health endpoints
- **Distributed Tracing**: OpenTelemetry integration for request tracing
- **Graceful Shutdown**: Ensures jobs aren't lost during restart
- **CLI Tool**: For queue management and administration

## Architecture

Gopher consists of the following components:

1. **Server**: HTTP API for job submission and management
2. **Workers**: Processes that execute jobs from the queue
3. **Redis**: Backend for job storage and queue management
4. **CLI**: Command-line tool for queue administration

For a detailed technical overview of the system architecture, workflow, and how Gopher solves real-world problems, see [ARCHITECTURE.md](./ARCHITECTURE.md).

## Getting Started

### Prerequisites

- Go 1.20+
- Redis 6.0+
- Docker and Docker Compose (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/aneeshsunganahalli/Gopher.git
cd Gopher

# Build the binaries
make build

# Or use Docker Compose
docker-compose up -d
```

### Running Gopher

#### Step 1: Start Redis

Redis is required as the backend for the job queue. You can start it using Docker:

```bash
docker run --rm -p 6379:6379 --name redis-job-queue redis:7-alpine
```

#### Step 2: Start the Server

The server provides an HTTP API for job submission and management:

```bash
go run ./cmd/server/main.go
```

Or if you've built the binary:

```bash
./bin/server
```

#### Step 3: Start the Worker

The worker processes jobs from the queue:

```bash
go run ./cmd/worker/main.go
```

Or if you've built the binary:

```bash
./bin/worker
```

#### Step 4: Using the CLI

The CLI tool allows you to interact with the job queue system:

```bash
# Check queue statistics
go run ./cmd/cli/cli.go stats

# Submit a job
go run ./cmd/cli/cli.go submit -t email -p '{"to":"user@example.com","subject":"Test Email","body":"This is a test email"}'

# Submit a math job
go run ./cmd/cli/cli.go submit -t math -p '{"operation":"fibonacci","value":10}'

# Submit an image processing job
go run ./cmd/cli/cli.go submit -t image_resize -p '{"source":"path/to/image.jpg","width":800,"height":600}'

# Check system health
go run ./cmd/cli/cli.go health

# View failed jobs
go run ./cmd/cli/cli.go list-failed

# Retry failed jobs
go run ./cmd/cli/cli.go retry-all
```

### Configuration

Gopher uses environment variables for configuration:

```bash
# Server configuration
SERVER_PORT=8080
SERVER_HOST=localhost
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=10s

# Redis configuration
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_TIMEOUT=5s

# Worker configuration
WORKER_CONCURRENCY=5
WORKER_POLL_INTERVAL=1s
WORKER_MAX_RETRIES=3
WORKER_SHUTDOWN_TIMEOUT=30s

# Logging
LOG_LEVEL=info
LOG_FORMAT=console
```

## API Usage

### Submitting a Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Hello",
      "body": "This is a test email"
    },
    "max_retries": 3
  }'
```

### Submitting a High-Priority Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "URGENT",
      "body": "This is an urgent email"
    },
    "max_retries": 3,
    "priority": "high"
  }'
```

### Scheduling a Job for Later

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "payload": {
      "to": "user@example.com",
      "subject": "Scheduled Email",
      "body": "This email was scheduled"
    },
    "execute_at": "2023-01-01T10:00:00Z"
  }'
```

### Creating a Recurring Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "report",
    "payload": {
      "report_type": "daily_summary"
    },
    "recurring": {
      "cron_expression": "0 0 * * *"
    }
  }'
```

## CLI Usage

```bash
# Show queue statistics
go run ./cmd/cli/cli.go stats

# Submit a job
go run ./cmd/cli/cli.go submit -t email -p '{"to":"user@example.com","subject":"Hello","body":"This is a test email"}'

# List failed jobs
go run ./cmd/cli/cli.go list-failed

# Retry a failed job from the dead letter queue
go run ./cmd/cli/cli.go retry <job-id>

# Retry all failed jobs
go run ./cmd/cli/cli.go retry-all

# Check system health
go run ./cmd/cli/cli.go health

# Purge a queue
go run ./cmd/cli/cli.go purge <queue-name>
```

If you've built the binary:

```bash
# Show queue statistics
./bin/cli stats

# Submit a job
./bin/cli submit -t email -p '{"to":"user@example.com","subject":"Hello","body":"This is a test email"}'

# Submit a high-priority job
./bin/cli submit -t email -p '{"to":"user@example.com","subject":"URGENT","body":"Important message"}' --priority high

# Submit a job with custom retries
./bin/cli submit -t image_resize -p '{"source":"image.jpg","width":800,"height":600}' -r 5
```

## Creating Job Handlers

Job handlers are Go functions that process specific job types. Here's how to create a new handler:

```go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/aneeshsunganahalli/Gopher/pkg/types"
    "go.uber.org/zap"
)

// EmailPayload defines the structure for email job payloads
type EmailPayload struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

// EmailHandler handles email sending jobs
type EmailHandler struct {
    logger *zap.Logger
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(logger *zap.Logger) *EmailHandler {
    return &EmailHandler{logger: logger}
}

// Type returns the job type this handler processes
func (h *EmailHandler) Type() string {
    return "email"
}

// Description returns a human-readable description
func (h *EmailHandler) Description() string {
    return "Sends email messages"
}

// Handle processes the email job
func (h *EmailHandler) Handle(ctx context.Context, job *types.Job) error {
    // Parse payload
    var payload EmailPayload
    if err := json.Unmarshal(job.Payload, &payload); err != nil {
        return fmt.Errorf("invalid email payload: %w", err)
    }
    
    // Validate payload
    if payload.To == "" {
        return fmt.Errorf("recipient email is required")
    }
    
    h.logger.Info("Sending email",
        zap.String("to", payload.To),
        zap.String("subject", payload.Subject),
    )
    
    // Implement email sending logic here
    // ...
    
    h.logger.Info("Email sent successfully",
        zap.String("to", payload.To),
    )
    
    return nil
}
```

## Best Practices

1. **Job Idempotency**: Design jobs to be idempotent whenever possible
2. **Payload Size**: Keep job payloads small and use external storage for large data
3. **Timeout Handling**: Implement proper timeout handling in job handlers
4. **Graceful Shutdown**: Allow workers to finish current jobs during shutdown
5. **Error Classification**: Distinguish between transient and permanent errors
6. **Monitoring**: Set up alerts for queue size and error rates
7. **Rate Limiting**: Use rate limiting to prevent overwhelming external services


#### Redis Connection Errors

If you see Redis connection errors, make sure Redis is running and accessible:

```bash
# Check if Redis is running
docker ps | grep redis

# Test Redis connection
redis-cli ping
```

#### Worker Not Processing Jobs

If jobs remain in the queue but aren't being processed:

1. Check that workers are running: `ps aux | grep worker`
2. Check worker logs for errors: `./bin/worker --log-level debug`
3. Verify the job type has a registered handler

#### Rate Limiting Issues

If jobs are being rate-limited unexpectedly:

```bash
# Check current rate limits
go run ./cmd/cli/cli.go stats

# Adjust rate limits if needed
curl -X POST http://localhost:8080/api/v1/rate-limits/email -d '{"limit": 100, "burst": 10}'
```

#### CLI Command Syntax

If you're having trouble with CLI commands, check the help documentation:

```bash
go run ./cmd/cli/cli.go --help
go run ./cmd/cli/cli.go submit --help
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.