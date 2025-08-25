.PHONY: test test-all test-unit test-integration test-protocols test-coverage build clean

# Variables
TIMEOUT := 60s
VERBOSE := -v

# Test commands
test-all:
	@echo "Running all tests..."
	go test $(VERBOSE) -timeout $(TIMEOUT) ./...

test-unit:
	@echo "Running unit tests..."
	go test $(VERBOSE) -timeout 30s ./pkg/...

test-protocols:
	@echo "Running protocol tests..."
	go test $(VERBOSE) -timeout $(TIMEOUT) ./protocols/...

test-integration:
	@echo "Running integration tests..."
	go test $(VERBOSE) -timeout $(TIMEOUT) -tags=integration ./protocols/...

test-coverage:
	@echo "Running tests with coverage..."
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race:
	@echo "Running tests with race detection..."
	go test -race $(VERBOSE) -timeout $(TIMEOUT) ./...

test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Individual module tests
test-logger:
	go test $(VERBOSE) ./pkg/logger/...

test-adapter:
	go test $(VERBOSE) ./pkg/adapter/...

test-config:
	go test $(VERBOSE) ./pkg/config/...

test-scenario:
	go test $(VERBOSE) ./pkg/scenario/...

test-docker:
	go test $(VERBOSE) ./pkg/docker/...

test-modbus:
	go test $(VERBOSE) -timeout $(TIMEOUT) ./protocols/modbus/...

# Build commands
build:
	go build -o bin/icscommemulator ./cmd/icscommemulator

# Clean commands
clean:
	go clean ./...
	rm -rf bin/ coverage.out coverage.html

# Development commands
deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

# CI commands
ci-test: deps fmt vet test-race test-coverage

# Quick development test
dev-test:
	go test -short ./...
