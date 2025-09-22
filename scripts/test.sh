#!/bin/bash
set -e

echo "🧪 Running Chimera Pool test suite..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run tests and track results
run_test_suite() {
    local name=$1
    local command=$2
    
    echo -e "\n${YELLOW}Running $name tests...${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if eval "$command"; then
        echo -e "${GREEN}✅ $name tests passed${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ $name tests failed${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# Ensure test environment is ready
setup_test_env() {
    echo "🔧 Setting up test environment..."
    
    # Start test database if not running
    if ! docker-compose -f deployments/docker/docker-compose.dev.yml ps postgres | grep -q "Up"; then
        echo "Starting test database..."
        docker-compose -f deployments/docker/docker-compose.dev.yml up -d postgres redis
        
        # Wait for database
        echo "⏳ Waiting for database..."
        until docker-compose -f deployments/docker/docker-compose.dev.yml exec -T postgres pg_isready -U chimera; do
            sleep 1
        done
    fi
    
    echo "✅ Test environment ready"
}

# Run Go tests
run_go_tests() {
    if [ -f "go.mod" ]; then
        # Unit tests
        run_test_suite "Go Unit" "go test -v -race -coverprofile=coverage.out ./..."
        
        # Integration tests (if they exist)
        if [ -d "tests/integration" ]; then
            run_test_suite "Go Integration" "go test -v -tags=integration ./tests/integration/..."
        fi
        
        # Generate coverage report
        if [ -f "coverage.out" ]; then
            echo "📊 Generating Go coverage report..."
            go tool cover -html=coverage.out -o coverage.html
            COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
            echo "📈 Go test coverage: $COVERAGE"
            
            # Check coverage threshold (90%)
            COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
            if (( $(echo "$COVERAGE_NUM >= 90" | bc -l) )); then
                echo -e "${GREEN}✅ Coverage meets threshold (90%)${NC}"
            else
                echo -e "${YELLOW}⚠️  Coverage below threshold: $COVERAGE < 90%${NC}"
            fi
        fi
    else
        echo "⏭️  No Go module found, skipping Go tests"
    fi
}

# Run Rust tests
run_rust_tests() {
    if [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Unit" "cargo test --workspace"
        run_test_suite "Rust Property" "cargo test --workspace --features proptest"
        
        # Run benchmarks if available
        if cargo test --workspace --benches --no-run &>/dev/null; then
            run_test_suite "Rust Benchmarks" "cargo test --workspace --benches"
        fi
    else
        echo "⏭️  No Cargo.toml found, skipping Rust tests"
    fi
}

# Run Node.js tests
run_node_tests() {
    if [ -f "package.json" ]; then
        run_test_suite "React Unit" "npm test -- --coverage --watchAll=false"
        
        # Check coverage
        if [ -f "coverage/lcov-report/index.html" ]; then
            echo "📊 React coverage report generated: coverage/lcov-report/index.html"
        fi
    else
        echo "⏭️  No package.json found, skipping Node.js tests"
    fi
}

# Run E2E tests
run_e2e_tests() {
    if [ -d "tests/e2e" ] && [ -n "$(ls -A tests/e2e)" ]; then
        echo -e "\n${YELLOW}Running E2E tests...${NC}"
        # E2E tests would go here when implemented
        echo "⏭️  E2E tests not yet implemented"
    else
        echo "⏭️  No E2E tests found"
    fi
}

# Security and quality checks
run_security_checks() {
    echo -e "\n${YELLOW}Running security checks...${NC}"
    
    # Go security check
    if command -v gosec &> /dev/null && [ -f "go.mod" ]; then
        run_test_suite "Go Security" "gosec ./..."
    else
        echo "⚠️  gosec not installed, installing..."
        if [ -f "go.mod" ]; then
            go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
            run_test_suite "Go Security" "gosec ./..."
        fi
    fi
    
    # Rust security check
    if command -v cargo-audit &> /dev/null && [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Security" "cargo audit"
    else
        echo "⚠️  cargo-audit not installed, installing..."
        if [ -f "Cargo.toml" ]; then
            cargo install cargo-audit
            run_test_suite "Rust Security" "cargo audit"
        fi
    fi
    
    # Node.js security check
    if [ -f "package.json" ]; then
        run_test_suite "Node Security" "npm audit --audit-level moderate"
    fi
}

# Quality gates validation
validate_quality_gates() {
    echo -e "\n${YELLOW}Validating quality gates...${NC}"
    
    local quality_passed=true
    
    # Check Go coverage
    if [ -f "coverage.out" ]; then
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "📊 Go test coverage: ${COVERAGE}%"
        
        if (( $(echo "${COVERAGE} < 90" | bc -l) )); then
            echo -e "${RED}❌ Go coverage below 90%: ${COVERAGE}%${NC}"
            quality_passed=false
        else
            echo -e "${GREEN}✅ Go coverage meets threshold: ${COVERAGE}% >= 90%${NC}"
        fi
    fi
    
    # Check React coverage
    if [ -f "coverage/lcov-report/index.html" ]; then
        echo -e "${GREEN}✅ React coverage report generated${NC}"
        # Additional coverage parsing could be added here
    fi
    
    # Check for security vulnerabilities
    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "${RED}❌ Security vulnerabilities found${NC}"
        quality_passed=false
    else
        echo -e "${GREEN}✅ No security vulnerabilities found${NC}"
    fi
    
    # Overall quality gate result
    if [ "$quality_passed" = true ]; then
        echo -e "\n${GREEN}🎉 All quality gates passed!${NC}"
        return 0
    else
        echo -e "\n${RED}❌ Quality gates failed${NC}"
        return 1
    fi
}

# Print test summary
print_summary() {
    echo -e "\n${YELLOW}=== Test Summary ===${NC}"
    echo "Total test suites: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

# Main execution
main() {
    setup_test_env
    run_go_tests
    run_rust_tests
    run_node_tests
    run_e2e_tests
    run_security_checks
    
    # Validate quality gates before final summary
    if ! validate_quality_gates; then
        echo -e "\n${RED}❌ Quality gates validation failed${NC}"
        exit 1
    fi
    
    print_summary
}

# Parse command line arguments
case "${1:-all}" in
    "go")
        setup_test_env
        run_go_tests
        ;;
    "rust")
        run_rust_tests
        ;;
    "node"|"react")
        run_node_tests
        ;;
    "e2e")
        setup_test_env
        run_e2e_tests
        ;;
    "security")
        run_security_checks
        ;;
    "all"|*)
        main
        ;;
esac
