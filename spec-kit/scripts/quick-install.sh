#!/bin/bash

# Chimera Mining Pool - Quick Installation Script
# This script provides a one-click installation experience for the Chimera Mining Pool

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REPO_URL="https://github.com/your-org/chimera-mining-pool.git"
INSTALL_DIR="$HOME/chimera-mining-pool"
LOG_FILE="/tmp/chimera-install.log"

# Function to print colored output
print_header() {
    echo -e "${PURPLE}========================================${NC}"
    echo -e "${PURPLE}  $1${NC}"
    echo -e "${PURPLE}========================================${NC}"
    echo ""
}

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

print_step() {
    echo -e "${CYAN}▶${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to detect OS
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command_exists lsb_release; then
            OS=$(lsb_release -si)
            VERSION=$(lsb_release -sr)
        elif [ -f /etc/os-release ]; then
            . /etc/os-release
            OS=$NAME
            VERSION=$VERSION_ID
        else
            OS="Linux"
            VERSION="Unknown"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="macOS"
        VERSION=$(sw_vers -productVersion)
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        OS="Windows"
        VERSION="Unknown"
    else
        OS="Unknown"
        VERSION="Unknown"
    fi
}

# Function to check system requirements
check_requirements() {
    print_step "Checking system requirements..."
    
    # Check CPU cores
    if command_exists nproc; then
        CPU_CORES=$(nproc)
    elif command_exists sysctl; then
        CPU_CORES=$(sysctl -n hw.ncpu)
    else
        CPU_CORES=1
    fi
    
    if [ "$CPU_CORES" -lt 4 ]; then
        print_warning "Minimum 4 CPU cores recommended (found: $CPU_CORES)"
    else
        print_success "CPU cores: $CPU_CORES ✓"
    fi
    
    # Check memory
    if command_exists free; then
        MEMORY_GB=$(free -g | awk '/^Mem:/{print $2}')
    elif command_exists vm_stat; then
        MEMORY_BYTES=$(sysctl -n hw.memsize)
        MEMORY_GB=$((MEMORY_BYTES / 1024 / 1024 / 1024))
    else
        MEMORY_GB=0
    fi
    
    if [ "$MEMORY_GB" -lt 8 ]; then
        print_warning "Minimum 8GB RAM recommended (found: ${MEMORY_GB}GB)"
    else
        print_success "Memory: ${MEMORY_GB}GB ✓"
    fi
    
    # Check disk space
    DISK_SPACE=$(df -BG . | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$DISK_SPACE" -lt 100 ]; then
        print_warning "Minimum 100GB disk space recommended (available: ${DISK_SPACE}GB)"
    else
        print_success "Disk space: ${DISK_SPACE}GB available ✓"
    fi
}

# Function to install dependencies
install_dependencies() {
    print_step "Installing dependencies..."
    
    case "$OS" in
        "Ubuntu"|"Debian"*)
            sudo apt-get update
            sudo apt-get install -y curl wget git unzip
            ;;
        "CentOS"|"Red Hat"*|"Fedora"*)
            if command_exists dnf; then
                sudo dnf install -y curl wget git unzip
            else
                sudo yum install -y curl wget git unzip
            fi
            ;;
        "macOS")
            if ! command_exists brew; then
                print_status "Installing Homebrew..."
                /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
            fi
            brew install curl wget git
            ;;
        *)
            print_warning "Unknown OS. Please ensure curl, wget, and git are installed."
            ;;
    esac
}

# Function to install Docker
install_docker() {
    if command_exists docker; then
        print_success "Docker is already installed"
        DOCKER_VERSION=$(docker --version | cut -d' ' -f3 | cut -d',' -f1)
        print_status "Docker version: $DOCKER_VERSION"
        return
    fi
    
    print_step "Installing Docker..."
    
    case "$OS" in
        "Ubuntu"|"Debian"*)
            # Install Docker using official script
            curl -fsSL https://get.docker.com -o get-docker.sh
            sudo sh get-docker.sh
            sudo usermod -aG docker $USER
            rm get-docker.sh
            ;;
        "CentOS"|"Red Hat"*|"Fedora"*)
            curl -fsSL https://get.docker.com -o get-docker.sh
            sudo sh get-docker.sh
            sudo systemctl start docker
            sudo systemctl enable docker
            sudo usermod -aG docker $USER
            rm get-docker.sh
            ;;
        "macOS")
            print_status "Please install Docker Desktop from: https://www.docker.com/products/docker-desktop"
            print_status "Press Enter after Docker Desktop is installed and running..."
            read -r
            ;;
        *)
            print_error "Automatic Docker installation not supported for $OS"
            print_status "Please install Docker manually: https://docs.docker.com/get-docker/"
            exit 1
            ;;
    esac
    
    print_success "Docker installed successfully"
}

# Function to install Docker Compose
install_docker_compose() {
    if command_exists docker-compose || docker compose version >/dev/null 2>&1; then
        print_success "Docker Compose is already available"
        return
    fi
    
    print_step "Installing Docker Compose..."
    
    # Try to install Docker Compose plugin first
    if command_exists docker; then
        if docker compose version >/dev/null 2>&1; then
            print_success "Docker Compose plugin is available"
            return
        fi
    fi
    
    # Install standalone Docker Compose
    COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
    
    case "$OS" in
        "Linux"*|"Ubuntu"|"Debian"*|"CentOS"|"Red Hat"*|"Fedora"*)
            sudo curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
            sudo chmod +x /usr/local/bin/docker-compose
            ;;
        "macOS")
            curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
            chmod +x /usr/local/bin/docker-compose
            ;;
    esac
    
    print_success "Docker Compose installed successfully"
}

# Function to clone repository
clone_repository() {
    print_step "Cloning Chimera Mining Pool repository..."
    
    if [ -d "$INSTALL_DIR" ]; then
        print_status "Directory already exists. Updating..."
        cd "$INSTALL_DIR"
        git pull origin main
    else
        git clone "$REPO_URL" "$INSTALL_DIR"
        cd "$INSTALL_DIR/chimera-pool-core"
    fi
    
    print_success "Repository cloned successfully"
}

# Function to generate configuration
generate_config() {
    print_step "Generating configuration..."
    
    # Copy example environment file
    if [ ! -f .env ]; then
        cp .env.example .env
        print_success "Environment file created"
    fi
    
    # Generate secure secrets
    print_status "Generating secure secrets..."
    
    # Generate JWT secret (64 characters)
    JWT_SECRET=$(openssl rand -hex 32)
    
    # Generate encryption key (32 characters)
    ENCRYPTION_KEY=$(openssl rand -hex 16)
    
    # Generate database password
    DB_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    
    # Generate admin password
    ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-12)
    
    # Update .env file
    sed -i.bak "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env
    sed -i.bak "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env
    sed -i.bak "s/POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$DB_PASSWORD/" .env
    sed -i.bak "s/ADMIN_PASSWORD=.*/ADMIN_PASSWORD=$ADMIN_PASSWORD/" .env
    
    # Update database URL
    sed -i.bak "s|DATABASE_URL=.*|DATABASE_URL=postgres://chimera:$DB_PASSWORD@postgres:5432/chimera_pool?sslmode=disable|" .env
    
    rm .env.bak
    
    print_success "Configuration generated successfully"
    
    # Save credentials for user
    cat > install-credentials.txt << EOF
Chimera Mining Pool Installation Credentials
==========================================

Database Password: $DB_PASSWORD
Admin Password: $ADMIN_PASSWORD
JWT Secret: $JWT_SECRET
Encryption Key: $ENCRYPTION_KEY

Web Dashboard: http://localhost:3000
Admin Email: admin@chimera-pool.com
Admin Password: $ADMIN_PASSWORD

API Endpoint: http://localhost:8080
Stratum Server: localhost:18332

IMPORTANT: Save these credentials securely!
EOF
    
    print_success "Credentials saved to install-credentials.txt"
}

# Function to start services
start_services() {
    print_step "Starting Chimera Mining Pool services..."
    
    # Use docker-compose or docker compose based on availability
    if command_exists docker-compose; then
        COMPOSE_CMD="docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    else
        print_error "Docker Compose not found"
        exit 1
    fi
    
    # Pull latest images
    print_status "Pulling Docker images..."
    $COMPOSE_CMD pull
    
    # Start services
    print_status "Starting services..."
    $COMPOSE_CMD up -d
    
    print_success "Services started successfully"
}

# Function to wait for services
wait_for_services() {
    print_step "Waiting for services to be ready..."
    
    # Wait for database
    print_status "Waiting for database..."
    for i in {1..60}; do
        if docker exec chimera-pool-core_postgres_1 pg_isready -U chimera >/dev/null 2>&1; then
            break
        fi
        if [ $i -eq 60 ]; then
            print_error "Database failed to start"
            exit 1
        fi
        sleep 2
    done
    print_success "Database is ready"
    
    # Wait for API server
    print_status "Waiting for API server..."
    for i in {1..60}; do
        if curl -f http://localhost:8080/health >/dev/null 2>&1; then
            break
        fi
        if [ $i -eq 60 ]; then
            print_error "API server failed to start"
            exit 1
        fi
        sleep 2
    done
    print_success "API server is ready"
    
    # Wait for web interface
    print_status "Waiting for web interface..."
    for i in {1..60}; do
        if curl -f http://localhost:3000 >/dev/null 2>&1; then
            break
        fi
        if [ $i -eq 60 ]; then
            print_warning "Web interface may not be ready yet"
            break
        fi
        sleep 2
    done
    print_success "Web interface is ready"
}

# Function to initialize database
initialize_database() {
    print_step "Initializing database..."
    
    # Run database migrations
    if [ -f scripts/init-database.sh ]; then
        ./scripts/init-database.sh
    else
        print_status "Running database migrations via Docker..."
        docker exec chimera-pool-core_api_1 /app/migrate up
    fi
    
    print_success "Database initialized successfully"
}

