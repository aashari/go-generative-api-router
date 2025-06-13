#!/bin/bash
set -e

BUILD_DIR="build"
BINARY_NAME="server"

echo "Building application..."
mkdir -p $BUILD_DIR

# Get version info
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build with version info
go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o $BUILD_DIR/$BINARY_NAME cmd/server/main.go

echo "Build complete: $BUILD_DIR/$BINARY_NAME (version: $VERSION)" 