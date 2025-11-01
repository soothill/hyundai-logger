# Copyright (c) 2025 Darren Soothill
# Email: darren [at] soothill [dot] com
# Licensed under the MIT License

.PHONY: build run init-db clean test install fmt lint build-all help docker-build docker-build-multiarch docker-push docker-deploy docker-stop docker-restart docker-logs docker-clean docker-status install-logrotate

# Build variables
BINARY_NAME=hyundai-logger
BUILD_DIR=.
CMD_DIR=cmd/hyundai-logger

# Container runtime detection (Docker or Podman)
CONTAINER_CMD := $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)
CONTAINER_NAME := $(shell command -v docker 2>/dev/null && echo "Docker" || (command -v podman 2>/dev/null && echo "Podman" || echo ""))

# Detect number of CPU cores for parallel builds
NPROC := $(shell nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

# Docker Hub repository (set DOCKER_REPO environment variable to override)
DOCKER_REPO ?= soothill/hyundai-logger

# Compose command detection (docker-compose, docker compose, or podman-compose)
COMPOSE_CMD := $(shell \
	if command -v docker-compose >/dev/null 2>&1; then \
		echo "docker-compose"; \
	elif docker compose version >/dev/null 2>&1; then \
		echo "docker compose"; \
	elif command -v podman-compose >/dev/null 2>&1; then \
		echo "podman-compose"; \
	else \
		echo ""; \
	fi)

# Helper function to check if container runtime is available
define check_container_runtime
	@if [ -z "$(CONTAINER_CMD)" ]; then \
		echo ""; \
		echo "❌ Error: No container runtime found!"; \
		echo ""; \
		echo "This command requires either Docker or Podman to be installed."; \
		echo ""; \
		echo "Install Docker:"; \
		echo "  Ubuntu/Debian: sudo apt-get install docker.io docker-compose"; \
		echo "  Fedora:        sudo dnf install docker docker-compose"; \
		echo "  macOS:         brew install docker docker-compose"; \
		echo "  Or visit:      https://docs.docker.com/get-docker/"; \
		echo ""; \
		echo "Install Podman (alternative):"; \
		echo "  Ubuntu/Debian: sudo apt-get install podman podman-compose"; \
		echo "  Fedora:        sudo dnf install podman podman-compose"; \
		echo "  macOS:         brew install podman podman-compose"; \
		echo "  Or visit:      https://podman.io/getting-started/installation"; \
		echo ""; \
		exit 1; \
	fi
	@if [ -z "$(COMPOSE_CMD)" ]; then \
		echo ""; \
		echo "❌ Error: No compose tool found!"; \
		echo ""; \
		echo "$(CONTAINER_NAME) is installed, but compose is missing."; \
		echo ""; \
		if [ "$(CONTAINER_NAME)" = "Docker" ]; then \
			echo "Install docker-compose:"; \
			echo "  Ubuntu/Debian: sudo apt-get install docker-compose"; \
			echo "  Fedora:        sudo dnf install docker-compose"; \
			echo "  macOS:         brew install docker-compose"; \
			echo "  Or use:        docker compose (built-in with newer Docker)"; \
		else \
			echo "Install podman-compose:"; \
			echo "  Ubuntu/Debian: sudo apt-get install podman-compose"; \
			echo "  Fedora:        sudo dnf install podman-compose"; \
			echo "  macOS:         brew install podman-compose"; \
		fi; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Using $(CONTAINER_NAME) with compose"
endef

# Build the application
build:
	@echo "Building $(BINARY_NAME) using $(NPROC) CPU cores..."
	GOMAXPROCS=$(NPROC) go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
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
	@echo "Running tests using $(NPROC) CPU cores..."
	GOMAXPROCS=$(NPROC) go test -v -p $(NPROC) ./...

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
	@echo "Building for multiple platforms using $(NPROC) CPU cores..."
	GOMAXPROCS=$(NPROC) GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	GOMAXPROCS=$(NPROC) GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/main.go
	GOMAXPROCS=$(NPROC) GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/main.go
	GOMAXPROCS=$(NPROC) GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/main.go
	GOMAXPROCS=$(NPROC) GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/main.go
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
	@echo "Container Deployment (Docker/Podman):"
	@echo "  docker-build            - Build container image (current platform)"
	@echo "  docker-build-multiarch  - Build multi-arch image (amd64, arm64, arm/v7)"
	@echo "  docker-push             - Push multi-arch image to Docker Hub"
	@echo "  docker-push-docs        - Update Docker Hub repository overview (requires credentials)"
	@echo "  docker-deploy           - Deploy with compose (data persisted in volumes)"
	@echo "  docker-stop             - Stop all containers"
	@echo "  docker-restart          - Restart all containers"
	@echo "  docker-logs             - View container logs"
	@echo "  docker-status           - Show container status"
	@echo "  docker-clean            - Remove containers and images (keeps data volumes)"
	@echo ""
	@echo "  Note: Supports both Docker and Podman runtimes"
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
# Build Docker image (single architecture)
docker-build:
	$(call check_container_runtime)
	@echo "Building container image for current platform..."
	@echo "Using $(NPROC) CPU cores for parallel build"
	@if command -v docker >/dev/null 2>&1 && docker ps >/dev/null 2>&1; then \
		DOCKER_BUILDKIT=1 docker build \
			--build-arg GOMAXPROCS=$(NPROC) \
			--build-arg BUILDKIT_INLINE_CACHE=1 \
			-t hyundai-logger:latest \
			.; \
	elif command -v podman >/dev/null 2>&1; then \
		echo ""; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
		echo "ℹ️  Docker Permission Issue - Using Podman Instead"; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
		echo ""; \
		echo "Docker is installed but you don't have permission to use it."; \
		echo "Building with Podman instead (functionally identical)."; \
		echo ""; \
		echo "To fix Docker permissions for future builds:"; \
		echo ""; \
		echo "  1. Add yourself to the docker group:"; \
		echo "     sudo usermod -aG docker $$USER"; \
		echo ""; \
		echo "  2. Log out and back in (or run):"; \
		echo "     newgrp docker"; \
		echo ""; \
		echo "  3. Verify access:"; \
		echo "     docker ps"; \
		echo ""; \
		echo "For now, continuing with Podman..."; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
		echo ""; \
		ARCH=$$(uname -m); \
		case $$ARCH in \
			x86_64) PLATFORM="linux/amd64" ;; \
			aarch64) PLATFORM="linux/arm64" ;; \
			armv7l) PLATFORM="linux/arm/v7" ;; \
			*) PLATFORM="linux/$$ARCH" ;; \
		esac; \
		podman build \
			--jobs=$(NPROC) \
			--build-arg GOMAXPROCS=$(NPROC) \
			--build-arg BUILDPLATFORM=$$PLATFORM \
			-t hyundai-logger:latest \
			.; \
	else \
		echo ""; \
		echo "❌ Error: Docker found but no permissions!"; \
		echo ""; \
		echo "Add yourself to the docker group:"; \
		echo "  sudo usermod -aG docker $$USER"; \
		echo "  newgrp docker"; \
		echo ""; \
		echo "Or use Podman instead:"; \
		echo "  sudo apt-get install podman"; \
		echo ""; \
		exit 1; \
	fi
	@echo "✓ Container image built successfully"

