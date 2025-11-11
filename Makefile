# Copyright (c) 2025 Darren Soothill
# Licensed under the MIT License

.PHONY: help build test test-integration test-integration-coverage clean run docker-build docker-run lint fmt vet tidy install-tools

# Variables
BINARY_NAME=matter-data-logger
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
GOARCH?=$(shell go env GOARCH)
GOOS?=$(shell go env GOOS)

# Build output directory
BUILD_DIR=./build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	@GOOS=linux GOARCH=arm64 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64
	@GOOS=linux GOARCH=arm GOARM=7 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7
	@GOOS=darwin GOARCH=amd64 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	@GOOS=darwin GOARCH=arm64 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	@GOOS=windows GOARCH=amd64 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@GOOS=windows GOARCH=arm64 $(MAKE) build
	@mv $(BUILD_DIR)/$(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe
	@echo "Multi-platform build complete"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-integration: ## Run integration tests (requires Docker)
	@echo "Running integration tests..."
	@go test -v -tags=integration -timeout=15m ./storage

test-integration-coverage: ## Run integration tests with coverage (requires Docker)
	@echo "Running integration tests with coverage..."
	@go test -v -tags=integration -coverprofile=coverage-integration.out -timeout=15m ./storage
	@go tool cover -html=coverage-integration.out -o coverage-integration.html
	@echo "Integration test coverage report generated: coverage-integration.html"

lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run 'make install-tools'" && exit 1)
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed successfully"

run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "Docker image built: $(BINARY_NAME):$(VERSION)"

docker-build-multiplatform: ## Build multi-platform Docker images
	@echo "Building multi-platform Docker images..."
	@docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest .
	@echo "Multi-platform Docker images built"

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	@docker run --rm -it \
		--network host \
		-v $(PWD)/config.yaml:/app/config.yaml \
		$(BINARY_NAME):latest

docker-compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	@docker-compose down

docker-compose-logs: ## View docker-compose logs
	@docker-compose logs -f

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@echo "Dependencies downloaded"

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

ci: tidy deps check build ## Run CI pipeline

.DEFAULT_GOAL := help
