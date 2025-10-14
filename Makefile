.PHONY: help build clean test fmt lint vet check tidy

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

# Format code
fmt: ## Format Go code using gofmt
	@echo "Formatting code..."
	@gofmt -s -w .

# Lint code
lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Run go vet
vet: ## Run go vet for correctness checks
	@echo "Running go vet..."
	@go vet ./...

# Run tests
test: ## Run tests with race detection
	@echo "Running tests..."
	@go test -race -v ./...

# Tidy dependencies
tidy: ## Sync go.mod and go.sum
	@echo "Tidying dependencies..."
	@go mod tidy

# Quality checks
check: fmt vet ## Run formatting and correctness checks
	@echo "All checks passed!"

# Development workflow
dev: clean tidy fmt vet build ## Full development workflow: clean, tidy, format, check, build
	@echo "Development build complete!"