# Build multi-architecture Docker image (amd64, arm64, arm/v7)
docker-build-multiarch:
	$(call check_container_runtime)
	@echo "Building multi-architecture container image..."
	@echo "Target platforms: linux/amd64, linux/arm64, linux/arm/v7"
	@echo "Using $(NPROC) CPU cores for parallel build"
	@echo ""
	@if command -v docker >/dev/null 2>&1 && docker buildx version >/dev/null 2>&1 && docker ps >/dev/null 2>&1; then \
		echo "Using Docker buildx for multi-arch build..."; \
		if ! docker buildx inspect multiarch-builder >/dev/null 2>&1; then \
			echo "Creating buildx builder instance..."; \
			docker buildx create --name multiarch-builder --use; \
		else \
			echo "Using existing buildx builder..."; \
			docker buildx use multiarch-builder; \
		fi; \
		DOCKER_BUILDKIT=1 docker buildx build \
			--platform linux/amd64,linux/arm64,linux/arm/v7 \
			--build-arg GOMAXPROCS=$(NPROC) \
			--build-arg BUILDKIT_INLINE_CACHE=1 \
			--tag hyundai-logger:latest \
			--load \
			.; \
	elif command -v docker >/dev/null 2>&1 && docker buildx version >/dev/null 2>&1 && ! docker ps >/dev/null 2>&1; then \
		echo "⚠️  Docker buildx detected but lacks permissions"; \
		echo ""; \
		echo "Docker is installed with buildx support, but you don't have permission to access the Docker daemon."; \
		echo ""; \
		echo "To fix this, add your user to the docker group:"; \
		echo "  sudo usermod -aG docker $$USER"; \
		echo "  newgrp docker"; \
		echo ""; \
		echo "Or run with sudo:"; \
		echo "  sudo make docker-build-multiarch"; \
		echo ""; \
		if command -v podman >/dev/null 2>&1; then \
			echo "Falling back to Podman (recommended - no sudo required)..."; \
			echo ""; \
			echo "Checking for QEMU emulation support..."; \
			if ! command -v qemu-aarch64-static >/dev/null 2>&1 && ! command -v qemu-arm-static >/dev/null 2>&1; then \
				echo ""; \
				echo "❌ Error: QEMU user-mode emulation not found!"; \
				echo ""; \
				echo "Multi-architecture builds require QEMU to emulate different CPU architectures."; \
				echo ""; \
				echo "Install QEMU:"; \
				echo "  Ubuntu/Debian: sudo apt-get install -y qemu-user-static binfmt-support"; \
				echo "  Fedora:        sudo dnf install -y qemu-user-static"; \
				echo "  Arch Linux:    sudo pacman -S qemu-user-static qemu-user-static-binfmt"; \
				echo ""; \
				echo "After installation, restart the binfmt service:"; \
				echo "  sudo systemctl restart systemd-binfmt.service"; \
				echo ""; \
				echo "Alternatively, build for your current platform only with:"; \
				echo "  make docker-build"; \
				echo ""; \
				exit 1; \
			fi; \
			echo "✓ QEMU emulation available"; \
			echo ""; \
			echo "Pre-pulling base images for all architectures..."; \
			echo "  Pulling golang:1.24-alpine for linux/amd64..."; \
			podman pull --platform linux/amd64 docker.io/library/golang:1.24-alpine; \
			echo "  Pulling alpine:latest for linux/amd64..."; \
			podman pull --platform linux/amd64 docker.io/library/alpine:latest; \
			echo "  Pulling golang:1.24-alpine for linux/arm64..."; \
			podman pull --platform linux/arm64 docker.io/library/golang:1.24-alpine; \
			echo "  Pulling alpine:latest for linux/arm64..."; \
			podman pull --platform linux/arm64 docker.io/library/alpine:latest; \
			echo "  Pulling golang:1.24-alpine for linux/arm/v7..."; \
			podman pull --platform linux/arm/v7 docker.io/library/golang:1.24-alpine; \
			echo "  Pulling alpine:latest for linux/arm/v7..."; \
			podman pull --platform linux/arm/v7 docker.io/library/alpine:latest; \
			echo "✓ All base images pre-pulled"; \
			echo ""; \
			echo "Cleaning up previous build artifacts..."; \
			podman rmi hyundai-logger:amd64 2>/dev/null || true; \
			podman rmi hyundai-logger:arm64 2>/dev/null || true; \
			podman rmi hyundai-logger:armv7 2>/dev/null || true; \
			podman manifest rm hyundai-logger:latest 2>/dev/null || true; \
			echo ""; \
			echo "Building for linux/amd64..."; \
			AMD64_ID=$$(podman build \
				--jobs=$(NPROC) \
				--platform linux/amd64 \
				--build-arg BUILDPLATFORM=linux/amd64 \
				--build-arg GOMAXPROCS=$(NPROC) \
				--tag hyundai-logger:amd64 \
				. | tail -1); \
			echo "Built amd64: $$AMD64_ID"; \
			echo ""; \
			echo "Building for linux/arm64..."; \
			ARM64_ID=$$(podman build \
				--jobs=$(NPROC) \
				--platform linux/arm64 \
				--build-arg BUILDPLATFORM=linux/amd64 \
				--build-arg GOMAXPROCS=$(NPROC) \
				--tag hyundai-logger:arm64 \
				. | tail -1); \
			echo "Built arm64: $$ARM64_ID"; \
			echo ""; \
			echo "Building for linux/arm/v7..."; \
			ARMV7_ID=$$(podman build \
				--jobs=$(NPROC) \
				--platform linux/arm/v7 \
				--build-arg BUILDPLATFORM=linux/amd64 \
				--build-arg GOMAXPROCS=$(NPROC) \
				--tag hyundai-logger:armv7 \
				. | tail -1); \
			echo "Built armv7: $$ARMV7_ID"; \
			echo ""; \
			echo "Creating manifest list..."; \
			podman manifest create hyundai-logger:latest; \
			podman manifest add --arch amd64 hyundai-logger:latest containers-storage:localhost/hyundai-logger:amd64; \
			podman manifest add --arch arm64 hyundai-logger:latest containers-storage:localhost/hyundai-logger:arm64; \
			podman manifest add --arch arm hyundai-logger:latest containers-storage:localhost/hyundai-logger:armv7; \
		else \
			echo "❌ Error: Neither Docker (with permissions) nor Podman is available"; \
			echo ""; \
			echo "Please either:"; \
			echo "  1. Fix Docker permissions (see above), or"; \
			echo "  2. Install Podman: sudo apt-get install -y podman"; \
			echo ""; \
			exit 1; \
		fi; \
	elif command -v podman >/dev/null 2>&1; then \
		echo "Using Podman for multi-arch build..."; \
		echo ""; \
		echo "Checking for QEMU emulation support..."; \
		if ! command -v qemu-aarch64-static >/dev/null 2>&1 && ! command -v qemu-arm-static >/dev/null 2>&1; then \
			echo ""; \
			echo "❌ Error: QEMU user-mode emulation not found!"; \
			echo ""; \
			echo "Multi-architecture builds require QEMU to emulate different CPU architectures."; \
			echo ""; \
			echo "Install QEMU:"; \
			echo "  Ubuntu/Debian: sudo apt-get install -y qemu-user-static binfmt-support"; \
			echo "  Fedora:        sudo dnf install -y qemu-user-static"; \
			echo "  Arch Linux:    sudo pacman -S qemu-user-static qemu-user-static-binfmt"; \
			echo ""; \
			echo "After installation, restart the binfmt service:"; \
			echo "  sudo systemctl restart systemd-binfmt.service"; \
			echo ""; \
			echo "Alternatively, build for your current platform only with:"; \
			echo "  make docker-build"; \
			echo ""; \
			exit 1; \
		fi; \
		echo "✓ QEMU emulation available"; \
		echo ""; \
		echo "Pre-pulling base images for all architectures..."; \
		echo "  Pulling golang:1.24-alpine for linux/amd64..."; \
		podman pull --platform linux/amd64 docker.io/library/golang:1.24-alpine; \
		echo "  Pulling alpine:latest for linux/amd64..."; \
		podman pull --platform linux/amd64 docker.io/library/alpine:latest; \
		echo "  Pulling golang:1.24-alpine for linux/arm64..."; \
		podman pull --platform linux/arm64 docker.io/library/golang:1.24-alpine; \
		echo "  Pulling alpine:latest for linux/arm64..."; \
		podman pull --platform linux/arm64 docker.io/library/alpine:latest; \
		echo "  Pulling golang:1.24-alpine for linux/arm/v7..."; \
		podman pull --platform linux/arm/v7 docker.io/library/golang:1.24-alpine; \
		echo "  Pulling alpine:latest for linux/arm/v7..."; \
		podman pull --platform linux/arm/v7 docker.io/library/alpine:latest; \
		echo "✓ All base images pre-pulled"; \
		echo ""; \
		echo "Cleaning up previous build artifacts..."; \
		podman rmi hyundai-logger:amd64 2>/dev/null || true; \
		podman rmi hyundai-logger:arm64 2>/dev/null || true; \
		podman rmi hyundai-logger:armv7 2>/dev/null || true; \
		podman manifest rm hyundai-logger:latest 2>/dev/null || true; \
		echo ""; \
		echo "Building for linux/amd64..."; \
		AMD64_ID=$$(podman build \
			--jobs=$(NPROC) \
			--platform linux/amd64 \
			--build-arg BUILDPLATFORM=linux/amd64 \
			--build-arg GOMAXPROCS=$(NPROC) \
			--tag hyundai-logger:amd64 \
			. | tail -1); \
		echo "Built amd64: $$AMD64_ID"; \
		echo ""; \
		echo "Building for linux/arm64..."; \
		ARM64_ID=$$(podman build \
			--jobs=$(NPROC) \
			--platform linux/arm64 \
			--build-arg BUILDPLATFORM=linux/amd64 \
			--build-arg GOMAXPROCS=$(NPROC) \
			--tag hyundai-logger:arm64 \
			. | tail -1); \
		echo "Built arm64: $$ARM64_ID"; \
		echo ""; \
		echo "Building for linux/arm/v7..."; \
		ARMV7_ID=$$(podman build \
			--jobs=$(NPROC) \
			--platform linux/arm/v7 \
			--build-arg BUILDPLATFORM=linux/amd64 \
			--build-arg GOMAXPROCS=$(NPROC) \
			--tag hyundai-logger:armv7 \
			. | tail -1); \
		echo "Built armv7: $$ARMV7_ID"; \
		echo ""; \
		echo "Creating manifest list..."; \
		podman manifest create hyundai-logger:latest; \
		podman manifest add --arch amd64 hyundai-logger:latest containers-storage:localhost/hyundai-logger:amd64; \
		podman manifest add --arch arm64 hyundai-logger:latest containers-storage:localhost/hyundai-logger:arm64; \
		podman manifest add --arch arm hyundai-logger:latest containers-storage:localhost/hyundai-logger:armv7; \
	else \
		echo "Error: No container runtime found"; \
		exit 1; \
	fi
	@echo ""
	@echo "✓ Multi-architecture image built successfully"
	@echo ""
	@echo "Supported architectures:"
	@echo "  - linux/amd64   (Intel/AMD 64-bit)"
	@echo "  - linux/arm64   (ARM 64-bit - Apple Silicon, Raspberry Pi 4+)"
	@echo "  - linux/arm/v7  (ARM 32-bit - Raspberry Pi 2/3)"

