#!/bin/bash

# Simple test runner for Stratum server
echo "🧪 Testing Stratum Server Implementation..."

# Create a simple Go test runner using Docker
docker run --rm -v "$(pwd)":/app -w /app golang:1.21-alpine sh -c "
    echo 'Installing dependencies...'
    go mod download
    echo 'Running Stratum tests...'
    go test -v ./internal/stratum/...
"