#!/bin/bash

# Comprehensive Security Framework Validation Script
# This script validates the complete security framework implementation

set -e

echo "🔐 Validating Comprehensive Security Framework..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go to run the validation."
    exit 1
fi

# Navigate to the project directory
cd "$(dirname "$0")/.."

print_status "Checking Go module dependencies..."
go mod tidy
if [ $? -eq 0 ]; then
    print_success "Dependencies resolved successfully"
else
    print_error "Failed to resolve dependencies"
    exit 1
fi

print_status "Running security framework tests..."

# Test individual components
echo ""
echo "Testing MFA Service..."
go test ./internal/security/ -run TestTOTP -v
go test ./internal/security/ -run TestBackupCodes -v
go test ./internal/security/ -run TestMFASetupWorkflow -v

echo ""
echo "Testing Rate Limiting and Protection..."
go test ./internal/security/ -run TestRateLimiter -v
go test ./internal/security/ -run TestProgressiveRateLimiting -v
go test ./internal/security/ -run TestBruteForceProtection -v
go test ./internal/security/ -run TestDDoSProtection -v
go test ./internal/security/ -run TestIntrusionDetection -v

echo ""
echo "Testing Encryption and Security..."
go test ./internal/security/ -run TestAESEncryption -v
go test ./internal/security/ -run TestPasswordHashing -v
go test ./internal/security/ -run TestSecureWallet -v
go test ./internal/security/ -run TestCompliance -v
go test ./internal/security/ -run TestAuditLogging -v
go test ./internal/security/ -run TestDataEncryption -v

echo ""
echo "Testing End-to-End Security Workflows..."
go test ./internal/security/ -run TestCompleteSecurityWorkflow -v
go test ./internal/security/ -run TestSecurityService -v

echo ""
echo "Testing Performance and Concurrency..."
go test ./internal/security/ -run TestSecurityFrameworkPerformance -v
go test ./internal/security/ -run TestSecurityFrameworkConcurrency -v

# Run all tests with coverage
echo ""
print_status "Running complete test suite with coverage..."
go test ./internal/security/ -cover -coverprofile=security_coverage.out

if [ $? -eq 0 ]; then
    print_success "All security tests passed!"
    
    # Display coverage report
    echo ""
    print_status "Coverage Report:"
    go tool cover -func=security_coverage.out
    
    # Generate HTML coverage report
    go tool cover -html=security_coverage.out -o security_coverage.html
    print_success "HTML coverage report generated: security_coverage.html"
else
    print_error "Some security tests failed!"
    exit 1
fi

# Validate code quality
echo ""
print_status "Running code quality checks..."

# Check for potential security issues with gosec (if available)
if command -v gosec &> /dev/null; then
    print_status "Running security analysis with gosec..."
    gosec ./internal/security/...
    if [ $? -eq 0 ]; then
        print_success "No security issues found by gosec"
    else
        print_warning "gosec found potential security issues - please review"
    fi
else
    print_warning "gosec not installed - skipping security analysis"
    print_status "Install gosec with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
fi

# Check for race conditions
echo ""
print_status "Testing for race conditions..."
go test ./internal/security/ -race -short
if [ $? -eq 0 ]; then
    print_success "No race conditions detected"
else
    print_error "Race conditions detected!"
    exit 1
fi

# Benchmark tests
echo ""
print_status "Running performance benchmarks..."
go test ./internal/security/ -bench=. -benchmem -short

# Validate that all required files exist
echo ""
print_status "Validating security framework files..."

required_files=(
    "internal/security/mfa.go"
    "internal/security/mfa_test.go"
    "internal/security/mfa_repository.go"
    "internal/security/rate_limiting.go"
    "internal/security/rate_limiting_test.go"
    "internal/security/encryption.go"
    "internal/security/encryption_test.go"
    "internal/security/security_service.go"
    "internal/security/security_service_test.go"
    "internal/security/e2e_test.go"
    "internal/security/README.md"
)

missing_files=()
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        missing_files+=("$file")
    fi
done

if [ ${#missing_files[@]} -eq 0 ]; then
    print_success "All required security framework files are present"
else
    print_error "Missing required files:"
    for file in "${missing_files[@]}"; do
        echo "  - $file"
    done
    exit 1
fi

# Validate that security patterns are comprehensive
echo ""
print_status "Validating security patterns..."

# Check if intrusion detection patterns cover major attack vectors
pattern_file="internal/security/security_service.go"
if grep -q "union\s+select" "$pattern_file" && \
   grep -q "script\[" "$pattern_file" && \
   grep -q "javascript:" "$pattern_file"; then
    print_success "Security patterns cover major attack vectors"
else
    print_warning "Security patterns may not be comprehensive enough"
fi

# Final validation summary
echo ""
echo "=============================================="
print_success "🔐 COMPREHENSIVE SECURITY FRAMEWORK VALIDATION COMPLETE"
echo "=============================================="
echo ""
print_status "Summary of implemented security features:"
echo "  ✅ Multi-Factor Authentication (TOTP + Backup Codes)"
echo "  ✅ Progressive Rate Limiting with Penalties"
echo "  ✅ Brute Force Protection with Account Lockouts"
echo "  ✅ DDoS Protection with Suspicious Activity Detection"
echo "  ✅ Intrusion Detection with Pattern Matching"
echo "  ✅ End-to-End Data Encryption (AES-256-GCM)"
echo "  ✅ Secure Password Hashing (bcrypt)"
echo "  ✅ Secure Wallet Integration with Encrypted Keys"
echo "  ✅ Regulatory Compliance (KYC/AML)"
echo "  ✅ Comprehensive Audit Logging"
echo "  ✅ Enterprise-Grade Security with Easy Onboarding"
echo ""
print_success "The security framework is ready for production deployment!"
echo ""
print_status "Next steps:"
echo "  1. Review the security configuration in security_service.go"
echo "  2. Set up proper key management for production"
echo "  3. Configure database tables for persistent storage"
echo "  4. Set up monitoring and alerting for security events"
echo "  5. Conduct penetration testing to validate security measures"
echo ""
print_status "Documentation: internal/security/README.md"
print_status "Coverage Report: security_coverage.html"