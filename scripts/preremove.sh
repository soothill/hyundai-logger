#!/bin/bash
# Pre-removal script for hyundai-logger package

set -e

# Stop and disable service if systemd is available
if command -v systemctl > /dev/null 2>&1; then
    if systemctl is-active --quiet hyundai-logger; then
        echo "Stopping hyundai-logger service..."
        systemctl stop hyundai-logger
    fi

    if systemctl is-enabled --quiet hyundai-logger 2>/dev/null; then
        echo "Disabling hyundai-logger service..."
        systemctl disable hyundai-logger
    fi

    # Remove systemd service file
    if [ -f /etc/systemd/system/hyundai-logger.service ]; then
        rm -f /etc/systemd/system/hyundai-logger.service
        systemctl daemon-reload
    fi
fi

echo "Hyundai Logger service stopped and disabled"
echo "Data and logs have been preserved in:"
echo "  - /var/log/hyundai-logger"
echo "  - /var/lib/hyundai-logger"
echo "  - /etc/hyundai-logger"
echo ""
echo "To completely remove all data:"
echo "  rm -rf /var/log/hyundai-logger /var/lib/hyundai-logger /etc/hyundai-logger"

exit 0
