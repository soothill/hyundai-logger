# Build stage
FROM docker.io/golang:1.23-alpine AS builder

# Build arguments for parallel compilation
ARG GOMAXPROCS=4

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with parallel compilation
# GOMAXPROCS controls the number of CPUs Go can use during compilation
RUN CGO_ENABLED=0 GOOS=linux GOMAXPROCS=${GOMAXPROCS} go build -a -installsuffix cgo -o hyundai-logger cmd/hyundai-logger/main.go

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
