#!/bin/bash
set -e

echo "🧪 Chimera Pool - Comprehensive Test Suite"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
COVERAGE_THRESHOLD=90
PARALLEL_JOBS=4

# Test results tracking
declare -A test_results
declare -A test_durations

# Function to run a test suite and track results
run_test_suite() {
    local suite_name=$1
    local test_command=$2
    local start_time=$(date +%s)
    
    echo -e "\n${BLUE}🔄 Running $suite_name...${NC}"
    
    if eval "$test_command"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        test_results["$suite_name"]="PASS"
        test_durations["$suite_name"]=$duration
        echo -e "${GREEN}✅ $suite_name completed in ${duration}s${NC}"
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        test_results["$suite_name"]="FAIL"
        test_durations["$suite_name"]=$duration
        echo -e "${RED}❌ $suite_name failed after ${duration}s${NC}"
    fi
}

# Setup test environment
setup_environment() {
    echo -e "${YELLOW}🔧 Setting up test environment...${NC}"
    
    # Ensure Docker is running
    if ! docker info >/dev/null 2>&1; then
        echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
        exit 1
    fi
    
    # Start test services
    echo "Starting test services..."
    docker-compose -f deployments/docker/docker-compose.test.yml up -d postgres-test redis-test
    
    # Wait for services to be ready
    echo "⏳ Waiting for test services..."
    timeout 60 bash -c 'until docker-compose -f deployments/docker/docker-compose.test.yml exec -T postgres-test pg_isready -U chimera -d chimera_pool_test; do sleep 1; done'
    timeout 60 bash -c 'until docker-compose -f deployments/docker/docker-compose.test.yml exec -T redis-test redis-cli ping | grep PONG; do sleep 1; done'
    
    echo -e "${GREEN}✅ Test environment ready${NC}"
}

