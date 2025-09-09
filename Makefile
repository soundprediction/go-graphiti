# Go Graphiti Makefile

.PHONY: build test clean fmt vet lint run-example deps tidy

# Build the project
build:
	go build ./...

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	go test -race ./...

# Clean build artifacts
clean:
	go clean ./...
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Install dependencies
deps:
	go mod download

# Tidy dependencies
tidy:
	go mod tidy

# Run basic example (requires environment variables)
run-example:
	cd examples/basic && go run main.go

# Development workflow
dev: fmt vet test

# CI workflow
ci: deps tidy fmt vet test-race

# Install development tools
install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run comprehensive checks
check: fmt vet lint test-race

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the project"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  test-race    - Run tests with race detection"
	@echo "  clean        - Clean build artifacts"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  deps         - Install dependencies"
	@echo "  tidy         - Tidy dependencies"
	@echo "  run-example  - Run basic example"
	@echo "  dev          - Development workflow (fmt, vet, test)"
	@echo "  ci           - CI workflow (deps, tidy, fmt, vet, test-race)"
	@echo "  install-tools- Install development tools"
	@echo "  check        - Run comprehensive checks"
	@echo "  help         - Show this help"