# Function to run post-install tests
run_tests() {
    print_step "Running post-installation tests..."
    
    # Test API health
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        print_success "API health check: ✓"
    else
        print_error "API health check: ✗"
    fi
    
    # Test Stratum server
    if nc -z localhost 18332 >/dev/null 2>&1; then
        print_success "Stratum server: ✓"
    else
        print_error "Stratum server: ✗"
    fi
    
    # Test web interface
    if curl -f http://localhost:3000 >/dev/null 2>&1; then
        print_success "Web interface: ✓"
    else
        print_warning "Web interface: May still be starting"
    fi
    
    # Test database
    if docker exec chimera-pool-core_postgres_1 pg_isready -U chimera >/dev/null 2>&1; then
        print_success "Database: ✓"
    else
        print_error "Database: ✗"
    fi
}

# Function to display final information
show_completion_info() {
    print_header "Installation Complete!"
    
    echo -e "${GREEN}🎉 Chimera Mining Pool has been successfully installed!${NC}"
    echo ""
    echo -e "${CYAN}Access Information:${NC}"
    echo -e "  📊 Web Dashboard: ${YELLOW}http://localhost:3000${NC}"
    echo -e "  🔌 API Endpoint:  ${YELLOW}http://localhost:8080${NC}"
    echo -e "  ⛏️  Stratum Server: ${YELLOW}localhost:18332${NC}"
    echo -e "  📈 Monitoring:    ${YELLOW}http://localhost:3000/grafana${NC}"
    echo ""
    echo -e "${CYAN}Admin Credentials:${NC}"
    echo -e "  📧 Email:    ${YELLOW}admin@chimera-pool.com${NC}"
    echo -e "  🔑 Password: ${YELLOW}$(grep ADMIN_PASSWORD .env | cut -d'=' -f2)${NC}"
    echo ""
    echo -e "${CYAN}Next Steps:${NC}"
    echo -e "  1. Visit the web dashboard to complete setup"
    echo -e "  2. Configure your mining algorithms"
    echo -e "  3. Set up monitoring and alerts"
    echo -e "  4. Invite miners to join your pool"
    echo ""
    echo -e "${CYAN}Useful Commands:${NC}"
    echo -e "  📋 View logs:     ${YELLOW}docker-compose logs -f${NC}"
    echo -e "  🔄 Restart:       ${YELLOW}docker-compose restart${NC}"
    echo -e "  🛑 Stop:          ${YELLOW}docker-compose down${NC}"
    echo -e "  📊 Status:        ${YELLOW}docker-compose ps${NC}"
    echo ""
    echo -e "${CYAN}Documentation:${NC}"
    echo -e "  📖 Full docs:     ${YELLOW}https://docs.chimera-pool.com${NC}"
    echo -e "  💬 Community:     ${YELLOW}https://discord.gg/chimera-pool${NC}"
    echo -e "  🐛 Issues:        ${YELLOW}https://github.com/your-org/chimera-mining-pool/issues${NC}"
    echo ""
    echo -e "${GREEN}Happy mining! 🚀${NC}"
    echo ""
    print_warning "IMPORTANT: Save the credentials from install-credentials.txt in a secure location!"
}

# Main installation function
main() {
    # Clear screen and show header
    clear
    print_header "Chimera Mining Pool - Quick Installer"
    
    echo -e "${CYAN}Welcome to the Chimera Mining Pool installer!${NC}"
    echo -e "This script will install and configure your mining pool automatically."
    echo ""
    
    # Detect system
    detect_os
    print_status "Detected OS: $OS $VERSION"
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        print_warning "Running as root. This is not recommended for security reasons."
        echo "Continue anyway? (y/N)"
        read -r response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Confirm installation
    echo -e "${YELLOW}This will install Chimera Mining Pool to: $INSTALL_DIR${NC}"
    echo "Continue with installation? (Y/n)"
    read -r response
    if [[ "$response" =~ ^[Nn]$ ]]; then
        echo "Installation cancelled."
        exit 0
    fi
    
    # Start installation
    echo "" | tee "$LOG_FILE"
    print_status "Starting installation..." | tee -a "$LOG_FILE"
    
    # Installation steps
    check_requirements 2>&1 | tee -a "$LOG_FILE"
    install_dependencies 2>&1 | tee -a "$LOG_FILE"
    install_docker 2>&1 | tee -a "$LOG_FILE"
    install_docker_compose 2>&1 | tee -a "$LOG_FILE"
    clone_repository 2>&1 | tee -a "$LOG_FILE"
    generate_config 2>&1 | tee -a "$LOG_FILE"
    start_services 2>&1 | tee -a "$LOG_FILE"
    wait_for_services 2>&1 | tee -a "$LOG_FILE"
    initialize_database 2>&1 | tee -a "$LOG_FILE"
    run_tests 2>&1 | tee -a "$LOG_FILE"
    
    # Show completion information
    show_completion_info
    
    print_status "Installation log saved to: $LOG_FILE"
}

# Error handling
trap 'print_error "Installation failed. Check $LOG_FILE for details."; exit 1' ERR

# Run main installation
main "$@"