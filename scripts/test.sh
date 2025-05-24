#!/bin/bash
set -e

echo "Running tests..."
go test -v ./...

if [ "$1" == "--coverage" ]; then
    echo "Generating coverage report..."
    go test -cover -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report generated: coverage.html"
fi 