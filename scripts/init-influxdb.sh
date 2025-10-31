#!/bin/bash
# Copyright (c) 2025 Darren Soothill
# Email: darren [at] soothill [dot] com
# Licensed under the MIT License
#
# InfluxDB v2 Initialization Script
# This script initializes InfluxDB with the necessary configuration

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "  Hyundai Logger - InfluxDB Initialization"
echo "  Author: Darren Soothill"
echo "  Email: darren [at] soothill [dot] com"
echo "================================================"
echo ""

# Load environment variables from .env if it exists
if [ -f .env ]; then
    echo -e "${GREEN}Loading configuration from .env file...${NC}"
    export $(cat .env | grep -v '^#' | xargs)
fi

# Default values
INFLUXDB_URL=${INFLUXDB_URL:-http://localhost:8086}
INFLUXDB_TOKEN=${INFLUXDB_TOKEN}
INFLUXDB_ORG=${INFLUXDB_ORG:-hyundai}
INFLUXDB_BUCKET=${INFLUXDB_BUCKET:-vehicle_data}

# Check if required variables are set
if [ -z "$INFLUXDB_TOKEN" ]; then
    echo -e "${RED}Error: INFLUXDB_TOKEN is not set${NC}"
    echo "Please set it in your .env file or as an environment variable"
    exit 1
fi

echo -e "${YELLOW}Configuration:${NC}"
echo "  URL: $INFLUXDB_URL"
echo "  Organization: $INFLUXDB_ORG"
echo "  Bucket: $INFLUXDB_BUCKET"
echo ""

# Check if influx CLI is available
if ! command -v influx &> /dev/null; then
    echo -e "${YELLOW}Warning: influx CLI not found. Attempting to use application's -init-db flag instead...${NC}"
    echo ""

    # Check if the binary exists
    if [ -f "./hyundai-logger" ]; then
        echo -e "${GREEN}Running: ./hyundai-logger -init-db${NC}"
        ./hyundai-logger -init-db
    elif [ -f "./cmd/hyundai-logger/main.go" ]; then
        echo -e "${GREEN}Running: go run ./cmd/hyundai-logger/main.go -init-db${NC}"
        go run ./cmd/hyundai-logger/main.go -init-db
    else
        echo -e "${RED}Error: Could not find hyundai-logger binary or source code${NC}"
        echo "Please build the application first with: make build"
        exit 1
    fi

    echo ""
    echo -e "${GREEN}✓ Database initialization complete!${NC}"
    exit 0
fi

# Check if InfluxDB is reachable
echo -e "${YELLOW}Checking InfluxDB connection...${NC}"
if ! curl -s "$INFLUXDB_URL/health" > /dev/null; then
    echo -e "${RED}Error: Cannot connect to InfluxDB at $INFLUXDB_URL${NC}"
    echo "Please ensure InfluxDB is running and the URL is correct"
    exit 1
fi
echo -e "${GREEN}✓ InfluxDB is reachable${NC}"
echo ""

# Check if organization exists
echo -e "${YELLOW}Checking organization '$INFLUXDB_ORG'...${NC}"
ORG_EXISTS=$(influx org list --host "$INFLUXDB_URL" --token "$INFLUXDB_TOKEN" --name "$INFLUXDB_ORG" --hide-headers 2>/dev/null | wc -l)

if [ "$ORG_EXISTS" -eq 0 ]; then
    echo -e "${YELLOW}Organization does not exist. Creating...${NC}"
    influx org create --host "$INFLUXDB_URL" --token "$INFLUXDB_TOKEN" --name "$INFLUXDB_ORG"
    echo -e "${GREEN}✓ Organization created${NC}"
else
    echo -e "${GREEN}✓ Organization already exists${NC}"
fi
echo ""

# Check if bucket exists
echo -e "${YELLOW}Checking bucket '$INFLUXDB_BUCKET'...${NC}"
BUCKET_EXISTS=$(influx bucket list --host "$INFLUXDB_URL" --token "$INFLUXDB_TOKEN" --org "$INFLUXDB_ORG" --name "$INFLUXDB_BUCKET" --hide-headers 2>/dev/null | wc -l)

if [ "$BUCKET_EXISTS" -eq 0 ]; then
    echo -e "${YELLOW}Bucket does not exist. Creating with 90-day retention...${NC}"
    influx bucket create \
        --host "$INFLUXDB_URL" \
        --token "$INFLUXDB_TOKEN" \
        --org "$INFLUXDB_ORG" \
        --name "$INFLUXDB_BUCKET" \
        --retention 2160h \
        --description "Hyundai vehicle telemetry data"
    echo -e "${GREEN}✓ Bucket created${NC}"
else
    echo -e "${GREEN}✓ Bucket already exists${NC}"

    # Optionally update retention policy
    echo -e "${YELLOW}Updating retention policy to 90 days...${NC}"
    BUCKET_ID=$(influx bucket list --host "$INFLUXDB_URL" --token "$INFLUXDB_TOKEN" --org "$INFLUXDB_ORG" --name "$INFLUXDB_BUCKET" --hide-headers 2>/dev/null | awk '{print $1}')
    influx bucket update \
        --host "$INFLUXDB_URL" \
        --token "$INFLUXDB_TOKEN" \
        --id "$BUCKET_ID" \
        --retention 2160h
    echo -e "${GREEN}✓ Retention policy updated${NC}"
fi
echo ""

echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}  ✓ InfluxDB initialization complete!${NC}"
echo -e "${GREEN}================================================${NC}"
echo ""
echo "You can now run the logger with:"
echo "  ./hyundai-logger"
echo "or"
echo "  make run"
