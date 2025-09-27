# <span style="color: #FF6B35;">Gopher</span> - <span style="color: #4A90E2;">Distributed Task Queue for Go</span>

<div align="center">

![Gopher Logo](https://img.shields.io/badge/Gopher-Go%20Task%20Queue-4A90E2?style=for-the-badge\&logo=go\&logoColor=white)
![Version](https://img.shields.io/badge/version-1.0.0-6495ED?style=for-the-badge)
![Go](https://img.shields.io/badge/go-%3E%3D1.20-00ADD8?style=for-the-badge\&logo=go\&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%3E%3D6.0-DC382D?style=for-the-badge\&logo=redis\&logoColor=white)

<span style="color: #4A90E2; font-weight: bold;">A robust, distributed task queue in Go with Redis backend, designed for reliable asynchronous job execution</span>

<span style="color: #666; font-style: italic;">Supports prioritization, scheduling, retries, dead-letter queues, monitoring, and CLI management for production-grade workloads</span>

</div>

---

## ✨ Features

### 🎯 Core Functionality

> * ⚡ **HTTP API** for job submission and management
> * 🔝 **Priority queues**: high, normal, and low
> * ⏰ **Scheduled & recurring jobs**
> * 🔄 **Automatic retries** with configurable limits
> * 💀 **Dead Letter Queue** for failed jobs
> * 🐌 **Rate limiting** per job type
> * 📊 **Monitoring**: Prometheus metrics & health endpoints
> * 🔗 **Distributed tracing** via OpenTelemetry
> * 🛑 **Graceful shutdown** ensures no jobs are lost
> * 🛠️ **CLI tool** for queue management

### 🛠️ Developer Experience

> * 🐳 **Docker Compose support** for easy setup
> * ⚙️ **Configurable concurrency and worker settings**
> * 📦 **Modular architecture** for easy extension
> * 📘 **Example job handlers** for email, math, and image processing

---

## <span style="color: #9B59B6;">📄 Workflow & Architecture Documentation</span>

For a detailed step-by-step explanation of Gopher's workflow, job lifecycle, and real-world use cases, see [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## <span style="color: #9B59B6;">📦 Installation</span>

### <span style="color: #27AE60;">Clone & Build</span>

```bash
git clone https://github.com/aneeshsunganahalli/Gopher.git
cd Gopher
make build
```

### <span style="color: #3498DB;">Using Docker Compose</span>

```bash
docker-compose up -d
```

---

## <span style="color: #E74C3C;">🚀 Quick Start</span>

### <span style="color: #F39C12;">Start Redis Backend</span>

```bash
docker run --rm -p 6379:6379 --name redis-job-queue redis:7-alpine
```

### <span style="color: #8E44AD;">Start Gopher Server</span>

```bash
# Using Go
go run ./cmd/server/main.go

# Or built binary
./bin/server
```

### <span style="color: #2ECC71;">Start Worker Process</span>

```bash
go run ./cmd/worker/main.go
# Or binary
./bin/worker
```

### <span style="color: #3498DB;">Use CLI Tool</span>

```bash
# Submit a job
go run ./cmd/cli/cli.go submit -t email -p '{"to":"user@example.com","subject":"Hello","body":"This is a test"}'

# Check queue stats
go run ./cmd/cli/cli.go stats

# Retry failed jobs
go run ./cmd/cli/cli.go retry-all
```

---

## ⚙️ Configuration

Gopher uses environment variables for server, Redis, and worker settings:

```bash
# Server
SERVER_PORT=8080
SERVER_HOST=localhost
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=10s

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_TIMEOUT=5s

# Worker
WORKER_CONCURRENCY=5
WORKER_POLL_INTERVAL=1s
WORKER_MAX_RETRIES=3
WORKER_SHUTDOWN_TIMEOUT=30s

# Logging
LOG_LEVEL=info
LOG_FORMAT=console
```

---

## <span style="color: #4A90E2;">📁 Project Structure</span>

```
Gopher/
├── cmd/
│   ├── server/
│   ├── worker/
│   └── cli/
├── pkg/
│   ├── handlers/
│   ├── types/
│   └── queue/
├── configs/
│   └── config.example.env
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## <span style="color: #1ABC9C;">📬 Example Job Submission</span>

```bash
# Email job
curl -X POST http://localhost:8080/api/v1/jobs \
-H "Content-Type: application/json" \
-d '{
  "type": "email",
  "payload": {"to":"user@example.com","subject":"Hello","body":"Test email"},
  "priority": "high",
  "max_retries": 3
}'

# Scheduled job
curl -X POST http://localhost:8080/api/v1/jobs \
-H "Content-Type: application/json" \
-d '{
  "type": "report",
  "payload": {"report_type":"daily_summary"},
  "execute_at":"2025-10-01T10:00:00Z"
}'

# Recurring job
curl -X POST http://localhost:8080/api/v1/jobs \
-H "Content-Type: application/json" \
-d '{
  "type": "cleanup",
  "payload": {},
  "recurring": {"cron_expression":"0 0 * * *"}
}'
```

---

## <span style="color: #FF6B35;">💡 Best Practices</span>

> * 🏗️ **Idempotent jobs** to prevent duplicate processing
> * 📦 **Keep payloads small**; use external storage for large files
> * ⏱️ **Timeout handling** in job handlers
> * 🛑 **Graceful shutdown** of workers
> * ⚠️ **Error classification**: transient vs permanent
> * 📊 **Monitor queues** and setup alerts
> * 🐌 **Rate limiting** to avoid overloading services

---

## <span style="color: #34495E;">⚡ Requirements</span>

<div align="center">

![Go](https://img.shields.io/badge/Go-≥1.20-00ADD8?style=for-the-badge\&logo=go\&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-≥6.0-DC382D?style=for-the-badge\&logo=redis\&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-optional-2496ED?style=for-the-badge\&logo=docker\&logoColor=white)

</div>

---

## <span style="color: #2C3E50;">📄 License</span>

<div align="center">

<span style="color: #7F8C8D;">This project is licensed under the</span> <span style="color: #E74C3C;">**MIT License**</span><span style="color: #7F8C8D;">.</span>

</div>
