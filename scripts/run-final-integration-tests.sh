#!/bin/bash

# Final Integration Testing Script
# This script runs comprehensive integration tests to validate production readiness

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT="30m"
PARALLEL_TESTS=4
COVERAGE_THRESHOLD=80

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Final Integration Testing Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to print status
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        print_error "Docker is not running"
        exit 1
    fi
    
    # Check if required environment variables are set
    required_vars=("DATABASE_URL" "REDIS_URL" "JWT_SECRET" "ENCRYPTION_KEY")
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            print_warning "Environment variable $var is not set, using default test value"
            case $var in
                "DATABASE_URL")
                    export DATABASE_URL="postgres://test:test@localhost:5432/chimera_pool_test?sslmode=disable"
                    ;;
                "REDIS_URL")
                    export REDIS_URL="redis://localhost:6379/0"
                    ;;
                "JWT_SECRET")
                    export JWT_SECRET="test-jwt-secret-key-for-integration-tests-only"
                    ;;
                "ENCRYPTION_KEY")
                    export ENCRYPTION_KEY="test-encryption-key-32-chars-long"
                    ;;
            esac
        fi
    done
    
    print_success "Prerequisites check completed"
}

# Function to setup test environment
setup_test_environment() {
    print_status "Setting up test environment..."
    
    # Start test databases using Docker Compose
    if [[ -f "deployments/docker/docker-compose.test.yml" ]]; then
        print_status "Starting test infrastructure..."
        docker-compose -f deployments/docker/docker-compose.test.yml up -d
        
        # Wait for services to be ready
        print_status "Waiting for services to be ready..."
        sleep 10
        
        # Check PostgreSQL
        for i in {1..30}; do
            if docker-compose -f deployments/docker/docker-compose.test.yml exec -T postgres pg_isready -U test &> /dev/null; then
                break
            fi
            if [[ $i -eq 30 ]]; then
                print_error "PostgreSQL failed to start"
                exit 1
            fi
            sleep 2
        done
        
        # Check Redis
        for i in {1..30}; do
            if docker-compose -f deployments/docker/docker-compose.test.yml exec -T redis redis-cli ping &> /dev/null; then
                break
            fi
            if [[ $i -eq 30 ]]; then
                print_error "Redis failed to start"
                exit 1
            fi
            sleep 2
        done
        
        print_success "Test infrastructure is ready"
    else
        print_warning "Docker Compose test file not found, assuming services are already running"
    fi
    
    # Run database migrations
    print_status "Running database migrations..."
    if [[ -f "scripts/migrate.sh" ]]; then
        ./scripts/migrate.sh up
    else
        # Run migrations using Go migrate tool or custom migration
        go run cmd/migrate/main.go up || print_warning "Migration script not found or failed"
    fi
    
    print_success "Test environment setup completed"
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."
    
    # Create test results directory
    mkdir -p test-results
    
    # Set test environment variables
    export GO_ENV=test
    export LOG_LEVEL=info
    export TEST_MODE=integration
    
    # Run tests with coverage
    print_status "Executing final integration test suite..."
    
    # Test suites to run in order
    test_suites=(
        "tests/integration/final_integration_test.go"
        "tests/integration/performance_test.go"
        "tests/integration/security_test.go"
        "tests/integration/production_readiness_test.go"
    )
    
    overall_success=true
    
    for suite in "${test_suites[@]}"; do
        suite_name=$(basename "$suite" .go)
        print_status "Running $suite_name..."
        
        # Run the test suite with comprehensive flags
        if go test -v -timeout="$TEST_TIMEOUT" \
           -coverprofile="test-results/${suite_name}_coverage.out" \
           -covermode=atomic \
           -race \
           -p="$PARALLEL_TESTS" \
           -tags="integration" \
           "./$suite" \
           > "test-results/${suite_name}_output.log" 2>&1; then
            print_success "$suite_name passed"
            
            # Extract key metrics from test output
            if grep -q "PASS" "test-results/${suite_name}_output.log"; then
                test_count=$(grep -c "=== RUN" "test-results/${suite_name}_output.log" || echo "0")
                duration=$(grep "PASS" "test-results/${suite_name}_output.log" | tail -1 | grep -o '[0-9.]*s' || echo "0s")
                print_status "$suite_name: $test_count tests completed in $duration"
            fi
        else
            print_error "$suite_name failed"
            echo "Test output:"
            tail -50 "test-results/${suite_name}_output.log"
            overall_success=false
            
            # Continue with other tests even if one fails
            print_status "Continuing with remaining test suites..."
        fi
    done
    
    # Generate combined coverage report
    print_status "Generating coverage report..."
    echo "mode: atomic" > test-results/combined_coverage.out
    for coverage_file in test-results/*_coverage.out; do
        if [[ -f "$coverage_file" ]]; then
            tail -n +2 "$coverage_file" >> test-results/combined_coverage.out
        fi
    done
    
    # Calculate coverage percentage
    if command -v go &> /dev/null; then
        coverage_percent=$(go tool cover -func=test-results/combined_coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        print_status "Overall test coverage: ${coverage_percent}%"
        
        if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_success "Coverage threshold met (${coverage_percent}% >= ${COVERAGE_THRESHOLD}%)"
        else
            print_warning "Coverage below threshold (${coverage_percent}% < ${COVERAGE_THRESHOLD}%)"
        fi
        
        # Generate HTML coverage report
        go tool cover -html=test-results/combined_coverage.out -o test-results/coverage.html
        print_status "HTML coverage report generated: test-results/coverage.html"
    fi
    
    return $overall_success
}

# Function to run performance benchmarks
run_performance_benchmarks() {
    print_status "Running performance benchmarks..."
    
    # Run Go benchmarks
    go test -bench=. -benchmem -timeout="$TEST_TIMEOUT" \
        ./tests/integration/performance_test.go \
        > test-results/benchmark_results.txt 2>&1
    
    if [[ $? -eq 0 ]]; then
        print_success "Performance benchmarks completed"
        print_status "Benchmark results saved to test-results/benchmark_results.txt"
    else
        print_warning "Some performance benchmarks failed"
    fi
}

# Function to validate security compliance
validate_security_compliance() {
    print_status "Validating security compliance..."
    
    # Run security-specific tests
    go test -v -timeout="$TEST_TIMEOUT" -tags=security \
        ./tests/integration/security_test.go \
        > test-results/security_validation.log 2>&1
    
    if [[ $? -eq 0 ]]; then
        print_success "Security compliance validation passed"
    else
        print_error "Security compliance validation failed"
        cat test-results/security_validation.log
        return 1
    fi
}

# Function to generate test report
generate_test_report() {
    print_status "Generating comprehensive test report..."
    
    report_file="test-results/final_integration_report.md"
    
    cat > "$report_file" << EOF
# Final Integration Test Report

**Generated:** $(date)
**Test Environment:** Integration
**Test Timeout:** $TEST_TIMEOUT
**Coverage Threshold:** $COVERAGE_THRESHOLD%

## Executive Summary

This report validates the production readiness of the Chimera Mining Pool software through comprehensive integration testing covering:

- **End-to-End Functionality**: Complete mining workflow validation
- **Performance Testing**: Load testing with 1000+ concurrent miners
- **Security Validation**: Comprehensive security framework testing
- **Production Readiness**: Deployment and operational readiness checks

## Test Results Summary

EOF
    
    # Add detailed test results summary
    total_tests=0
    passed_tests=0
    failed_tests=0
    
    for suite in tests/integration/*_test.go; do
        suite_name=$(basename "$suite" .go)
        log_file="test-results/${suite_name}_output.log"
        
        if [[ -f "$log_file" ]]; then
            total_tests=$((total_tests + 1))
            
            if grep -q "PASS" "$log_file"; then
                passed_tests=$((passed_tests + 1))
                test_count=$(grep -c "=== RUN" "$log_file" || echo "0")
                duration=$(grep "PASS" "$log_file" | tail -1 | grep -o '[0-9.]*s' || echo "0s")
                echo "- ✅ **$suite_name**: PASSED ($test_count tests in $duration)" >> "$report_file"
            else
                failed_tests=$((failed_tests + 1))
                echo "- ❌ **$suite_name**: FAILED" >> "$report_file"
                
                # Add failure details
                if grep -q "FAIL" "$log_file"; then
                    echo "  - Failure details:" >> "$report_file"
                    grep "FAIL:" "$log_file" | head -3 | sed 's/^/    - /' >> "$report_file"
                fi
            fi
        fi
    done
    
    # Calculate success rate
    if [[ $total_tests -gt 0 ]]; then
        success_rate=$(( passed_tests * 100 / total_tests ))
    else
        success_rate=0
    fi
    
    cat >> "$report_file" << EOF

### Overall Results
- **Total Test Suites**: $total_tests
- **Passed**: $passed_tests
- **Failed**: $failed_tests
- **Success Rate**: $success_rate%

## Coverage Analysis

EOF
    
    if [[ -f "test-results/combined_coverage.out" ]]; then
        coverage_percent=$(go tool cover -func=test-results/combined_coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        
        echo "**Overall Test Coverage**: ${coverage_percent}%" >> "$report_file"
        echo "" >> "$report_file"
        
        if (( $(echo "$coverage_percent >= $COVERAGE_THRESHOLD" | bc -l) )); then
            echo "✅ **Coverage Status**: PASSED (${coverage_percent}% >= ${COVERAGE_THRESHOLD}%)" >> "$report_file"
        else
            echo "❌ **Coverage Status**: FAILED (${coverage_percent}% < ${COVERAGE_THRESHOLD}%)" >> "$report_file"
        fi
        
        echo "" >> "$report_file"
        echo "### Coverage by Package" >> "$report_file"
        echo "" >> "$report_file"
        echo '```' >> "$report_file"
        go tool cover -func=test-results/combined_coverage.out >> "$report_file"
        echo '```' >> "$report_file"
    else
        echo "❌ **Coverage Status**: No coverage data available" >> "$report_file"
    fi
    
    cat >> "$report_file" << EOF

## Performance Analysis

EOF
    
    if [[ -f "test-results/benchmark_results.txt" ]]; then
        echo "### Benchmark Results" >> "$report_file"
        echo "" >> "$report_file"
        echo '```' >> "$report_file"
        cat test-results/benchmark_results.txt >> "$report_file"
        echo '```' >> "$report_file"
    fi
    
    # Add performance metrics from test logs
    echo "### Performance Metrics" >> "$report_file"
    echo "" >> "$report_file"
    
    for suite in tests/integration/*_test.go; do
        suite_name=$(basename "$suite" .go)
        log_file="test-results/${suite_name}_output.log"
        
        if [[ -f "$log_file" ]] && [[ "$suite_name" == "performance_test" ]]; then
            echo "#### $suite_name Results" >> "$report_file"
            echo "" >> "$report_file"
            
            # Extract performance metrics
            if grep -q "connections/second" "$log_file"; then
                grep "connections/second\|requests/second\|response time\|Success Rate" "$log_file" | sed 's/^/- /' >> "$report_file"
            fi
            echo "" >> "$report_file"
        fi
    done
    
    cat >> "$report_file" << EOF

## Security Validation

EOF
    
    # Add security test results
    security_log="test-results/security_test_output.log"
    if [[ -f "$security_log" ]]; then
        echo "### Security Test Results" >> "$report_file"
        echo "" >> "$report_file"
        
        if grep -q "PASS" "$security_log"; then
            echo "✅ **Security Framework**: All security tests passed" >> "$report_file"
        else
            echo "❌ **Security Framework**: Some security tests failed" >> "$security_log"
        fi
        
        # Extract security metrics
        if grep -q "Rate limiting\|MFA\|Encryption" "$security_log"; then
            echo "" >> "$report_file"
            echo "#### Security Features Validated" >> "$report_file"
            grep -i "rate limiting\|mfa\|encryption\|authentication\|authorization" "$security_log" | head -10 | sed 's/^/- /' >> "$report_file"
        fi
    fi
    
    cat >> "$report_file" << EOF

## Production Readiness Assessment

EOF
    
    # Add production readiness results
    prod_log="test-results/production_readiness_test_output.log"
    if [[ -f "$prod_log" ]]; then
        echo "### Production Readiness Checklist" >> "$report_file"
        echo "" >> "$report_file"
        
        # Extract readiness indicators
        if grep -q "Health check\|Monitoring\|Backup\|Docker" "$prod_log"; then
            grep -i "✅\|❌\|health\|monitoring\|backup\|docker\|deployment" "$prod_log" | head -15 | sed 's/^/- /' >> "$report_file"
        fi
    fi
    
    cat >> "$report_file" << EOF

## Recommendations

EOF
    
    # Generate recommendations based on test results
    if [[ $failed_tests -gt 0 ]]; then
        echo "### Critical Issues" >> "$report_file"
        echo "" >> "$report_file"
        echo "- $failed_tests test suite(s) failed and must be addressed before production deployment" >> "$report_file"
        echo "- Review failed test logs for specific issues and remediation steps" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    if [[ -f "test-results/combined_coverage.out" ]]; then
        coverage_percent=$(go tool cover -func=test-results/combined_coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$coverage_percent < $COVERAGE_THRESHOLD" | bc -l) )); then
            echo "### Coverage Improvements" >> "$report_file"
            echo "" >> "$report_file"
            echo "- Test coverage is below threshold (${coverage_percent}% < ${COVERAGE_THRESHOLD}%)" >> "$report_file"
            echo "- Add more unit and integration tests for uncovered code paths" >> "$report_file"
            echo "" >> "$report_file"
        fi
    fi
    
    if [[ $success_rate -eq 100 ]] && [[ $failed_tests -eq 0 ]]; then
        echo "### Production Deployment" >> "$report_file"
        echo "" >> "$report_file"
        echo "✅ **System is ready for production deployment**" >> "$report_file"
        echo "" >> "$report_file"
        echo "All integration tests passed successfully. The system demonstrates:" >> "$report_file"
        echo "- Stable end-to-end functionality" >> "$report_file"
        echo "- Adequate performance under load" >> "$report_file"
        echo "- Comprehensive security measures" >> "$report_file"
        echo "- Production deployment readiness" >> "$report_file"
    else
        echo "### Pre-Production Tasks" >> "$report_file"
        echo "" >> "$report_file"
        echo "❌ **System requires additional work before production deployment**" >> "$report_file"
        echo "" >> "$report_file"
        echo "Address the following issues:" >> "$report_file"
        echo "- Fix all failing test suites" >> "$report_file"
        echo "- Improve test coverage if below threshold" >> "$report_file"
        echo "- Validate all security measures" >> "$report_file"
        echo "- Complete production readiness checklist" >> "$report_file"
    fi
    
    cat >> "$report_file" << EOF

## Test Environment Details

- **Go Version**: $(go version)
- **Test Timeout**: $TEST_TIMEOUT
- **Parallel Tests**: $PARALLEL_TESTS
- **Test Tags**: integration
- **Coverage Mode**: atomic
- **Race Detection**: enabled

## Files Generated

- Test logs: \`test-results/*_output.log\`
- Coverage reports: \`test-results/*_coverage.out\`
- HTML coverage: \`test-results/coverage.html\`
- This report: \`test-results/final_integration_report.md\`

---
*Report generated by Chimera Pool Final Integration Test Suite*
EOF
    
    print_success "Comprehensive test report generated: $report_file"
    
    # Also generate a summary for console output
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Test Report Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo -e "Total Test Suites: $total_tests"
    echo -e "Passed: ${GREEN}$passed_tests${NC}"
    echo -e "Failed: ${RED}$failed_tests${NC}"
    echo -e "Success Rate: $success_rate%"
    
    if [[ -f "test-results/combined_coverage.out" ]]; then
        coverage_percent=$(go tool cover -func=test-results/combined_coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        echo -e "Test Coverage: ${coverage_percent}%"
    fi
    
    echo -e "${BLUE}========================================${NC}"
}

# Function to cleanup test environment
cleanup_test_environment() {
    print_status "Cleaning up test environment..."
    
    # Stop test infrastructure
    if [[ -f "deployments/docker/docker-compose.test.yml" ]]; then
        docker-compose -f deployments/docker/docker-compose.test.yml down -v
    fi
    
    print_success "Test environment cleanup completed"
}

# Main execution
main() {
    echo -e "${BLUE}Starting Final Integration Testing...${NC}"
    echo ""
    
    # Trap to ensure cleanup on exit
    trap cleanup_test_environment EXIT
    
    # Run all test phases
    check_prerequisites
    setup_test_environment
    
    # Run the actual tests
    test_success=true
    
    if ! run_integration_tests; then
        test_success=false
    fi
    
    run_performance_benchmarks
    
    if ! validate_security_compliance; then
        test_success=false
    fi
    
    generate_test_report
    
    echo ""
    echo -e "${BLUE}========================================${NC}"
    if [[ "$test_success" == true ]]; then
        echo -e "${GREEN}  ALL TESTS PASSED - PRODUCTION READY${NC}"
        echo -e "${BLUE}========================================${NC}"
        exit 0
    else
        echo -e "${RED}  SOME TESTS FAILED - NOT PRODUCTION READY${NC}"
        echo -e "${BLUE}========================================${NC}"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    "help"|"-h"|"--help")
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  help, -h, --help    Show this help message"
        echo "  clean               Clean test results and exit"
        echo ""
        echo "Environment Variables:"
        echo "  TEST_TIMEOUT        Test timeout (default: 30m)"
        echo "  PARALLEL_TESTS      Number of parallel tests (default: 4)"
        echo "  COVERAGE_THRESHOLD  Coverage threshold percentage (default: 80)"
        exit 0
        ;;
    "clean")
        print_status "Cleaning test results..."
        rm -rf test-results/
        cleanup_test_environment
        print_success "Test results cleaned"
        exit 0
        ;;
    "")
        main
        ;;
    *)
        print_error "Unknown option: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac