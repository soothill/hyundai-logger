#!/bin/bash
# Post-installation script for hyundai-logger package

set -e

# Create system user if it doesn't exist
if ! id -u hyundai-logger > /dev/null 2>&1; then
    useradd --system --no-create-home --shell /bin/false hyundai-logger
fi

# Create directories
mkdir -p /var/log/hyundai-logger
mkdir -p /var/lib/hyundai-logger

# Set ownership
chown -R hyundai-logger:hyundai-logger /var/log/hyundai-logger
chown -R hyundai-logger:hyundai-logger /var/lib/hyundai-logger

# Set permissions
chmod 755 /var/log/hyundai-logger
chmod 755 /var/lib/hyundai-logger

# Create systemd service file if systemd is available
if command -v systemctl > /dev/null 2>&1; then
    cat > /etc/systemd/system/hyundai-logger.service <<'EOF'
[Unit]
Description=Hyundai Logger - Vehicle Telemetry Data Collector
Documentation=https://github.com/soothill/hyundai-logger
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=hyundai-logger
Group=hyundai-logger
ExecStart=/usr/bin/hyundai-logger --config /etc/hyundai-logger/config.yaml
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=hyundai-logger

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/hyundai-logger /var/lib/hyundai-logger

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd daemon
    systemctl daemon-reload

    echo "Hyundai Logger installed successfully"
    echo "Configuration file: /etc/hyundai-logger/config.example.yaml"
    echo ""
    echo "To configure and start the service:"
    echo "  1. Copy config: cp /etc/hyundai-logger/config.example.yaml /etc/hyundai-logger/config.yaml"
    echo "  2. Edit config: nano /etc/hyundai-logger/config.yaml"
    echo "  3. Enable service: systemctl enable hyundai-logger"
    echo "  4. Start service: systemctl start hyundai-logger"
fi

exit 0
