.PHONY: all build clean run install test docker-build get-token init-db

# Variables
BINARY_NAME=hyundai-logger
MAIN_PATH=cmd/hyundai-logger/main.go
BUILD_DIR=build
DOCKER_IMAGE=hyundai-logger:latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

all: test build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	# Linux ARM (Raspberry Pi)
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 $(MAIN_PATH)
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	# Darwin ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml

run-verbose: build
	@echo "Running $(BINARY_NAME) with verbose output..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config config.yaml -verbose

install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f /usr/local/bin/$(BINARY_NAME)

# Token management
get-token:
	@echo "Starting token fetcher..."
	@echo "Usage: make get-token REGION=EU BRAND=hyundai"
	@if [ -z "$(REGION)" ]; then echo "Error: REGION not set"; exit 1; fi
	@if [ -z "$(BRAND)" ]; then echo "Error: BRAND not set"; exit 1; fi
	python3 scripts/get_refresh_token.py $(REGION) $(BRAND)

install-python-deps:
	@echo "Installing Python dependencies for token fetcher..."
	pip3 install selenium requests

# Database operations
init-db: build
	@echo "Initializing database schema..."
	./$(BUILD_DIR)/$(BINARY_NAME) -init-db

# Docker operations
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "Running Docker container..."
	docker run -d \
		--name hyundai-logger \
		-v $(PWD)/config.yaml:/app/config.yaml:ro \
		-v $(HOME)/.hyundai-logger:/root/.hyundai-logger \
		--restart unless-stopped \
		$(DOCKER_IMAGE)

docker-stop:
	@echo "Stopping Docker container..."
	docker stop hyundai-logger
	docker rm hyundai-logger

docker-logs:
	@echo "Showing Docker logs..."
	docker logs -f hyundai-logger

# Development helpers
fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

lint:
	@echo "Running golint..."
	golint ./...

# Help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make build-all      - Build for multiple platforms"
	@echo "  make run            - Build and run the application"
	@echo "  make run-verbose    - Run with verbose output"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make deps           - Download dependencies"
	@echo "  make install        - Install binary to /usr/local/bin"
	@echo "  make uninstall      - Remove installed binary"
	@echo ""
	@echo "Token Management:"
	@echo "  make get-token REGION=EU BRAND=hyundai - Get refresh token"
	@echo "  make install-python-deps                - Install Python dependencies"
	@echo ""
	@echo "Database:"
	@echo "  make init-db        - Initialize InfluxDB schema"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"
	@echo "  make docker-stop    - Stop Docker container"
	@echo "  make docker-logs    - Show Docker logs"
	@echo ""
	@echo "Development:"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run golint"

.DEFAULT_GOAL := help
