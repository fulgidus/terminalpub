.PHONY: help build run dev test lint format migrate-up migrate-down docker-up docker-down clean install-deps

# Variables
BINARY_NAME=terminalpub
WORKER_NAME=terminalpub-worker
GO=go
GOFLAGS=-v
MAIN_PATH=./cmd/server
WORKER_PATH=./cmd/worker

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install development dependencies
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) install github.com/air-verse/air@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dependencies installed!"

build: ## Build the server binary
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: bin/$(BINARY_NAME)"

build-worker: ## Build the worker binary
	@echo "Building $(WORKER_NAME)..."
	$(GO) build $(GOFLAGS) -o bin/$(WORKER_NAME) $(WORKER_PATH)
	@echo "Build complete: bin/$(WORKER_NAME)"

build-all: build build-worker ## Build all binaries

run: build ## Build and run the server
	@echo "Starting $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

dev: ## Run server with auto-reload (requires air)
	@echo "Starting development server with hot reload..."
	@air -c .air.toml || (echo "air not found. Run 'make install-deps' first" && exit 1)

test: ## Run all tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-single: ## Run a single test (usage: make test-single TEST=TestName)
	@echo "Running test: $(TEST)"
	$(GO) test -v -run $(TEST) ./...

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./... || (echo "golangci-lint not found. Run 'make install-deps' first" && exit 1)

format: ## Format code with gofmt and goimports
	@echo "Formatting code..."
	@gofmt -s -w .
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	else \
		echo "goimports not found, skipping..."; \
	fi
	@echo "Code formatted!"

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	$(GO) run cmd/migrate/main.go up

migrate-down: ## Rollback database migrations
	@echo "Rolling back migrations..."
	$(GO) run cmd/migrate/main.go down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=migration_name)
	@echo "Creating migration: $(NAME)"
	@if [ -z "$(NAME)" ]; then echo "NAME is required. Usage: make migrate-create NAME=migration_name"; exit 1; fi
	@timestamp=$$(date +%Y%m%d%H%M%S); \
	touch migrations/$${timestamp}_$(NAME).up.sql migrations/$${timestamp}_$(NAME).down.sql; \
	echo "Created migrations/$${timestamp}_$(NAME).up.sql and migrations/$${timestamp}_$(NAME).down.sql"

docker-up: ## Start Docker services (PostgreSQL + Redis)
	@echo "Starting Docker services..."
	docker-compose up -d
	@echo "Services started! Waiting for databases to be ready..."
	@sleep 3

docker-down: ## Stop Docker services
	@echo "Stopping Docker services..."
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf tmp/
	@rm -f coverage.txt
	@echo "Clean complete!"

.DEFAULT_GOAL := help
