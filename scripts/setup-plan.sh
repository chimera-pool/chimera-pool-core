#!/bin/bash

# Chimera Pool - Setup Plan Script
# This script sets up the spec-driven development environment

set -e

echo "🚀 Setting up Chimera Pool Spec-Driven Development Environment..."

# Create spec-kit directory structure
echo "📁 Creating spec-kit directory structure..."
mkdir -p specs/001-chimera-pool-universal-platform/{contracts,data-model}

# Create templates directory
echo "📝 Creating templates..."
mkdir -p templates

# Create scripts directory structure
echo "🔧 Setting up scripts..."
mkdir -p scripts/{common,spec-kit}

# Create spec template
cat > templates/spec-template.md << 'EOF'
# Feature Specification: [FEATURE_NAME]

## Overview
Brief description of the feature and its purpose.

## User Stories
- As a [user type], I want [functionality] so that [benefit]

## Functional Requirements
### Requirement 1: [Name]
**User Story:** As a [user], I want [functionality] so that [benefit]

#### Acceptance Criteria
1. WHEN [condition] THEN [expected behavior]
2. WHEN [condition] THEN [expected behavior]
3. IF [condition] THEN [expected behavior]

## Technical Specifications
- Architecture considerations
- Performance requirements
- Security requirements
- Integration points

## Review & Acceptance Checklist
- [ ] All user stories are covered
- [ ] Acceptance criteria are testable
- [ ] Performance requirements are specified
- [ ] Security considerations are addressed
- [ ] Integration points are defined
- [ ] Error handling is specified
- [ ] Documentation requirements are clear

## Implementation Notes
Any additional notes for implementation.
EOF

# Create plan template
cat > templates/plan-template.md << 'EOF'
# Implementation Plan: [FEATURE_NAME]

## Overview
Implementation strategy and approach for the feature.

## Technical Architecture
### Components
- Component 1: Description
- Component 2: Description

### Dependencies
- Internal dependencies
- External dependencies

## Implementation Tasks
### Phase 1: Core Implementation
- [ ] Task 1
- [ ] Task 2
- [ ] Task 3

### Phase 2: Integration
- [ ] Task 1
- [ ] Task 2

### Phase 3: Testing & Validation
- [ ] Unit tests
- [ ] Integration tests
- [ ] Performance tests

## Risk Assessment
- Risk 1: Description and mitigation
- Risk 2: Description and mitigation

## Success Criteria
- Criteria 1
- Criteria 2
- Criteria 3
EOF

# Create tasks template
cat > templates/tasks-template.md << 'EOF'
# Task Breakdown: [FEATURE_NAME]

## Task List

### Core Development Tasks
1. **[TASK_NAME]** (Priority: High)
   - Description: What needs to be done
   - Acceptance Criteria: How to know it's complete
   - Dependencies: What must be done first
   - Estimated Effort: Time estimate
   - Assignee: Who will do it

2. **[TASK_NAME]** (Priority: Medium)
   - Description: What needs to be done
   - Acceptance Criteria: How to know it's complete
   - Dependencies: What must be done first
   - Estimated Effort: Time estimate
   - Assignee: Who will do it

### Testing Tasks
1. **Unit Tests**
   - Write comprehensive unit tests
   - Achieve 90%+ coverage
   - Dependencies: Core implementation

2. **Integration Tests**
   - End-to-end testing
   - Performance validation
   - Dependencies: Unit tests

### Documentation Tasks
1. **API Documentation**
   - Document all endpoints
   - Include examples
   - Dependencies: Implementation

2. **User Documentation**
   - Setup guides
   - Usage examples
   - Dependencies: Implementation

## Task Dependencies
```
Task A → Task B → Task C
Task D → Task E
```

## Parallel Execution
Tasks that can be done simultaneously:
- Task X and Task Y
- Task Z and Task W
EOF

# Create common script functions
cat > scripts/common.sh << 'EOF'
#!/bin/bash

# Common functions for Chimera Pool scripts

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    if [[ "${DEBUG}" == "true" ]]; then
        echo -e "${PURPLE}[DEBUG]${NC} $1"
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    
    if ! command_exists git; then
        missing_tools+=("git")
    fi
    
    if ! command_exists go; then
        missing_tools+=("go")
    fi
    
    if ! command_exists cargo; then
        missing_tools+=("rust/cargo")
    fi
    
    if ! command_exists node; then
        missing_tools+=("node.js")
    fi
    
    if ! command_exists docker; then
        missing_tools+=("docker")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install the missing tools and try again."
        return 1
    fi
    
    log_success "All prerequisites are installed"
    return 0
}

# Get project root directory
get_project_root() {
    git rev-parse --show-toplevel 2>/dev/null || pwd
}

# Validate spec structure
validate_spec_structure() {
    local spec_dir="$1"
    
    if [[ ! -f "${spec_dir}/spec.md" ]]; then
        log_error "Missing spec.md in ${spec_dir}"
        return 1
    fi
    
    if [[ ! -f "${spec_dir}/plan.md" ]]; then
        log_warning "Missing plan.md in ${spec_dir}"
    fi
    
    if [[ ! -f "${spec_dir}/tasks.md" ]]; then
        log_warning "Missing tasks.md in ${spec_dir}"
    fi
    
    return 0
}
EOF

# Make scripts executable
chmod +x scripts/common.sh

# Source common functions
source scripts/common.sh

# Create spec-kit specific script
cat > scripts/create-new-feature.sh << 'EOF'
#!/bin/bash

# Create new feature specification
# Usage: ./scripts/create-new-feature.sh <feature-name>

set -e

# Source common functions
source "$(dirname "$0")/common.sh"

