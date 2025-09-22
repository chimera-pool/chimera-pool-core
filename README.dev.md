# Chimera Pool - Development Guide

This guide will help you set up and work with the Chimera Pool development environment.

## Quick Start

```bash
# 1. Set up development environment
make setup

# 2. Start development environment
make dev

# 3. Run tests
make test
```

## Prerequisites

- **Go 1.21+** - Backend services and pool management
- **Rust 1.70+** - High-performance algorithm engine
- **Node.js 18+** - Frontend dashboard
- **Docker & Docker Compose** - Development infrastructure
- **PostgreSQL 15+** - Database (via Docker)
- **Redis 7+** - Caching (via Docker)

## Project Structure

```
chimera-pool-core/
├── src/
│   ├── algorithm-engine/     # Rust - Hot-swappable algorithms
│   ├── pool-manager/         # Go - Core pool management
│   ├── auth-service/         # Go - Authentication service
│   ├── stratum-server/       # Go - Stratum protocol server
│   └── web-dashboard/        # React - Cyber-minimal dashboard
├── tests/
│   ├── unit/                 # Unit tests
│   ├── integration/          # Integration tests
│   └── e2e/                  # End-to-end tests
├── deployments/
│   └── docker/               # Docker configurations
├── scripts/
│   ├── dev/                  # Development scripts
│   ├── test.sh              # Test runner
│   └── install.sh           # Production installer
└── docs/                     # Documentation
```

## Development Workflow

### 1. Component-First TDD Approach

Each component follows this strict workflow:

1. **🔴 Red Phase**: Write failing tests that define behavior
2. **🟢 Green Phase**: Write minimal code to make tests pass
3. **🔵 Refactor Phase**: Improve code quality while keeping tests green
4. **✅ E2E Validation**: Test component integration
5. **📊 Quality Gates**: Ensure 90% coverage and security validation

### 2. Available Commands

```bash
# Development Environment
make setup          # Set up development environment
make dev            # Start development environment
make stop           # Stop all services
make status         # Show environment status

# Testing
make test           # Run all tests with coverage
make test-go        # Run Go tests only
make test-rust      # Run Rust tests only
make test-react     # Run React tests only
make quick-test     # Fast tests without coverage

# Code Quality
make lint           # Run all linters
make security       # Run security checks

# Building
make build          # Build all components
make build-release  # Build optimized release versions

# Database
make db-reset       # Reset development database

# Utilities
make logs           # Show development logs
make clean          # Clean all build artifacts
```

### 3. Testing Strategy

#### Unit Tests
- **Go**: `go test` with testify framework
- **Rust**: `cargo test` with proptest for property testing
- **React**: Jest with React Testing Library

#### Integration Tests
- Use testcontainers for isolated database testing
- Test component interactions
- Validate API contracts

#### E2E Tests
- Full workflow testing
- Simulated mining scenarios
- Performance validation

#### Coverage Requirements
- **Minimum 90% coverage** for all components
- Automated coverage reporting in CI
- Coverage gates prevent merging low-coverage code

### 4. Security Standards

- **Go**: gosec security scanner
- **Rust**: cargo-audit for vulnerability scanning
- **Node.js**: npm audit for dependency vulnerabilities
- **Docker**: Security scanning in CI pipeline

## Development Environment

### Services

When you run `make dev`, the following services start:

| Service | URL | Purpose |
|---------|-----|---------|
| Frontend | http://localhost:3000 | React dashboard |
| API | http://localhost:8080 | REST API |
| Database UI | http://localhost:8080 | Adminer (PostgreSQL UI) |
| PostgreSQL | localhost:5432 | Primary database |
| Redis | localhost:6379 | Caching and sessions |

### Database Access

```bash
# Via Adminer (Web UI)
open http://localhost:8080

# Via psql
docker-compose -f deployments/docker/docker-compose.dev.yml exec postgres psql -U chimera -d chimera_pool_dev

# Connection string
postgresql://chimera:dev_password@localhost:5432/chimera_pool_dev
```

## Component Development

### Adding a New Go Component

1. Create directory structure:
   ```bash
   mkdir -p src/my-service/{cmd,internal}
   ```

2. Write failing tests first:
   ```go
   func TestMyService(t *testing.T) {
       // Define expected behavior
   }
   ```

3. Implement minimal functionality
4. Refactor and improve
5. Add integration tests

### Adding a New Rust Component

1. Add to workspace in `Cargo.toml`:
   ```toml
   members = [
       "src/algorithm-engine",
       "src/my-rust-component",
   ]
   ```

2. Follow TDD approach with property testing:
   ```rust
   #[cfg(test)]
   mod tests {
       use proptest::prelude::*;
       
       proptest! {
           #[test]
           fn test_property(input in any::<u64>()) {
               // Property-based test
           }
       }
   }
   ```

### Adding React Components

1. Create component with tests:
   ```typescript
   // MyComponent.test.tsx
   import { render, screen } from '@testing-library/react';
   import MyComponent from './MyComponent';
   
   test('renders component', () => {
       render(<MyComponent />);
       // Test behavior
   });
   ```

2. Implement component following cyber-minimal design system
3. Ensure accessibility compliance

## Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Reset database
make db-reset

# Check database status
docker-compose -f deployments/docker/docker-compose.dev.yml ps postgres
```

#### Port Conflicts
```bash
# Check what's using ports
lsof -i :3000  # Frontend
lsof -i :8080  # API/Adminer
lsof -i :5432  # PostgreSQL
```

#### Test Failures
```bash
# Run specific test suite
make test-go
make test-rust
make test-react

# Run with verbose output
go test -v ./...
cargo test --workspace -- --nocapture
npm test -- --verbose
```

### Getting Help

1. **Check logs**: `make logs`
2. **Verify setup**: `make status`
3. **Reset environment**: `make clean && make setup`
4. **Review test output**: Tests provide detailed error messages and suggested actions

## Quality Gates

Before submitting code, ensure:

- [ ] All tests pass: `make test`
- [ ] Coverage ≥ 90%
- [ ] Security checks pass: `make security`
- [ ] Linting passes: `make lint`
- [ ] Component follows TDD approach
- [ ] Integration tests validate component interaction

## Next Steps

1. **Start with Task 1**: Development Environment (already completed!)
2. **Move to Task 2**: Database Foundation
3. **Follow the component-first approach**: One component at a time
4. **Maintain quality gates**: Never compromise on testing and security

Happy coding! 🚀