# Push multi-architecture image to Docker Hub
docker-push:
	$(call check_container_runtime)
	@echo "Pushing multi-architecture image to Docker Hub..."
	@echo "Repository: $(DOCKER_REPO)"
	@echo ""
	@echo "📝 Reminder: Update CHANGELOG.md before pushing!"
	@echo "   - Document all changes since last release"
	@echo "   - Follow semantic versioning (MAJOR.MINOR.PATCH)"
	@echo ""
	@echo "📚 Docker Hub Documentation:"
	@echo "   - README is auto-synced from GitHub"
	@echo "   - DOCKER_HUB.md provides Docker-specific docs"
	@echo "   - Update both files before pushing"
	@echo ""
	@if command -v docker >/dev/null 2>&1 && docker buildx version >/dev/null 2>&1 && docker ps >/dev/null 2>&1; then \
		echo "Using Docker to push..."; \
		echo "Checking Docker Hub login status..."; \
		if ! docker info 2>/dev/null | grep -q "Username:"; then \
			echo "⚠️  Not logged in to Docker Hub"; \
			echo "Please login with: docker login"; \
			exit 1; \
		fi; \
		echo "✓ Logged in to Docker Hub"; \
		echo ""; \
		echo "Tagging image as $(DOCKER_REPO):latest..."; \
		docker tag hyundai-logger:latest $(DOCKER_REPO):latest; \
		echo "Pushing $(DOCKER_REPO):latest..."; \
		docker push $(DOCKER_REPO):latest; \
	elif command -v podman >/dev/null 2>&1; then \
		echo "Using Podman to push..."; \
		echo "Checking Docker Hub login status..."; \
		if ! podman login --get-login docker.io >/dev/null 2>&1; then \
			echo "⚠️  Not logged in to Docker Hub"; \
			echo "Please login with: podman login docker.io"; \
			exit 1; \
		fi; \
		echo "✓ Logged in to Docker Hub"; \
		echo ""; \
		echo "Tagging manifest as $(DOCKER_REPO):latest..."; \
		podman tag localhost/hyundai-logger:latest $(DOCKER_REPO):latest; \
		echo "Pushing multi-architecture manifest to $(DOCKER_REPO):latest..."; \
		podman manifest push $(DOCKER_REPO):latest docker://$(DOCKER_REPO):latest; \
	else \
		echo "❌ Error: No container runtime found"; \
		exit 1; \
	fi
	@echo ""
	@echo "✓ Multi-architecture image pushed successfully to Docker Hub"
	@echo ""
	@echo "📦 Image Details:"
	@echo "  Repository: $(DOCKER_REPO):latest"
	@echo "  Architectures: linux/amd64, linux/arm64, linux/arm/v7"
	@echo "  View on Docker Hub: https://hub.docker.com/r/$(DOCKER_REPO)"
	@echo ""
	@echo "To pull this image on any platform:"
	@echo "  docker pull $(DOCKER_REPO):latest"
	@echo "  # or"
	@echo "  podman pull $(DOCKER_REPO):latest"
	@echo ""
	@echo "The correct architecture will be automatically selected based on your system."
	@echo ""
	@echo "📄 Next Steps:"
	@echo "  1. Verify image on Docker Hub: https://hub.docker.com/r/$(DOCKER_REPO)/tags"
	@echo "  2. Update CHANGELOG.md with release notes"
	@echo "  3. Create a Git tag for this release"
	@echo "  4. GitHub README.md will auto-sync to Docker Hub"