# Run unit tests
run_unit_tests() {
    echo -e "\n${YELLOW}📋 Running Unit Tests${NC}"
    
    # Go unit tests
    if [ -f "go.mod" ]; then
        run_test_suite "Go Unit Tests" "go test -v -race -short ./..."
    fi
    
    # Rust unit tests
    if [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Unit Tests" "cargo test --workspace --lib"
    fi
    
    # React unit tests
    if [ -f "package.json" ]; then
        run_test_suite "React Unit Tests" "npm test -- --watchAll=false --testPathIgnorePatterns=integration"
    fi
}

# Run integration tests
run_integration_tests() {
    echo -e "\n${YELLOW}🔗 Running Integration Tests${NC}"
    
    # Go integration tests
    if [ -f "go.mod" ]; then
        export DATABASE_URL="postgres://chimera:test_password@localhost:5433/chimera_pool_test?sslmode=disable"
        export REDIS_URL="redis://localhost:6380"
        run_test_suite "Go Integration Tests" "go test -v -tags=integration ./tests/integration/..."
    fi
    
    # Rust integration tests
    if [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Integration Tests" "cargo test --workspace --features integration"
    fi
    
    # End-to-end tests (if available)
    if [ -d "tests/e2e" ] && [ -n "$(ls -A tests/e2e)" ]; then
        run_test_suite "E2E Tests" "./scripts/run-integration-tests.sh"
    fi
}

# Run property-based tests
run_property_tests() {
    echo -e "\n${YELLOW}🎲 Running Property-Based Tests${NC}"
    
    if [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Property Tests" "cargo test --workspace --features proptest"
    fi
}

# Run benchmarks
run_benchmarks() {
    echo -e "\n${YELLOW}⚡ Running Benchmarks${NC}"
    
    if [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Benchmarks" "cargo bench --workspace"
    fi
    
    if [ -f "go.mod" ]; then
        run_test_suite "Go Benchmarks" "go test -bench=. -benchmem ./..."
    fi
}

# Run security tests
run_security_tests() {
    echo -e "\n${YELLOW}🔒 Running Security Tests${NC}"
    
    # Install security tools if needed
    if [ -f "go.mod" ] && ! command -v gosec &> /dev/null; then
        echo "Installing gosec..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    if [ -f "Cargo.toml" ] && ! command -v cargo-audit &> /dev/null; then
        echo "Installing cargo-audit..."
        cargo install cargo-audit
    fi
    
    # Run security scans
    if [ -f "go.mod" ]; then
        run_test_suite "Go Security Scan" "gosec -quiet ./..."
    fi
    
    if [ -f "Cargo.toml" ]; then
        run_test_suite "Rust Security Audit" "cargo audit"
    fi
    
    if [ -f "package.json" ]; then
        run_test_suite "Node.js Security Audit" "npm audit --audit-level moderate"
    fi
}

# Generate coverage reports
generate_coverage() {
    echo -e "\n${YELLOW}📊 Generating Coverage Reports${NC}"
    
    # Go coverage
    if [ -f "go.mod" ]; then
        echo "Generating Go coverage report..."
        go test -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage-go.html
        
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo "Go coverage: ${COVERAGE}%"
        
        if (( $(echo "$COVERAGE < $COVERAGE_THRESHOLD" | bc -l) )); then
            test_results["Go Coverage"]="FAIL"
            echo -e "${RED}❌ Go coverage below threshold: ${COVERAGE}% < ${COVERAGE_THRESHOLD}%${NC}"
        else
            test_results["Go Coverage"]="PASS"
            echo -e "${GREEN}✅ Go coverage meets threshold: ${COVERAGE}% >= ${COVERAGE_THRESHOLD}%${NC}"
        fi
    fi
    
    # React coverage
    if [ -f "package.json" ]; then
        echo "Generating React coverage report..."
        npm test -- --coverage --watchAll=false
        test_results["React Coverage"]="PASS"  # npm test will fail if coverage is below threshold
    fi
}

# Generate test report
generate_report() {
    echo -e "\n${YELLOW}📋 Generating Test Report${NC}"
    
    local report_file="test-report.md"
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    cat > "$report_file" << EOF
# Chimera Pool Test Report

Generated: $(date)

## Test Results Summary

| Test Suite | Status | Duration |
|------------|--------|----------|
EOF

    for suite in "${!test_results[@]}"; do
        local status="${test_results[$suite]}"
        local duration="${test_durations[$suite]:-0}"
        echo "| $suite | $status | ${duration}s |" >> "$report_file"
        
        total_tests=$((total_tests + 1))
        if [ "$status" = "PASS" ]; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done

    cat >> "$report_file" << EOF

## Summary

- **Total Test Suites**: $total_tests
- **Passed**: $passed_tests
- **Failed**: $failed_tests
- **Success Rate**: $(( passed_tests * 100 / total_tests ))%

## Coverage Reports

- Go: [coverage-go.html](coverage-go.html)
- React: [coverage/lcov-report/index.html](coverage/lcov-report/index.html)

## Quality Gates

- Coverage Threshold: ${COVERAGE_THRESHOLD}%
- Security Vulnerabilities: 0 (High/Critical)
- All Tests: Must Pass

EOF

    echo -e "${GREEN}✅ Test report generated: $report_file${NC}"
}

# Print final summary
print_summary() {
    echo -e "\n${YELLOW}=========================================="
    echo -e "🏁 Test Suite Complete"
    echo -e "==========================================${NC}"
    
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    local total_duration=0
    
    for suite in "${!test_results[@]}"; do
        local status="${test_results[$suite]}"
        local duration="${test_durations[$suite]:-0}"
        
        total_tests=$((total_tests + 1))
        total_duration=$((total_duration + duration))
        
        if [ "$status" = "PASS" ]; then
            passed_tests=$((passed_tests + 1))
            echo -e "${GREEN}✅ $suite${NC}"
        else
            failed_tests=$((failed_tests + 1))
            echo -e "${RED}❌ $suite${NC}"
        fi
    done
    
    echo -e "\n${YELLOW}Summary:${NC}"
    echo -e "Total: $total_tests | Passed: ${GREEN}$passed_tests${NC} | Failed: ${RED}$failed_tests${NC}"
    echo -e "Total Duration: ${total_duration}s"
    
    if [ $failed_tests -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All tests passed! Ready for deployment.${NC}"
        exit 0
    else
        echo -e "\n${RED}❌ Some tests failed. Please review and fix issues.${NC}"
        exit 1
    fi
}

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up test environment...${NC}"
    docker-compose -f deployments/docker/docker-compose.test.yml down
}

# Set trap for cleanup
trap cleanup EXIT

# Main execution
main() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        "unit")
            setup_environment
            run_unit_tests
            ;;
        "integration")
            setup_environment
            run_integration_tests
            ;;
        "property")
            run_property_tests
            ;;
        "security")
            run_security_tests
            ;;
        "benchmark")
            run_benchmarks
            ;;
        "coverage")
            setup_environment
            generate_coverage
            ;;
        "all"|*)
            setup_environment
            run_unit_tests
            run_integration_tests
            run_property_tests
            run_security_tests
            run_benchmarks
            generate_coverage
            generate_report
            ;;
    esac
    
    print_summary
}

# Execute main function
main "$@"