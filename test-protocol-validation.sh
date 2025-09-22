#!/bin/bash

# Protocol validation test runner
echo "🧪 Testing Stratum Protocol Validation..."

# Create a simple Go test runner using Docker
docker run --rm -v "$(pwd)":/app -w /app golang:1.21-alpine sh -c "
    echo 'Installing dependencies...'
    go mod download
    echo 'Running protocol validation tests...'
    go test -v -run 'TestStratumV1ProtocolCompliance|TestMessageFormatCompliance|TestConcurrentConnectionHandling|TestResourceCleanup' ./internal/stratum/...
"