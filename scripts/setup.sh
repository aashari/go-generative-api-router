#!/bin/bash
set -e

echo "Setting up development environment..."

# Download dependencies
echo "Downloading Go dependencies..."
go mod download
go mod tidy

# Install development tools
echo "Installing development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/swaggo/swag/cmd/swag@latest

# Setup configuration
if [ ! -f configs/credentials.json ]; then
    echo "Creating credentials.json from example..."
    cp configs/credentials.json.example configs/credentials.json
    echo "⚠️  Please edit configs/credentials.json with your API keys"
fi

# Create necessary directories
mkdir -p build logs

echo "✅ Setup complete!" 