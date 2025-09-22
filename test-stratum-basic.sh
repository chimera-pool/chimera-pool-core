#!/bin/bash

# Simple test runner for Stratum server - basic tests only
echo "🧪 Testing Stratum Server Implementation (Basic Tests)..."

# Create a simple Go test runner using Docker
docker run --rm -v "$(pwd)":/app -w /app golang:1.21-alpine sh -c "
    echo 'Installing dependencies...'
    go mod download
    echo 'Running basic Stratum tests...'
    go test -v -run 'TestStratumMessageParsing|TestStratumResponseGeneration|TestE2EStratumProtocolCompliance' ./internal/stratum/...
"