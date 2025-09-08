.PHONY: help build clean test test-unit test-integration test-race test-cover test-bench fmt lint vet check install run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build the application
build: ## Build the takl binary
	@echo "Building takl..."
	@go build -o takl .

# Clean build artifacts
clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -f takl
	@go clean

# Install dependencies
deps: ## Download and install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Testing targets
test: test-unit ## Run unit tests (fast, isolated)

test-unit: ## Run unit tests with -short flag
	@echo "Running unit tests..."
	@go test -short -v ./...

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@go test -short -race -v ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -tags=integration -v ./...

test-bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

test-cover: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -short -coverprofile=cover.out -covermode=atomic ./...
	@go tool cover -html=cover.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=cover.out | grep "total:" | awk '{print "Total coverage: " $$3}'

# Format code
fmt: ## Format Go code using gofmt
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		$(HOME)/go/bin/golangci-lint run; \
	fi

# Run go vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Quality checks
check: fmt vet lint test-race ## Run all quality checks
	@echo "All checks passed!"

# CI pipeline
ci-test: ## CI test pipeline with coverage gate
	@echo "Running CI test pipeline..."
	@go vet ./...
	@go test -race -coverprofile=cover.out ./...
	@go tool cover -func=cover.out | awk '/total:/ { if ($$3+0 < 80) exit 1 }'
	@echo "CI tests passed!"

test-all: deps fmt vet test-race test-integration test-cover ## Run comprehensive test suite

# Install the binary to $GOPATH/bin
install: ## Install takl binary to $GOPATH/bin
	@echo "Installing takl..."
	@go install .

# Run the application
run: ## Run the application with arguments: make run ARGS="command args"
	@go run . $(ARGS)

# Development workflow
dev: clean fmt vet lint test build ## Full development workflow

# Show Go environment
env: ## Show Go environment information
	@go env

# Show module information
mod-info: ## Show module information
	@go list -m all