# Chimera Pool Core - Source Code Structure

This directory contains the core components of the Chimera Pool mining software, organized by technology and responsibility.

## Component Architecture

### Algorithm Engine (Rust)
- **Location**: `algorithm-engine/`
- **Purpose**: Hot-swappable mining algorithm implementations
- **Technology**: Rust (for performance and safety)
- **Testing**: Unit tests + Property-based tests with proptest

### Pool Manager (Go)
- **Location**: `pool-manager/`
- **Purpose**: Core mining pool logic and coordination
- **Technology**: Go (for concurrency and networking)
- **Testing**: Unit tests + Integration tests with testcontainers

### Stratum Server (Go)
- **Location**: `stratum-server/`
- **Purpose**: Stratum protocol implementation for miner connections
- **Technology**: Go (for network performance)
- **Testing**: Unit tests + Protocol compliance tests

### Authentication Service (Go)
- **Location**: `auth-service/`
- **Purpose**: User authentication and authorization
- **Technology**: Go (for security and integration)
- **Testing**: Unit tests + Security tests

### Web Dashboard (React)
- **Location**: `web-dashboard/`
- **Purpose**: User interface and monitoring dashboard
- **Technology**: React + TypeScript (for modern UI)
- **Testing**: Unit tests + Component tests with Jest + RTL

## Component Interaction

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Web Dashboard  │    │ Stratum Server  │    │ Algorithm Engine│
│    (React)      │    │      (Go)       │    │     (Rust)      │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │              ┌───────▼───────┐              │
          └──────────────►│ Pool Manager  │◄─────────────┘
                         │     (Go)      │
                         └───────┬───────┘
                                 │
                         ┌───────▼───────┐
                         │ Auth Service  │
                         │     (Go)      │
                         └───────────────┘
```

## Development Guidelines

### Component Independence
- Each component should be independently testable
- Minimal coupling between components
- Clear interfaces and contracts
- Dependency injection for testing

### Testing Strategy
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **Property Tests**: Test algorithm correctness with random inputs
- **E2E Tests**: Test complete user workflows

### Quality Gates
- 90% test coverage minimum
- Zero security vulnerabilities
- All tests must pass
- Code formatting and linting compliance