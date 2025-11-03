#!/bin/bash

# Extraction script for Hyundai Logger updates

echo "Extracting Hyundai Logger updates..."

if [ -d "src" ]; then
    echo "Found source directory. Copying files..."
    cp -r src/* ./
    echo "✅ Files extracted successfully!"
else
    echo "❌ Source directory not found. Please extract manually from all-files-combined.txt"
fi

echo ""
echo "Next steps:"
echo "1. Install Python dependencies: pip3 install selenium requests"
echo "2. Get refresh token: python3 scripts/get_refresh_token.py <REGION> <BRAND>"
echo "3. Build: go build -o hyundai-logger cmd/hyundai-logger/main.go"
echo "4. Configure: Edit config.yaml with your settings"
echo "5. Run: ./hyundai-logger -config config.yaml"
