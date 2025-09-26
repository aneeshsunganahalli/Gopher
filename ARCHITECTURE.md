# Gopher - Distributed Task Queue System

## Overview

Made this file to explain the workflow of Gopher step by step

## Core Architecture

Gopher consists of three main components:

1. **Server**: A RESTful API service that accepts job submissions and provides status information
2. **Worker**: Processing daemons that execute jobs from the queue
3. **CLI**: A command-line interface for administration and job submission

These components communicate through Redis, which serves as the central message broker and job store.

## The Complete Workflow

### 1. Job Submission

When an application needs to perform a background task:

```go
// Client submits a job via HTTP API
POST /api/jobs
{
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome!",
    "body": "Thank you for signing up."
  },
  "priority": "high",
  "retries": 3
}

// Response
{
  "job_id": "e52fe7d2-8be1-4d9f-a27d-2f02cb16ad00",
  "status": "enqueued"
}
```

The server validates the request, generates a unique job ID, and enqueues it to Redis.

### 2. Queue Management

Redis maintains multiple queues based on priority:

- `queue:high` - Critical operations requiring immediate attention
- `queue:normal` - Standard background tasks
- `queue:low` - Non-urgent operations that can wait

For scheduled jobs, a separate sorted set tracks execution times:
```
ZADD scheduled_jobs 1695916800 job-data-json
```

### 3. Worker Processing

1. Workers poll Redis for available jobs, respecting priority ratios (e.g., 3:2:1 for high:normal:low)
2. When a job is dequeued, it's atomically moved to an "in-progress" state
3. The worker deserializes the job payload and routes it to the appropriate handler:

```go
// Example handler registration
registry.RegisterHandler("email", emailHandler)
registry.RegisterHandler("image_resize", imageResizeHandler)
registry.RegisterHandler("report_generation", reportHandler)
```

4. The handler processes the job, with automatic instrumentation for metrics and tracing
5. On success, the job is marked complete
6. On failure, the system either:
   - Retries the job with exponential backoff if retries remain
   - Moves the job to a Dead Letter Queue (DLQ) for later inspection

### 4. Observability and Monitoring

Throughout this process, Gopher collects extensive metrics:
- Queue depths by priority
- Processing time distributions
- Success/failure rates by job type
- Worker utilization statistics

These metrics are exposed via Prometheus endpoints and can be visualized in Grafana dashboards.

### 5. Job Lifecycle Management

The system provides tools for managing the entire job lifecycle:
- Scheduled jobs with cron-like patterns
- Rate limiting to prevent resource exhaustion
- Dead Letter Queue inspection and retry capabilities
- Historical job information for auditing and debugging

## Solving Real-World Problems

### Challenge: Web Application Responsiveness

**Problem:** In web applications, operations like sending emails, generating PDFs, or processing images can block the main thread, causing slow responses and poor user experience.

**Solution:** Gopher allows these operations to be offloaded to background workers:

```go
// Instead of processing directly in the web handler:
func SignupHandler(w http.ResponseWriter, r *http.Request) {
    // Process signup...
    
    // Enqueue welcome email instead of sending synchronously
    client.SubmitJob("email", EmailPayload{
        To: newUser.Email,
        Subject: "Welcome to our platform!",
        Template: "welcome",
        Data: map[string]interface{}{
            "username": newUser.Name,
        },
    })
    
    // Respond immediately to user
    w.WriteHeader(http.StatusCreated)
}
```

This pattern drastically improves response times and user experience.

### Challenge: Microservice Coordination

**Problem:** Microservices often need to trigger workflows that span multiple services without tight coupling.

**Solution:** Gopher serves as a coordination layer, allowing services to submit jobs that other services can process:

1. Service A submits a job to process a video
2. Worker picks up the job and processes it
3. On completion, another job is enqueued to notify the user
4. A different worker processes the notification job

This decoupled architecture improves system resilience and scalability.

### Challenge: Resource Management

**Problem:** Systems can be overwhelmed by spikes in workload, causing failures or degraded performance.

**Solution:** Gopher's priority queues, rate limiting, and configurable worker pools ensure resources are allocated efficiently:

- Critical operations get processed first
- Rate limits prevent any single job type from consuming all resources
- Worker pools scale based on available system resources

### Challenge: System Reliability

**Problem:** In distributed systems, failures are inevitable (network issues, crashes, bugs).

**Solution:** Gopher's retry mechanisms, Dead Letter Queue, and persistent job storage ensure that tasks eventually complete:

1. Failed jobs are automatically retried with exponential backoff
2. Jobs that exhaust retries are preserved in the DLQ
3. Operators can inspect failures and requeue them after fixing underlying issues

## Real-World Applications

- **E-commerce**: Processing orders, sending confirmation emails, updating inventory
- **Content platforms**: Transcoding media, generating thumbnails, updating search indexes
- **Financial systems**: Processing transactions, generating reports, sending notifications
- **IoT platforms**: Processing sensor data, triggering alerts, updating dashboards
- **SaaS applications**: User onboarding, report generation, data exports, batch operations

## Technical Implementation Details

Gopher leverages several key technologies and patterns:

- **Redis Atomic Operations**: Ensures jobs are processed exactly once despite multiple workers
- **Worker Pool Pattern**: Manages concurrency with configurable parallelism
- **Circuit Breaker Pattern**: Prevents cascading failures when downstream systems fail
- **Prometheus Metrics**: Provides real-time visibility into queue performance
- **OpenTelemetry**: Enables distributed tracing across job processing pipelines

## Conclusion

Gopher solves the universal challenge of background processing in distributed systems through a combination of robust queuing, intelligent worker management, comprehensive observability, and fault-tolerant design. Its modular architecture makes it adaptable to a wide range of applications while maintaining simplicity for developers.

By separating the concerns of job submission, storage, and execution, Gopher enables systems to scale more effectively, remain responsive under load, and recover gracefully from failures - addressing critical requirements for modern distributed applications.