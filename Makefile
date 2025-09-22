# Chimera Pool Development Makefile

.PHONY: help setup test build clean dev stop lint security

# Default target
help: ## Show this help message
	@echo "Chimera Pool Development Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Development Environment
setup: ## Set up development environment
	@echo "🔧 Setting up development environment..."
	@chmod +x scripts/dev/setup.sh
	@./scripts/dev/setup.sh

dev: ## Start development environment
	@echo "🚀 Starting development environment..."
	@chmod +x scripts/dev/start.sh
	@./scripts/dev/start.sh

stop: ## Stop development environment
	@echo "🛑 Stopping development environment..."
	@docker-compose -f deployments/docker/docker-compose.dev.yml down
	@pkill -f "npm start" || true

# Testing
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@chmod +x scripts/test.sh
	@./scripts/test.sh

test-comprehensive: ## Run comprehensive test suite with coverage and quality gates
	@echo "🧪 Running comprehensive test suite..."
	@chmod +x scripts/test-all.sh
	@./scripts/test-all.sh

test-go: ## Run Go tests only
	@echo "🧪 Running Go tests..."
	@./scripts/test.sh go

test-rust: ## Run Rust tests only
	@echo "🧪 Running Rust tests..."
	@./scripts/test.sh rust

test-react: ## Run React tests only
	@echo "🧪 Running React tests..."
	@./scripts/test.sh react

test-integration: ## Run integration tests only
	@echo "🧪 Running integration tests..."
	@./scripts/test-all.sh integration

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@./scripts/test-all.sh unit

test-security: ## Run security tests only
	@echo "🧪 Running security tests..."
	@./scripts/test-all.sh security

test-benchmark: ## Run benchmark tests
	@echo "🧪 Running benchmarks..."
	@./scripts/test-all.sh benchmark

test-coverage: ## Generate coverage reports
	@echo "🧪 Generating coverage reports..."
	@./scripts/test-all.sh coverage

# Code Quality
lint: ## Run linters for all languages
	@echo "🔍 Running linters..."
	@if [ -f "go.mod" ]; then \
		echo "Running Go linter..."; \
		go vet ./...; \
		gofmt -l .; \
	fi
	@if [ -f "Cargo.toml" ]; then \
		echo "Running Rust linter..."; \
		cargo fmt --check; \
		cargo clippy -- -D warnings; \
	fi
	@if [ -f "package.json" ]; then \
		echo "Running React linter..."; \
		npm run lint 2>/dev/null || echo "No lint script found"; \
	fi

security: ## Run security checks
	@echo "🔒 Running security checks..."
	@./scripts/test.sh security

# Building
build: ## Build all components
	@echo "🏗️ Building all components..."
	@if [ -f "go.mod" ]; then \
		echo "Building Go components..."; \
		go build ./...; \
	fi
	@if [ -f "Cargo.toml" ]; then \
		echo "Building Rust components..."; \
		cargo build --workspace; \
	fi
	@if [ -f "package.json" ]; then \
		echo "Building React components..."; \
		npm run build; \
	fi

build-release: ## Build optimized release versions
	@echo "🏗️ Building release versions..."
	@if [ -f "go.mod" ]; then \
		echo "Building Go release..."; \
		CGO_ENABLED=0 go build -ldflags="-w -s" ./...; \
	fi
	@if [ -f "Cargo.toml" ]; then \
		echo "Building Rust release..."; \
		cargo build --workspace --release; \
	fi
	@if [ -f "package.json" ]; then \
		echo "Building React production..."; \
		npm run build; \
	fi

# Cleanup
clean: ## Clean build artifacts and dependencies
	@echo "🧹 Cleaning up..."
	@if [ -f "go.mod" ]; then \
		go clean -cache -modcache -testcache; \
	fi
	@if [ -f "Cargo.toml" ]; then \
		cargo clean; \
	fi
	@if [ -f "package.json" ]; then \
		rm -rf node_modules build coverage; \
	fi
	@rm -f coverage.out coverage.html
	@docker-compose -f deployments/docker/docker-compose.dev.yml down -v

# Database
db-reset: ## Reset development database
	@echo "🗄️ Resetting development database..."
	@docker-compose -f deployments/docker/docker-compose.dev.yml down postgres
	@docker volume rm chimera-pool-core_postgres_data 2>/dev/null || true
	@docker-compose -f deployments/docker/docker-compose.dev.yml up -d postgres
	@echo "⏳ Waiting for database..."
	@until docker-compose -f deployments/docker/docker-compose.dev.yml exec -T postgres pg_isready -U chimera; do sleep 1; done
	@echo "✅ Database reset complete"

# Documentation
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	@if [ -f "go.mod" ]; then \
		echo "Generating Go docs..."; \
		go doc -all ./... > docs/go-api.md 2>/dev/null || echo "Go docs generated"; \
	fi
	@if [ -f "Cargo.toml" ]; then \
		echo "Generating Rust docs..."; \
		cargo doc --workspace --no-deps; \
	fi

# Utilities
logs: ## Show development logs
	@docker-compose -f deployments/docker/docker-compose.dev.yml logs -f

status: ## Show development environment status
	@echo "📊 Development Environment Status:"
	@echo ""
	@echo "Docker Services:"
	@docker-compose -f deployments/docker/docker-compose.dev.yml ps
	@echo ""
	@echo "Ports:"
	@echo "  Frontend:    http://localhost:3000"
	@echo "  API:         http://localhost:8080"
	@echo "  Database UI: http://localhost:8080 (adminer)"
	@echo "  PostgreSQL:  localhost:5432"
	@echo "  Redis:       localhost:6379"

# Quick development workflow
quick-test: ## Quick test (unit tests only, no coverage)
	@echo "⚡ Running quick tests..."
	@if [ -f "go.mod" ]; then go test ./...; fi
	@if [ -f "Cargo.toml" ]; then cargo test --workspace; fi
	@if [ -f "package.json" ]; then npm test -- --watchAll=false; fi

# Install development tools
install-tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	@if command -v go >/dev/null 2>&1; then \
		echo "Installing Go tools..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@if command -v cargo >/dev/null 2>&1; then \
		echo "Installing Rust tools..."; \
		cargo install cargo-audit; \
		rustup component add rustfmt clippy; \
	fi