if [[ $# -ne 1 ]]; then
    log_error "Usage: $0 <feature-name>"
    log_info "Example: $0 advanced-mining-analytics"
    exit 1
fi

FEATURE_NAME="$1"
SPEC_NUMBER=$(printf "%03d" $(($(ls specs/ | grep -E '^[0-9]{3}-' | wc -l) + 1)))
SPEC_DIR="specs/${SPEC_NUMBER}-${FEATURE_NAME}"

log_info "Creating new feature specification: ${FEATURE_NAME}"
log_info "Spec directory: ${SPEC_DIR}"

# Create spec directory structure
mkdir -p "${SPEC_DIR}"/{contracts,data-model}

# Create spec.md from template
sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/spec-template.md > "${SPEC_DIR}/spec.md"

# Create plan.md from template
sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/plan-template.md > "${SPEC_DIR}/plan.md"

# Create tasks.md from template
sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/tasks-template.md > "${SPEC_DIR}/tasks.md"

# Create empty files for additional documentation
touch "${SPEC_DIR}/research.md"
touch "${SPEC_DIR}/quickstart.md"

log_success "Feature specification created successfully!"
log_info "Next steps:"
log_info "1. Edit ${SPEC_DIR}/spec.md to define the feature requirements"
log_info "2. Run: ./scripts/update-claude-md.sh to update Claude context"
log_info "3. Use Claude to help develop the specification and implementation plan"
EOF

chmod +x scripts/create-new-feature.sh

# Create Claude context update script
cat > scripts/update-claude-md.sh << 'EOF'
#!/bin/bash

# Update CLAUDE.md with current project status
# This helps maintain context for Claude AI assistant

set -e

source "$(dirname "$0")/common.sh"

PROJECT_ROOT=$(get_project_root)
CLAUDE_FILE="${PROJECT_ROOT}/CLAUDE.md"

log_info "Updating Claude context file..."

# Get current timestamp
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")

# Count specs
SPEC_COUNT=$(find specs -name "spec.md" 2>/dev/null | wc -l)

# Get recent git activity
RECENT_COMMITS=$(git log --oneline -5 2>/dev/null || echo "No git history available")

# Update CLAUDE.md with current status
cat > "${CLAUDE_FILE}" << EOF
# Claude AI Assistant Context

## Project Overview

This is the **Chimera Pool Universal Mining Pool Platform** - a next-generation, universal mining pool platform that supports multiple cryptocurrencies through its revolutionary hot-swappable algorithm engine.

## Current Development Status (Updated: ${TIMESTAMP})

### Specifications
- Total specifications: ${SPEC_COUNT}
- Active development using Spec-Driven Development methodology
- Following GitHub Spec Kit best practices

### Recent Development Activity
\`\`\`
${RECENT_COMMITS}
\`\`\`

### Key Features Implemented
- Hot-swappable algorithm engine (Rust)
- Multi-cryptocurrency support (Bitcoin, Ethereum Classic, BlockDAG, Litecoin, etc.)
- Stratum v1 protocol implementation
- Web dashboard with cyber-minimal design
- Comprehensive testing suite with TDD approach
- One-click installation system
- Enterprise-grade security features

### Technology Stack
- **Backend**: Go (pool management, Stratum server, APIs)
- **Algorithm Engine**: Rust (high-performance cryptographic operations)
- **Frontend**: React + TypeScript (cyber-minimal themed dashboard)
- **Database**: PostgreSQL (primary), Redis (caching), InfluxDB (time-series)
- **Infrastructure**: Docker, NGINX, Prometheus + Grafana

## Development Approach

We are using **Spec-Driven Development** with the GitHub Spec Kit methodology:

1. **Specification First**: Define requirements and acceptance criteria
2. **Implementation Planning**: Break down into actionable tasks
3. **Test-Driven Development**: Write tests before implementation
4. **Continuous Validation**: Ensure implementation matches specification

## Key Differentiators

### Universal Platform Approach
- **100x larger addressable market** than single-coin solutions
- **Multiple revenue streams** from different cryptocurrencies
- **Competitive moat** through technical innovation
- **Future-proof architecture** ready for any new cryptocurrency

### Technical Excellence
- Hot-swappable algorithms with zero downtime
- Enterprise performance (10,000+ concurrent miners)
- Comprehensive security framework
- One-click deployment and setup
- Cyber-minimal design aesthetic

## Quality Standards

- 90%+ test coverage requirement
- TDD methodology throughout
- Interface Segregation Principle (ISP)
- Event-driven architecture
- Comprehensive error handling and recovery

## Available Commands

- \`./scripts/create-new-feature.sh <name>\` - Create new feature specification
- \`./scripts/setup-plan.sh\` - Setup spec-driven development environment
- \`./scripts/update-claude-md.sh\` - Update this context file

---

*This file is automatically updated to maintain context for AI-assisted development.*
*Last updated: ${TIMESTAMP}*
EOF

log_success "Claude context file updated successfully!"
log_info "File location: \${CLAUDE_FILE}"
EOF

chmod +x scripts/update-claude-md.sh

# Update the CLAUDE.md file
./scripts/update-claude-md.sh

echo "✅ Spec-driven development environment setup complete!"
echo ""
echo "📋 Next steps:"
echo "1. Create a new feature specification: ./scripts/create-new-feature.sh <feature-name>"
echo "2. Use Claude AI to help develop specifications and implementation plans"
echo "3. Follow the spec-driven development workflow"
echo ""
echo "📁 Directory structure created:"
echo "   specs/                    - Feature specifications"
echo "   templates/                - Specification templates"
echo "   scripts/                  - Development scripts"
echo "   memory/                   - Project constitution and context"
echo ""
echo "🤖 Claude context file: CLAUDE.md"
echo "📖 Project constitution: memory/constitution.md"
