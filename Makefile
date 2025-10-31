# Copyright (c) 2025 Darren Soothill
# Email: darren [at] soothill [dot] com
# Licensed under the MIT License

.PHONY: build run init-db clean test install fmt lint build-all help docker-build docker-deploy docker-stop docker-restart docker-logs docker-clean docker-status install-logrotate

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
	@echo ""
	@echo "Build & Run:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  init-db       - Initialize the database schema"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run tests"
	@echo "  install       - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  build-all     - Build for multiple platforms"
	@echo ""
	@echo "Docker Deployment:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-deploy  - Deploy with docker-compose (data persisted in volumes)"
	@echo "  docker-stop    - Stop all containers"
	@echo "  docker-restart - Restart all containers"
	@echo "  docker-logs    - View container logs"
	@echo "  docker-status  - Show container status"
	@echo "  docker-clean   - Remove containers and images (keeps data volumes)"
	@echo ""
	@echo ""
	@echo "System Installation:"
	@echo "  install-logrotate - Install logrotate configuration (requires sudo)"
	@echo ""
	@echo "  help          - Show this help message"

# System installation targets
# Install logrotate configuration
install-logrotate:
	@echo "Installing logrotate configuration..."
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo ""; \
		echo "⚠️  This target requires root privileges."; \
		echo "Please run: sudo make install-logrotate"; \
		echo ""; \
		exit 1; \
	fi
	@echo "Copying logrotate.conf to /etc/logrotate.d/hyundai-logger..."
	install -m 0644 logrotate.conf /etc/logrotate.d/hyundai-logger
	@echo "Validating logrotate configuration..."
	@logrotate -d /etc/logrotate.d/hyundai-logger 2>&1 | head -n 10 || true
	@echo ""
	@echo "✓ Logrotate configuration installed successfully"
	@echo ""
	@echo "Configuration details:"
	@echo "  - Log path:     /var/log/hyundai-logger/*.log"
	@echo "  - Rotation:     Daily"
	@echo "  - Retention:    30 days"
	@echo "  - Max size:     100MB"
	@echo "  - Compression:  Enabled"
	@echo ""
	@echo "Note: Ensure the hyundai-logger user and log directory exist:"
	@echo "  sudo useradd -r -s /bin/false hyundai-logger"
	@echo "  sudo mkdir -p /var/log/hyundai-logger"
	@echo "  sudo chown hyundai-logger:hyundai-logger /var/log/hyundai-logger"

# Docker targets
# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t hyundai-logger:latest .
	@echo "Docker image built successfully"

# Deploy with docker-compose (data and config externalized)
docker-deploy:
	@echo "Deploying Hyundai Logger with Docker Compose..."
	@echo ""
	@if [ ! -f .env ]; then \
		echo "⚠️  WARNING: .env file not found!"; \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo ""; \
		echo "Please edit .env with your credentials before continuing."; \
		echo "Run 'nano .env' to edit the file."; \
		exit 1; \
	fi
	@echo "✓ Configuration file found"
	@echo ""
	@echo "Starting services..."
	docker-compose up -d --build
	@echo ""
	@echo "==============================================="
	@echo "  Deployment Complete!"
	@echo "==============================================="
	@echo ""
	@echo "Services running:"
	@echo "  - InfluxDB UI:  http://localhost:8086"
	@echo "  - Grafana:      http://localhost:3000 (admin/admin)"
	@echo ""
	@echo "Data volumes (persistent):"
	@echo "  - influxdb-data:   InfluxDB database"
	@echo "  - influxdb-config: InfluxDB configuration"
	@echo "  - grafana-data:    Grafana dashboards"
	@echo ""
	@echo "Configuration files (on host):"
	@echo "  - .env:         Credentials (mounted read-only)"
	@echo "  - config.yaml:  App configuration (mounted read-only)"
	@echo "  - ./logs:       Application logs"
	@echo ""
	@echo "Useful commands:"
	@echo "  make docker-logs     - View logs"
	@echo "  make docker-status   - Check status"
	@echo "  make docker-stop     - Stop services"
	@echo "  make docker-restart  - Restart services"

# Stop all containers
docker-stop:
	@echo "Stopping containers..."
	docker-compose down
	@echo "Containers stopped (data volumes preserved)"

# Restart all containers
docker-restart:
	@echo "Restarting containers..."
	docker-compose restart
	@echo "Containers restarted"

# View container logs
docker-logs:
	@echo "Showing logs (Ctrl+C to exit)..."
	@echo ""
	docker-compose logs -f hyundai-logger

# Show container status
docker-status:
	@echo "Container status:"
	@echo ""
	docker-compose ps
	@echo ""
	@echo "Docker volumes:"
	docker volume ls | grep hyundai

# Clean up containers and images (preserve data volumes)
docker-clean:
	@echo "⚠️  This will remove containers and images but preserve data volumes."
	@read -p "Continue? [y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		echo "Stopping and removing containers..."; \
		docker-compose down; \
		echo "Removing Docker image..."; \
		docker rmi hyundai-logger:latest 2>/dev/null || true; \
		echo ""; \
		echo "✓ Cleanup complete"; \
		echo ""; \
		echo "Data volumes preserved:"; \
		docker volume ls | grep hyundai-logger || docker volume ls | grep hyundai; \
		echo ""; \
		echo "To remove data volumes as well, run:"; \
		echo "  docker-compose down -v"; \
	else \
		echo "Cancelled"; \
	fi
