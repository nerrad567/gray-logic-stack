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

.PHONY: help build run clean test test-coverage lint fmt vet tidy vendor check all dev \
       build-panel build-panel-dev build-all \
       dev-services dev-services-down dev-run dev-run-quick \
       docker-build docker-up docker-down

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

## build-panel: Build Flutter web UI and copy to embed directory
build-panel:
	@echo "Building Flutter wall panel..."
	cd ../ui/wallpanel && flutter build web --release --base-href="/panel/"
	@echo "Copying web build to embed directory..."
	rm -rf internal/panel/web
	cp -r ../ui/wallpanel/build/web internal/panel/web
	@echo "Panel build complete (ready for go:embed)"

## build-panel-dev: Quick Flutter web build (skips icon tree shaking for speed)
build-panel-dev:
	@echo "Building Flutter wall panel (dev — fast)..."
	cd ../ui/wallpanel && flutter build web --release --base-href="/panel/" --no-tree-shake-icons
	@echo "Copying web build to embed directory..."
	rm -rf internal/panel/web
	cp -r ../ui/wallpanel/build/web internal/panel/web
	@echo "Panel dev build complete"

## build-all: Build panel + Go binary (single production binary)
build-all: build-panel build
	@echo "Built Go binary with embedded Flutter panel"

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
# DEV WORKFLOW TARGETS (default mode — native Go + Docker support services)
# ============================================================================

## dev-services: Start support services (mosquitto, knxsim, influxdb)
dev-services:
	docker compose -f ../../docker-compose.dev.yml up -d --remove-orphans

## dev-services-down: Stop support services
dev-services-down:
	docker compose -f ../../docker-compose.dev.yml down

## dev-run: Build and run with filesystem panel serving
dev-run: fmt vet build
	GRAYLOGIC_PANEL_DIR=./internal/panel/web $(BUILD_DIR)/$(APP_NAME)

## dev-run-quick: Fastest iteration — skip fmt/vet, just build + run
dev-run-quick: build
	GRAYLOGIC_PANEL_DIR=./internal/panel/web $(BUILD_DIR)/$(APP_NAME)

# ============================================================================
# DOCKER TARGETS (production-like — full containerised stack)
# ============================================================================

## docker-build: Build production Docker image
docker-build:
	docker build -t graylogic/core:latest ./

## docker-up: Start full stack (production-like)
docker-up:
	docker compose -f ../../docker-compose.prod.yml up -d --build

## docker-down: Stop full stack
docker-down:
	docker compose -f ../../docker-compose.prod.yml down
