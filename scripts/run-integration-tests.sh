#!/bin/bash
set -e

echo "🔗 Running Chimera Pool integration tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Wait for services to be ready
wait_for_service() {
    local service=$1
    local port=$2
    local timeout=${3:-30}
    
    echo "⏳ Waiting for $service on port $port..."
    
    for i in $(seq 1 $timeout); do
        if nc -z localhost $port 2>/dev/null; then
            echo -e "${GREEN}✅ $service is ready${NC}"
            return 0
        fi
        sleep 1
    done
    
    echo -e "${RED}❌ $service failed to start within ${timeout}s${NC}"
    return 1
}

# Run integration tests
run_integration_tests() {
    echo -e "\n${YELLOW}Running integration tests...${NC}"
    
    # Test database connectivity
    if [ -n "$DATABASE_URL" ]; then
        echo "Testing database connectivity..."
        go run -tags=integration tests/integration/database_test.go || {
            echo -e "${RED}❌ Database integration test failed${NC}"
            return 1
        }
    fi
    
    # Test Redis connectivity
    if [ -n "$REDIS_URL" ]; then
        echo "Testing Redis connectivity..."
        go run -tags=integration tests/integration/redis_test.go || {
            echo -e "${RED}❌ Redis integration test failed${NC}"
            return 1
        }
    fi
    
    # Test algorithm engine integration
    echo "Testing algorithm engine integration..."
    cargo test --workspace --features integration || {
        echo -e "${RED}❌ Algorithm engine integration test failed${NC}"
        return 1
    }
    
    # Test API endpoints (if running)
    if nc -z localhost 8080 2>/dev/null; then
        echo "Testing API endpoints..."
        go run -tags=integration tests/integration/api_test.go || {
            echo -e "${RED}❌ API integration test failed${NC}"
            return 1
        }
    fi
    
    echo -e "${GREEN}✅ All integration tests passed${NC}"
}

# Main execution
main() {
    # Check if we're in Docker environment
    if [ "$TEST_ENV" = "integration" ]; then
        echo "🐳 Running in Docker integration test environment"
        
        # Wait for dependent services
        wait_for_service "PostgreSQL" 5432
        wait_for_service "Redis" 6379
    fi
    
    # Run the integration tests
    run_integration_tests
    
    echo -e "\n${GREEN}🎉 Integration tests completed successfully!${NC}"
}

# Execute main function
main "$@"