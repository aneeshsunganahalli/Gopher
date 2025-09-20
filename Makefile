

# Build variables
BINARY_SERVER=bin/server
BINARY_WORKER=bin/worker
BINARY_CLI=bin/cli

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOMOD=$(GOCMD) mod
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOCLEAN=$(GOCMD) clean

# Docker variables
DOCKER_REGISTRY=your-registry
SERVER_IMAGE=job-queue-server
WORKER_IMAGE=job-queue-worker

 Default target
.PHONY: all
all: clean deps test build

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf bin/
	rm -rf vendor/

# Download dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run tests
.PHONY: test
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
.PHONY: test-coverage
test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build all binaries
.PHONY: build
build: build-server build-worker

# Build server binary
.PHONY: build-server
build-server:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags="-w -s" \
		-o $(BINARY_SERVER) \
		./cmd/server

# Build worker binary  
.PHONY: build-worker
build-worker:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags="-w -s" \
		-o $(BINARY_WORKER) \
		./cmd/worker

# Run server locally
.PHONY: run-server
run-server:
	$(GOCMD) run ./cmd/server/main.go

# Run worker locally
.PHONY: run-worker
run-worker:
	$(GOCMD) run ./cmd/worker/main.go

# Start Redis for local development
.PHONY: redis
redis:
	docker run --rm -p 6379:6379 --name redis-job-queue redis:7-alpine

# Stop local Redis
.PHONY: redis-stop
redis-stop:
	docker stop redis-job-queue || true

# Start full development environment
.PHONY: dev-up
dev-up:
	docker-compose -f docker-compose.dev.yml up -d

# Stop development environment
.PHONY: dev-down
dev-down:
	docker-compose -f docker-compose.dev.yml down

# Build Docker images
.PHONY: docker-build
docker-build: docker-build-server docker-build-worker

# Build server Docker image
.PHONY: docker-build-server
docker-build-server:
	docker build -f deployments/docker/Dockerfile.server \
		-t $(DOCKER_REGISTRY)/$(SERVER_IMAGE):latest .

# Build worker Docker image
.PHONY: docker-build-worker
docker-build-worker:
	docker build -f deployments/docker/Dockerfile.worker \
		-t $(DOCKER_REGISTRY)/$(WORKER_IMAGE):latest .

# Push Docker images
.PHONY: docker-push
docker-push: docker-build
	docker push $(DOCKER_REGISTRY)/$(SERVER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(WORKER_IMAGE):latest

# Run linting
.PHONY: lint
lint:
	golangci-lint run ./...

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Run security checks
.PHONY: security
security:
	gosec ./...

# Generate mocks (if using mockery)
.PHONY: mocks
mocks:
	mockery --all --output=./tests/mocks

# Database migrations (for future phases)
.PHONY: migrate-up
migrate-up:
	@echo "Migrations will be added in Phase 2"

# Load test the system
.PHONY: load-test
load-test:
	@echo "Starting load test..."
	./scripts/load-test.sh

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, download deps, test, and build"
	@echo "  clean        - Remove build artifacts"
	@echo "  deps         - Download Go dependencies"
	@echo "  test         - Run all tests"
	@echo "  build        - Build all binaries"
	@echo "  run-server   - Run server locally"
	@echo "  run-worker   - Run worker locally"
	@echo "  redis        - Start Redis container for development"
	@echo "  dev-up       - Start full development environment"
	@echo "  dev-down     - Stop development environment"
	@echo "  docker-build - Build Docker images"
	@echo "  lint         - Run Go linter"
	@echo "  fmt          - Format Go code"
	@echo "  help         - Show this help message"