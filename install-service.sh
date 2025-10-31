#!/bin/bash
#
# Installation script for Hyundai Logger as a systemd service
# Run with sudo: sudo ./install-service.sh
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="hyundai-logger"
SERVICE_USER="hyundai-logger"
SERVICE_GROUP="hyundai-logger"
INSTALL_DIR="/opt/hyundai-logger"
CONFIG_DIR="/etc/hyundai-logger"
LOG_DIR="/var/log/hyundai-logger"
BINARY_NAME="hyundai-logger"

echo "========================================"
echo "Hyundai Logger Service Installation"
echo "========================================"
echo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

# Check if binary exists
if [ ! -f "${BINARY_NAME}" ]; then
    echo -e "${YELLOW}Binary not found. Building...${NC}"
    if command -v go &> /dev/null; then
        go build -o ${BINARY_NAME} cmd/hyundai-logger/main.go
        echo -e "${GREEN}Build successful${NC}"
    else
        echo -e "${RED}Error: Go is not installed and binary not found${NC}"
        echo "Please install Go or build the binary manually"
        exit 1
    fi
fi

# Create service user and group
echo "Creating service user and group..."
if ! id -u ${SERVICE_USER} > /dev/null 2>&1; then
    useradd --system --no-create-home --shell /bin/false ${SERVICE_USER}
    echo -e "${GREEN}Created user: ${SERVICE_USER}${NC}"
else
    echo "User ${SERVICE_USER} already exists"
fi

# Create directories
echo "Creating directories..."
mkdir -p ${INSTALL_DIR}
mkdir -p ${CONFIG_DIR}
mkdir -p ${LOG_DIR}

# Copy binary
echo "Installing binary..."
cp ${BINARY_NAME} ${INSTALL_DIR}/
chmod 755 ${INSTALL_DIR}/${BINARY_NAME}
chown root:root ${INSTALL_DIR}/${BINARY_NAME}
echo -e "${GREEN}Binary installed to ${INSTALL_DIR}/${BINARY_NAME}${NC}"

# Copy configuration files
echo "Installing configuration files..."
if [ -f "config.yaml" ]; then
    if [ -f "${CONFIG_DIR}/config.yaml" ]; then
        echo -e "${YELLOW}Config file already exists, backing up...${NC}"
        cp ${CONFIG_DIR}/config.yaml ${CONFIG_DIR}/config.yaml.backup.$(date +%Y%m%d_%H%M%S)
    fi
    cp config.yaml ${CONFIG_DIR}/
    chown root:${SERVICE_GROUP} ${CONFIG_DIR}/config.yaml
    chmod 640 ${CONFIG_DIR}/config.yaml
    echo -e "${GREEN}Configuration installed to ${CONFIG_DIR}/config.yaml${NC}"
else
    echo -e "${YELLOW}Warning: config.yaml not found${NC}"
fi

# Copy .env file if exists
if [ -f ".env" ]; then
    if [ -f "${CONFIG_DIR}/.env" ]; then
        echo -e "${YELLOW}.env file already exists, backing up...${NC}"
        cp ${CONFIG_DIR}/.env ${CONFIG_DIR}/.env.backup.$(date +%Y%m%d_%H%M%S)
    fi
    cp .env ${CONFIG_DIR}/
    chown root:${SERVICE_GROUP} ${CONFIG_DIR}/.env
    chmod 640 ${CONFIG_DIR}/.env
    echo -e "${GREEN}Environment file installed to ${CONFIG_DIR}/.env${NC}"
else
    echo -e "${YELLOW}Warning: .env file not found${NC}"
    echo "You can copy it later to ${CONFIG_DIR}/.env"
fi

# Set directory permissions
chown root:root ${INSTALL_DIR}
chmod 755 ${INSTALL_DIR}

chown root:${SERVICE_GROUP} ${CONFIG_DIR}
chmod 750 ${CONFIG_DIR}

chown ${SERVICE_USER}:${SERVICE_GROUP} ${LOG_DIR}
chmod 755 ${LOG_DIR}

# Install systemd service
echo "Installing systemd service..."
cp hyundai-logger.service /etc/systemd/system/
chmod 644 /etc/systemd/system/hyundai-logger.service
echo -e "${GREEN}Service file installed${NC}"

# Install logrotate configuration
echo "Installing logrotate configuration..."
if [ -f "logrotate.conf" ]; then
    cp logrotate.conf /etc/logrotate.d/hyundai-logger
    chmod 644 /etc/logrotate.d/hyundai-logger
    echo -e "${GREEN}Logrotate configuration installed${NC}"
fi

# Reload systemd
echo "Reloading systemd daemon..."
systemctl daemon-reload

echo
echo "========================================"
echo -e "${GREEN}Installation Complete!${NC}"
echo "========================================"
echo
echo "Next steps:"
echo
echo "1. Edit the configuration file:"
echo "   sudo nano ${CONFIG_DIR}/config.yaml"
echo "   or"
echo "   sudo nano ${CONFIG_DIR}/.env"
echo
echo "2. Initialize the database (first time only):"
echo "   sudo -u ${SERVICE_USER} ${INSTALL_DIR}/${BINARY_NAME} -config ${CONFIG_DIR}/config.yaml -init-db"
echo
echo "3. Enable the service to start on boot:"
echo "   sudo systemctl enable ${SERVICE_NAME}"
echo
echo "4. Start the service:"
echo "   sudo systemctl start ${SERVICE_NAME}"
echo
echo "5. Check service status:"
echo "   sudo systemctl status ${SERVICE_NAME}"
echo
echo "6. View logs:"
echo "   sudo journalctl -u ${SERVICE_NAME} -f"
echo "   or"
echo "   sudo tail -f ${LOG_DIR}/${BINARY_NAME}.log"
echo
echo "Configuration files:"
echo "  - Service: /etc/systemd/system/${SERVICE_NAME}.service"
echo "  - Config:  ${CONFIG_DIR}/config.yaml"
echo "  - Env:     ${CONFIG_DIR}/.env"
echo "  - Logs:    ${LOG_DIR}/"
echo "  - Binary:  ${INSTALL_DIR}/${BINARY_NAME}"
echo
