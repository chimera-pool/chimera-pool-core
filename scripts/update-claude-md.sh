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
