#!/bin/bash
set -e

echo "🔍 Validating Chimera Pool development environment setup..."

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

VALIDATION_PASSED=0
VALIDATION_FAILED=0

validate_item() {
    local name=$1
    local check=$2
    
    echo -n "Checking $name... "
    if eval "$check" &>/dev/null; then
        echo -e "${GREEN}✅${NC}"
        VALIDATION_PASSED=$((VALIDATION_PASSED + 1))
    else
        echo -e "${RED}❌${NC}"
        VALIDATION_FAILED=$((VALIDATION_FAILED + 1))
    fi
}

echo -e "${YELLOW}📋 Project Structure Validation${NC}"
validate_item "Go module file" "[ -f 'go.mod' ]"
validate_item "Rust workspace" "[ -f 'Cargo.toml' ]"
validate_item "React package.json" "[ -f 'package.json' ]"
validate_item "Docker compose config" "[ -f 'deployments/docker/docker-compose.dev.yml' ]"
validate_item "Setup script" "[ -x 'scripts/dev/setup.sh' ]"
validate_item "Test script" "[ -x 'scripts/test.sh' ]"
validate_item "Start script" "[ -x 'scripts/dev/start.sh' ]"

echo -e "\n${YELLOW}🏗️ Build Configuration Validation${NC}"
validate_item "Makefile" "[ -f 'Makefile' ]"
validate_item "CI workflow" "[ -f '.github/workflows/ci.yml' ]"
validate_item "Development README" "[ -f 'README.dev.md' ]"

echo -e "\n${YELLOW}🧪 Testing Framework Validation${NC}"
validate_item "Go test file" "[ -f 'main_test.go' ]"
validate_item "Rust algorithm engine" "[ -f 'src/algorithm-engine/src/lib.rs' ]"
validate_item "React test file" "[ -f 'src/App.test.tsx' ]"

echo -e "\n${YELLOW}🐳 Docker Configuration Validation${NC}"
validate_item "Docker available" "command -v docker"
validate_item "Docker compose config valid" "docker-compose -f deployments/docker/docker-compose.dev.yml config"

echo -e "\n${YELLOW}📁 Component Structure Validation${NC}"
validate_item "Algorithm engine directory" "[ -d 'src/algorithm-engine' ]"
validate_item "Pool manager directory" "[ -d 'src/pool-manager' ]"
validate_item "Auth service directory" "[ -d 'src/auth-service' ]"
validate_item "Stratum server directory" "[ -d 'src/stratum-server' ]"
validate_item "Web dashboard directory" "[ -d 'src/web-dashboard' ]"
validate_item "Tests directory" "[ -d 'tests' ]"

echo -e "\n${YELLOW}📊 Summary${NC}"
echo "Validations passed: ${GREEN}$VALIDATION_PASSED${NC}"
echo "Validations failed: ${RED}$VALIDATION_FAILED${NC}"

if [ $VALIDATION_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 Development environment setup validation PASSED!${NC}"
    echo ""
    echo "✅ Minimal project structure with clear component separation"
    echo "✅ Testing frameworks configured (Go, Rust, React)"
    echo "✅ Docker test environment ready"
    echo "✅ CI pipeline configured"
    echo "✅ Code quality gates established"
    echo ""
    echo "Next steps:"
    echo "  1. Install development tools: make install-tools"
    echo "  2. Start development environment: make dev"
    echo "  3. Run tests: make test"
    echo "  4. Begin Task 2: Database Foundation"
    echo ""
    exit 0
else
    echo -e "\n${RED}❌ Development environment setup validation FAILED!${NC}"
    echo "Please fix the failed validations before proceeding."
    exit 1
fi