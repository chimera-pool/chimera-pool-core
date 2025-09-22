#!/bin/bash

# Simple setup script for Chimera Pool Spec-Driven Development

set -e

echo "🚀 Setting up Chimera Pool Spec-Driven Development Environment..."

# Create directory structure
echo "📁 Creating directory structure..."
mkdir -p specs/001-chimera-pool-universal-platform/{contracts,data-model}
mkdir -p templates
mkdir -p scripts/spec-kit

# Create basic templates
echo "📝 Creating templates..."

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

# Create helper scripts
echo "🔧 Creating helper scripts..."

cat > scripts/create-new-feature.sh << 'EOF'
#!/bin/bash

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <feature-name>"
    echo "Example: $0 advanced-mining-analytics"
    exit 1
fi

FEATURE_NAME="$1"
SPEC_NUMBER=$(printf "%03d" $(($(ls specs/ | grep -E '^[0-9]{3}-' | wc -l) + 1)))
SPEC_DIR="specs/${SPEC_NUMBER}-${FEATURE_NAME}"

echo "Creating new feature specification: ${FEATURE_NAME}"
echo "Spec directory: ${SPEC_DIR}"

mkdir -p "${SPEC_DIR}"/{contracts,data-model}

sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/spec-template.md > "${SPEC_DIR}/spec.md"
sed "s/\[FEATURE_NAME\]/${FEATURE_NAME}/g" templates/plan-template.md > "${SPEC_DIR}/plan.md"

touch "${SPEC_DIR}/research.md"
touch "${SPEC_DIR}/quickstart.md"

echo "✅ Feature specification created successfully!"
echo "Next steps:"
echo "1. Edit ${SPEC_DIR}/spec.md to define the feature requirements"
echo "2. Use Claude to help develop the specification and implementation plan"
EOF

chmod +x scripts/create-new-feature.sh

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

