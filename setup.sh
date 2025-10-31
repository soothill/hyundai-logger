#!/bin/bash
set -e

echo "================================"
echo "Hyundai Logger Setup Script"
echo "================================"
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

echo "Go version:"
go version
echo

# Check if PostgreSQL is available
if ! command -v psql &> /dev/null; then
    echo "Warning: psql command not found. Make sure PostgreSQL with TimescaleDB is installed."
    echo "Installation instructions:"
    echo "  Ubuntu/Debian: sudo apt-get install postgresql timescaledb-postgresql-14"
    echo "  macOS: brew install postgresql timescaledb"
    echo
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "Creating .env file from template..."
    cp .env.example .env
    echo "Please edit .env with your credentials:"
    echo "  nano .env"
    echo
else
    echo ".env file already exists"
    echo
fi

# Install Go dependencies
echo "Installing Go dependencies..."
go mod download
go mod tidy
echo "Dependencies installed successfully"
echo

# Build the application
echo "Building application..."
go build -o hyundai-logger cmd/hyundai-logger/main.go
echo "Build successful: ./hyundai-logger"
echo

echo "================================"
echo "Setup Complete!"
echo "================================"
echo
echo "Next steps:"
echo
echo "1. Configure your credentials:"
echo "   Edit .env or config.yaml with your Hyundai Bluelink credentials"
echo
echo "2. Set up PostgreSQL with TimescaleDB:"
echo "   sudo -u postgres psql"
echo "   Then run the SQL commands in the README"
echo
echo "3. Initialize the database:"
echo "   ./hyundai-logger -init-db"
echo
echo "4. Start the logger:"
echo "   ./hyundai-logger"
echo
echo "Or use Docker:"
echo "   docker-compose up -d"
echo
