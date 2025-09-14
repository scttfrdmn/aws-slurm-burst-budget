# Copyright 2025 Scott Friedman. All rights reserved.
# Use of this source code is governed by the MIT license
# that can be found in the LICENSE file.

.PHONY: build test clean install lint fmt vet coverage help

# Build configuration
BINARY_DIR := build
BINARY_NAME := asbb
SERVICE_NAME := budget-service
RECOVERY_NAME := recovery

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS := -ldflags "-X github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version.Version=$(VERSION) \
                    -X github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version.GitCommit=$(GIT_COMMIT) \
                    -X github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version.BuildTime=$(BUILD_TIME)"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Default target
all: build

## Build all binaries
build: build-cli build-service build-recovery

## Build CLI binary
build-cli:
	@echo "Building CLI binary..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/asbb

## Build service binary
build-service:
	@echo "Building service binary..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(SERVICE_NAME) ./cmd/budget-service

## Build recovery binary
build-recovery:
	@echo "Building recovery binary..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(RECOVERY_NAME) ./cmd/recovery

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

## Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -short ./...

## Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./test/integration/...

## Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## Lint code
lint:
	@echo "Linting code..."
	$(GOLINT) run

## Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## Check for security issues
security:
	@echo "Checking for security issues..."
	@which gosec > /dev/null || (echo "Installing gosec..." && $(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...

## Check code quality
quality: fmt vet lint security

## Install binaries
install: build
	@echo "Installing binaries..."
	@sudo mkdir -p /usr/local/bin
	@sudo cp $(BINARY_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo cp $(BINARY_DIR)/$(SERVICE_NAME) /usr/local/bin/
	@sudo cp $(BINARY_DIR)/$(RECOVERY_NAME) /usr/local/bin/
	@echo "Binaries installed to /usr/local/bin/"

## Install SLURM plugin
install-plugin:
	@echo "Installing SLURM plugin..."
	@cd plugins/slurm && $(MAKE) install

## Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html
	@cd plugins/slurm && $(MAKE) clean

## Run database migrations
migrate-up:
	@echo "Running database migrations..."
	@./$(BINARY_DIR)/$(BINARY_NAME) database migrate

## Rollback database migrations
migrate-down:
	@echo "Rolling back database migrations..."
	@./$(BINARY_DIR)/$(BINARY_NAME) database rollback

## Start development environment
dev:
	@echo "Starting development environment..."
	@docker-compose -f deployments/docker/docker-compose.yml up -d

## Stop development environment
dev-stop:
	@echo "Stopping development environment..."
	@docker-compose -f deployments/docker/docker-compose.yml down

## Generate mocks
mocks:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || $(GOGET) github.com/golang/mock/mockgen@latest
	@go generate ./...

## Generate documentation
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null || $(GOGET) golang.org/x/tools/cmd/godoc@latest
	@mkdir -p docs/generated
	@godoc -html > docs/generated/godoc.html

## Check environment
check-env:
	@echo "Checking environment..."
	@echo "Go version: $(shell $(GOCMD) version)"
	@echo "Git version: $(shell git --version 2>/dev/null || echo "Git not found")"
	@echo "Docker version: $(shell docker --version 2>/dev/null || echo "Docker not found")"
	@echo "PostgreSQL client: $(shell psql --version 2>/dev/null || echo "psql not found")"

## Pre-commit hook (run before committing)
pre-commit: fmt vet lint test-unit coverage
	@echo "Pre-commit checks completed successfully!"

## CI/CD pipeline
ci: deps quality test coverage
	@echo "CI pipeline completed successfully!"

## Release build
release: clean
	@echo "Building release binaries..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/asbb
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_DIR)/$(SERVICE_NAME)-linux-amd64 ./cmd/budget-service
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/asbb
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_DIR)/$(SERVICE_NAME)-darwin-amd64 ./cmd/budget-service

## Show help
help:
	@echo "Available commands:"
	@grep -E '^## .*' $(MAKEFILE_LIST) | sed 's/## /  /' | sort