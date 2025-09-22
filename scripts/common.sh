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