# Update Docker Hub repository overview and description
docker-push-docs:
	@echo "==============================================="
	@echo "  Update Docker Hub Repository Overview"
	@echo "==============================================="
	@echo ""
	@echo "This will update your Docker Hub repository with:"
	@echo "  - Full description from DOCKER_HUB.md (or README.md)"
	@echo "  - Short description"
	@echo ""
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "❌ Error: jq is not installed"; \
		echo ""; \
		echo "The Docker Hub API script requires jq for JSON parsing."; \
		echo ""; \
		echo "Install jq:"; \
		echo "  Ubuntu/Debian: sudo apt-get install -y jq"; \
		echo "  Fedora:        sudo dnf install -y jq"; \
		echo "  Arch Linux:    sudo pacman -S jq"; \
		echo "  macOS:         brew install jq"; \
		echo ""; \
		exit 1; \
	fi
	@if [ -z "$$DOCKERHUB_USERNAME" ] || [ -z "$$DOCKERHUB_TOKEN" ]; then \
		echo "Prerequisites:"; \
		echo "  1. Docker Hub account with access to $(DOCKER_REPO)"; \
		echo "  2. Docker Hub access token"; \
		echo ""; \
		echo "To create a Docker Hub access token:"; \
		echo "  1. Go to https://hub.docker.com/settings/security"; \
		echo "  2. Click 'New Access Token'"; \
		echo "  3. Give it a description (e.g., 'Documentation Updates')"; \
		echo "  4. Select 'Read & Write' permissions"; \
		echo "  5. Copy the token"; \
		echo ""; \
		echo "Set environment variables:"; \
		echo "  export DOCKERHUB_USERNAME=your-username"; \
		echo "  export DOCKERHUB_TOKEN=your-token"; \
		echo ""; \
		read -p "Press Enter to continue or Ctrl+C to cancel..." dummy; \
	else \
		echo "✓ Docker Hub credentials found"; \
		echo "  Repository: $(DOCKER_REPO)"; \
		echo ""; \
	fi
	@./scripts/update-dockerhub.sh

