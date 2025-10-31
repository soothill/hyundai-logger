# Copyright (c) 2025 Darren Soothill
# Email: darren [at] soothill [dot] com
# Licensed under the MIT License

.PHONY: build run init-db clean test install

# Build variables
BINARY_NAME=hyundai-logger
BUILD_DIR=.
CMD_DIR=cmd/hyundai-logger

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	go run $(CMD_DIR)/main.go

# Initialize the database
init-db:
	@echo "Initializing database..."
	go run $(CMD_DIR)/main.go -init-db

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f *.log
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/main.go
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/main.go
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/main.go
	@echo "Multi-platform build complete"

# Help
help:
	@echo "==============================================="
	@echo "  Hyundai Logger - Makefile"
	@echo "  Author: Darren Soothill"
	@echo "  Email: darren [at] soothill [dot] com"
	@echo "==============================================="
	@echo ""
	@echo "Available targets:"
	@echo "  build      - Build the application"
	@echo "  run        - Run the application"
	@echo "  init-db    - Initialize the database schema"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  install    - Install dependencies"
	@echo "  fmt        - Format code"
	@echo "  lint       - Run linter"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  help       - Show this help message"
