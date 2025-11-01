# Build stage - use native platform for compilation (much faster!)
# BUILDPLATFORM is automatically set to the host platform (e.g., linux/amd64)
FROM --platform=$BUILDPLATFORM docker.io/golang:1.23-alpine AS builder

# Build arguments for parallel compilation
ARG GOMAXPROCS=4

# Docker/Podman automatically provides these when using --platform
ARG TARGETARCH
ARG TARGETOS
ARG BUILDPLATFORM

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with native cross-compilation
# The Go compiler runs natively (fast!) and cross-compiles for the target architecture
# TARGETARCH can be: amd64, arm64, arm (from --platform flag)
# For arm/v7, TARGETARCH=arm and we set GOARM=7
RUN if [ "$TARGETARCH" = "arm" ]; then \
        echo "Cross-compiling for linux/arm (armv7) on $BUILDPLATFORM"; \
        CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOARM=7 GOMAXPROCS=${GOMAXPROCS} \
        go build -a -installsuffix cgo -o hyundai-logger cmd/hyundai-logger/main.go; \
    else \
        echo "Cross-compiling for ${TARGETOS:-linux}/${TARGETARCH} on $BUILDPLATFORM"; \
        CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOMAXPROCS=${GOMAXPROCS} \
        go build -a -installsuffix cgo -o hyundai-logger cmd/hyundai-logger/main.go; \
    fi

# Final stage
FROM docker.io/alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/hyundai-logger .

# Create logs directory
RUN mkdir -p /app/logs

# Run as non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Run the logger
# Note: InfluxDB bucket is auto-created by docker-compose DOCKER_INFLUXDB_INIT_* variables
# For manual deployments, run with -init-db flag once: docker run ... hyundai-logger -init-db
CMD ["./hyundai-logger"]
