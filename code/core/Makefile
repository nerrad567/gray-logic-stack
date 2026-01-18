# Gray Logic Core - Makefile
# ===========================
#
# This Makefile provides common development tasks for Gray Logic Core.
# Run `make help` to see available commands.
#
# Philosophy: Simple, readable, and maintainable for 10+ years.
# We avoid complex Make tricks in favor of explicit, understandable rules.

# ============================================================================
# VARIABLES
# ============================================================================

# Application name (the binary we produce)
APP_NAME := graylogic

# Go build settings
GO := go
GOFLAGS := -v

# Version information (injected at build time)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags - inject version info into the binary
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Output directory
BUILD_DIR := build

# Source directories
CMD_DIR := ./cmd/graylogic
INTERNAL_DIR := ./internal/...
PKG_DIR := ./pkg/...

# ============================================================================
# DEFAULT TARGET
# ============================================================================

# The first target is the default - show help
.DEFAULT_GOAL := help

# ============================================================================
# PHONY TARGETS
# ============================================================================
# Phony targets are not files - they're just command names.
# Declaring them prevents Make from confusing them with filenames.

.PHONY: help build run clean test test-coverage lint fmt vet tidy vendor check all dev

# ============================================================================
# HELP
# ============================================================================

## help: Show this help message
help:
	@echo "Gray Logic Core - Build Commands"
	@echo "================================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

# ============================================================================
# BUILD TARGETS
# ============================================================================

## build: Build the application binary
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

## build-release: Build optimized release binary (smaller, no debug info)
build-release:
	@echo "Building release $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)"

## run: Build and run the application
run: build
	@echo "Running $(APP_NAME)..."
	$(BUILD_DIR)/$(APP_NAME)

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# ============================================================================
# TEST TARGETS
# ============================================================================

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## test-short: Run tests without verbose output
test-short:
	$(GO) test ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-integration: Run integration tests (requires test environment)
test-integration:
	@echo "Running integration tests..."
	$(GO) test -tags=integration -v ./tests/integration/...

# ============================================================================
# CODE QUALITY TARGETS
# ============================================================================

## lint: Run golangci-lint (must be installed separately)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

## fmt: Format all Go code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

## vet: Run go vet (finds subtle bugs)
vet:
	@echo "Running vet..."
	$(GO) vet ./...

## tidy: Clean up go.mod and go.sum
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy

## vendor: Download dependencies to vendor/ directory
vendor:
	@echo "Vendoring dependencies..."
	$(GO) mod vendor
	@echo "Vendor complete"

# ============================================================================
# COMBINED TARGETS
# ============================================================================

## check: Run all code quality checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

## all: Clean, check, and build release
all: clean check build-release
	@echo "Full build complete!"

## dev: Quick development build and run
dev: fmt vet build run

# ============================================================================
# DATABASE TARGETS (for future use)
# ============================================================================

## migrate-up: Run database migrations
migrate-up:
	@echo "Running migrations..."
	@echo "TODO: Implement migration runner"

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back migration..."
	@echo "TODO: Implement migration runner"

# ============================================================================
# DOCKER TARGETS (for future use)
# ============================================================================

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@echo "TODO: Implement Docker build"

## docker-up: Start development environment (Mosquitto, InfluxDB)
docker-up:
	@echo "Starting development environment..."
	@echo "TODO: Implement docker-compose up"

## docker-down: Stop development environment
docker-down:
	@echo "Stopping development environment..."
	@echo "TODO: Implement docker-compose down"