# Deploy with docker-compose (data and config externalized)
docker-deploy:
	$(call check_container_runtime)
	@echo "Deploying Hyundai Logger with $(CONTAINER_NAME) Compose..."
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
	$(COMPOSE_CMD) up -d --build
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
	$(call check_container_runtime)
	@echo "Stopping containers..."
	$(COMPOSE_CMD) down
	@echo "✓ Containers stopped (data volumes preserved)"

# Restart all containers
docker-restart:
	$(call check_container_runtime)
	@echo "Restarting containers..."
	$(COMPOSE_CMD) restart
	@echo "✓ Containers restarted"

# View container logs
docker-logs:
	$(call check_container_runtime)
	@echo "Showing logs (Ctrl+C to exit)..."
	@echo ""
	$(COMPOSE_CMD) logs -f hyundai-logger

# Show container status
docker-status:
	$(call check_container_runtime)
	@echo "Container status:"
	@echo ""
	$(COMPOSE_CMD) ps
	@echo ""
	@echo "Volumes:"
	$(CONTAINER_CMD) volume ls | grep hyundai

# Clean up containers and images (preserve data volumes)
docker-clean:
	$(call check_container_runtime)
	@echo "⚠️  This will remove containers and images but preserve data volumes."
	@read -p "Continue? [y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		echo "Stopping and removing containers..."; \
		$(COMPOSE_CMD) down; \
		echo "Removing container image..."; \
		$(CONTAINER_CMD) rmi hyundai-logger:latest 2>/dev/null || true; \
		echo ""; \
		echo "✓ Cleanup complete"; \
		echo ""; \
		echo "Data volumes preserved:"; \
		$(CONTAINER_CMD) volume ls | grep hyundai-logger || $(CONTAINER_CMD) volume ls | grep hyundai; \
		echo ""; \
		echo "To remove data volumes as well, run:"; \
		echo "  $(COMPOSE_CMD) down -v"; \
	else \
		echo "Cancelled"; \
	fi
