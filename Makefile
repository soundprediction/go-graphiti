# Go Predicato Makefile

.PHONY: build build-cli test clean fmt vet lint run-example run-server deps tidy

# Build the project
build:
	go generate ./cmd/main.go
	go build -tags system_ladybug ./...

# Build CLI binary
build-cli:
	go generate ./cmd/main.go
	go build -tags system_ladybug -o bin/predicato ./cmd/main.go

# Build CLI for multiple platforms
build-cli-all:
	go generate ./cmd/main.go
	GOOS=linux GOARCH=amd64 go build -tags system_ladybug -o bin/predicato-linux-amd64 ./cmd/main.go
	GOOS=darwin GOARCH=amd64 go build -tags system_ladybug -o bin/predicato-darwin-amd64 ./cmd/main.go
	GOOS=darwin GOARCH=arm64 go build -tags system_ladybug -o bin/predicato-darwin-arm64 ./cmd/main.go
	GOOS=windows GOARCH=amd64 go build -tags system_ladybug -o bin/predicato-windows-amd64.exe ./cmd/main.go

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

# Run server (requires environment variables)
run-server:
	go generate ./cmd/main.go
	go run -tags system_ladybug ./cmd/main.go server

# Run server with debug mode
run-server-debug:
	go generate ./cmd/main.go
	go run -tags system_ladybug ./cmd/main.go server --mode debug --log-level debug

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
	@echo "  build-cli    - Build CLI binary"
	@echo "  build-cli-all- Build CLI for multiple platforms"
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
	@echo "  run-server   - Run server"
	@echo "  run-server-debug - Run server with debug mode"
	@echo "  dev          - Development workflow (fmt, vet, test)"
	@echo "  ci           - CI workflow (deps, tidy, fmt, vet, test-race)"
	@echo "  install-tools- Install development tools"
	@echo "  check        - Run comprehensive checks"
	@echo "  help         - Show this help"