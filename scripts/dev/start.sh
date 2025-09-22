#!/bin/bash
set -e

echo "🚀 Starting Chimera Pool development environment..."

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check if development environment is set up
check_setup() {
    if [ ! -f "go.mod" ] || [ ! -f "Cargo.toml" ] || [ ! -f "package.json" ]; then
        echo "❌ Development environment not set up. Run ./scripts/dev/setup.sh first"
        exit 1
    fi
}

# Start infrastructure services
start_infrastructure() {
    echo -e "${YELLOW}🗄️ Starting infrastructure services...${NC}"
    docker-compose -f deployments/docker/docker-compose.dev.yml up -d
    
    # Wait for services to be ready
    echo "⏳ Waiting for services to be ready..."
    until docker-compose -f deployments/docker/docker-compose.dev.yml exec -T postgres pg_isready -U chimera; do
        sleep 1
    done
    
    until docker-compose -f deployments/docker/docker-compose.dev.yml exec -T redis redis-cli ping | grep -q PONG; do
        sleep 1
    done
    
    echo -e "${GREEN}✅ Infrastructure services ready${NC}"
}

# Start backend services in development mode
start_backend() {
    echo -e "${YELLOW}🔧 Starting backend services...${NC}"
    
    # This will be implemented when we have the actual services
    echo "⏭️  Backend services will be started here when implemented"
    
    # Example of what this would look like:
    # go run src/pool-manager/cmd/main.go &
    # go run src/stratum-server/cmd/main.go &
    # go run src/auth-service/cmd/main.go &
}

# Start frontend development server
start_frontend() {
    echo -e "${YELLOW}🎨 Starting frontend development server...${NC}"
    
    # Start React development server in background
    npm start &
    FRONTEND_PID=$!
    
    echo -e "${GREEN}✅ Frontend server starting on http://localhost:3000${NC}"
}

# Display development URLs
show_urls() {
    echo -e "\n${BLUE}🌐 Development URLs:${NC}"
    echo "  Frontend:     http://localhost:3000"
    echo "  API:          http://localhost:8080"
    echo "  Database UI:  http://localhost:8080 (adminer)"
    echo "  PostgreSQL:   localhost:5432"
    echo "  Redis:        localhost:6379"
    echo ""
    echo -e "${YELLOW}📝 Logs:${NC}"
    echo "  View logs: docker-compose -f deployments/docker/docker-compose.dev.yml logs -f"
    echo ""
    echo -e "${YELLOW}🛑 Stop services:${NC}"
    echo "  Stop all: docker-compose -f deployments/docker/docker-compose.dev.yml down"
    echo "  Stop frontend: kill $FRONTEND_PID (if running)"
}

# Handle cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up...${NC}"
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
}

# Set up signal handlers
trap cleanup EXIT INT TERM

# Main execution
main() {
    check_setup
    start_infrastructure
    start_backend
    start_frontend
    show_urls
    
    echo -e "${GREEN}🎉 Development environment is running!${NC}"
    echo "Press Ctrl+C to stop all services"
    
    # Keep script running
    wait
}

main "$@"