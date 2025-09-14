#!/bin/bash
#
# AWS SLURM Bursting Budget Installation Script
# 
# This script installs the aws-slurm-bursting-budget system on a SLURM cluster
# with support for various Linux distributions and configurations.
#

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
INSTALL_LOG="/tmp/asbb-install-$(date +%Y%m%d-%H%M%S).log"

# Default configuration
DEFAULT_INSTALL_PREFIX="/usr/local"
DEFAULT_CONFIG_DIR="/etc/asbb"
DEFAULT_SERVICE_DIR="/etc/systemd/system"
DEFAULT_SLURM_PLUGIN_DIR="/usr/lib64/slurm"
DEFAULT_DB_NAME="asbb"
DEFAULT_DB_USER="asbb"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Installation options
INSTALL_PREFIX="${INSTALL_PREFIX:-$DEFAULT_INSTALL_PREFIX}"
CONFIG_DIR="${CONFIG_DIR:-$DEFAULT_CONFIG_DIR}"
SERVICE_DIR="${SERVICE_DIR:-$DEFAULT_SERVICE_DIR}"
SLURM_PLUGIN_DIR="${SLURM_PLUGIN_DIR:-$DEFAULT_SLURM_PLUGIN_DIR}"
DB_NAME="${DB_NAME:-$DEFAULT_DB_NAME}"
DB_USER="${DB_USER:-$DEFAULT_DB_USER}"
DB_PASSWORD="${DB_PASSWORD:-}"
ADVISOR_URL="${ADVISOR_URL:-http://localhost:8081}"
SKIP_DB_SETUP="${SKIP_DB_SETUP:-false}"
SKIP_SLURM_CONFIG="${SKIP_SLURM_CONFIG:-false}"
DRY_RUN="${DRY_RUN:-false}"
FORCE="${FORCE:-false}"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$INSTALL_LOG"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$INSTALL_LOG"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$INSTALL_LOG"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$INSTALL_LOG"
}

# Function to show help
show_help() {
    cat << EOF
AWS SLURM Bursting Budget Installation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --install-prefix PATH      Installation prefix (default: $DEFAULT_INSTALL_PREFIX)
    --config-dir PATH          Configuration directory (default: $DEFAULT_CONFIG_DIR)
    --slurm-plugin-dir PATH    SLURM plugin directory (default: $DEFAULT_SLURM_PLUGIN_DIR)
    --db-name NAME             Database name (default: $DEFAULT_DB_NAME)
    --db-user USER             Database user (default: $DEFAULT_DB_USER)
    --db-password PASS         Database password (will prompt if not provided)
    --advisor-url URL          Advisor service URL (default: $ADVISOR_URL)
    --skip-db-setup            Skip database setup
    --skip-slurm-config        Skip SLURM configuration
    --dry-run                  Show what would be done without making changes
    --force                    Force installation even if components exist
    --help                     Show this help message

ENVIRONMENT VARIABLES:
    All options can be set via environment variables:
    INSTALL_PREFIX, CONFIG_DIR, SLURM_PLUGIN_DIR, DB_NAME, DB_USER, 
    DB_PASSWORD, ADVISOR_URL, SKIP_DB_SETUP, SKIP_SLURM_CONFIG

EXAMPLES:
    # Basic installation
    sudo $0

    # Custom installation paths
    sudo $0 --install-prefix /opt/asbb --config-dir /opt/asbb/etc

    # Skip database setup (for existing database)
    sudo $0 --skip-db-setup --db-password existing_password

    # Dry run to see what would be installed
    sudo $0 --dry-run

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --install-prefix)
            INSTALL_PREFIX="$2"
            shift 2
            ;;
        --config-dir)
            CONFIG_DIR="$2"
            shift 2
            ;;
        --slurm-plugin-dir)
            SLURM_PLUGIN_DIR="$2"
            shift 2
            ;;
        --db-name)
            DB_NAME="$2"
            shift 2
            ;;
        --db-user)
            DB_USER="$2"
            shift 2
            ;;
        --db-password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        --advisor-url)
            ADVISOR_URL="$2"
            shift 2
            ;;
        --skip-db-setup)
            SKIP_DB_SETUP="true"
            shift
            ;;
        --skip-slurm-config)
            SKIP_SLURM_CONFIG="true"
            shift
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --force)
            FORCE="true"
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Function to detect OS and distribution
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    elif type lsb_release >/dev/null 2>&1; then
        OS=$(lsb_release -si)
        VER=$(lsb_release -sr)
    elif [[ -f /etc/redhat-release ]]; then
        OS="Red Hat Enterprise Linux"
        VER=$(awk '{print $4}' /etc/redhat-release | cut -d. -f1-2)
    else
        OS=$(uname -s)
        VER=$(uname -r)
    fi
    
    print_status "Detected OS: $OS $VER"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    local missing_deps=()
    
    # Check for required commands
    local required_commands=("curl" "gcc" "make" "pkg-config" "systemctl")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done
    
    # Check for SLURM installation
    if ! command -v scontrol >/dev/null 2>&1; then
        print_error "SLURM is not installed or not in PATH"
        missing_deps+=("slurm")
    fi
    
    # Check for Go installation (for building from source)
    if ! command -v go >/dev/null 2>&1; then
        print_warning "Go is not installed - will try to use pre-built binaries"
    fi
    
    # Check for required libraries
    local required_libs=("libcurl" "libjson-c")
    for lib in "${required_libs[@]}"; do
        if ! pkg-config --exists "$lib" 2>/dev/null; then
            missing_deps+=("$lib-dev")
        fi
    done
    
    # Check database
    if [[ "$SKIP_DB_SETUP" != "true" ]]; then
        if ! command -v psql >/dev/null 2>&1; then
            print_warning "PostgreSQL client not found - database setup may fail"
        fi
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        print_error "Missing dependencies: ${missing_deps[*]}"
        print_status "Install missing dependencies and try again"
        suggest_package_install "${missing_deps[@]}"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to suggest package installation commands
suggest_package_install() {
    local deps=("$@")
    print_status "To install missing dependencies, try:"
    
    case "$OS" in
        *"Ubuntu"*|*"Debian"*)
            echo "  sudo apt-get update"
            echo "  sudo apt-get install build-essential libcurl4-openssl-dev libjson-c-dev slurm-wlm postgresql-client"
            ;;
        *"CentOS"*|*"Red Hat"*|*"Rocky"*|*"AlmaLinux"*)
            echo "  sudo yum install -y gcc make libcurl-devel json-c-devel slurm postgresql"
            # or: sudo dnf install -y ...
            ;;
        *"SUSE"*)
            echo "  sudo zypper install gcc make libcurl-devel libjson-c-devel slurm postgresql"
            ;;
        *)
            echo "  Please install: ${deps[*]}"
            ;;
    esac
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]] && [[ "$DRY_RUN" != "true" ]]; then
        print_error "This script must be run as root (or with sudo)"
        print_status "Try: sudo $0 $*"
        exit 1
    fi
}

# Function to create directories
create_directories() {
    print_status "Creating directories..."
    
    local dirs=(
        "$INSTALL_PREFIX/bin"
        "$INSTALL_PREFIX/sbin"
        "$CONFIG_DIR"
        "$CONFIG_DIR/examples"
        "/var/log/asbb"
        "/var/lib/asbb"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ "$DRY_RUN" == "true" ]]; then
            print_status "Would create directory: $dir"
        else
            mkdir -p "$dir"
            print_status "Created directory: $dir"
        fi
    done
}

# Function to create asbb user
create_user() {
    print_status "Creating asbb user..."
    
    if id "asbb" &>/dev/null; then
        print_status "User 'asbb' already exists"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            print_status "Would create user 'asbb'"
        else
            useradd -r -s /bin/false -d /var/lib/asbb -c "AWS SLURM Bursting Budget" asbb
            print_success "Created user 'asbb'"
        fi
    fi
    
    # Set ownership
    if [[ "$DRY_RUN" != "true" ]]; then
        chown -R asbb:asbb /var/log/asbb /var/lib/asbb
        chmod 755 /var/log/asbb /var/lib/asbb
    fi
}

# Function to build or download binaries
install_binaries() {
    print_status "Installing binaries..."
    
    local build_dir="$PROJECT_DIR/build"
    
    # Check if pre-built binaries exist
    if [[ -f "$build_dir/budget-service" ]] && [[ -f "$build_dir/asbb" ]]; then
        print_status "Using pre-built binaries from $build_dir"
    elif command -v go >/dev/null 2>&1; then
        print_status "Building binaries from source..."
        if [[ "$DRY_RUN" != "true" ]]; then
            cd "$PROJECT_DIR"
            make build
        fi
    else
        print_error "No pre-built binaries found and Go is not installed"
        print_status "Please build the project first or install Go"
        exit 1
    fi
    
    # Install binaries
    local binaries=(
        "budget-service:$INSTALL_PREFIX/sbin/"
        "asbb:$INSTALL_PREFIX/bin/"
        "budget-recovery:$INSTALL_PREFIX/sbin/"
    )
    
    for binary_info in "${binaries[@]}"; do
        local binary="${binary_info%:*}"
        local dest_dir="${binary_info#*:}"
        local source="$build_dir/$binary"
        local dest="$dest_dir$binary"
        
        if [[ -f "$source" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                print_status "Would install: $source -> $dest"
            else
                cp "$source" "$dest"
                chmod 755 "$dest"
                print_success "Installed: $dest"
            fi
        else
            print_error "Binary not found: $source"
            exit 1
        fi
    done
    
    # Create symlink for asbb alias
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "Would create symlink: $INSTALL_PREFIX/bin/asbb"
    else
        ln -sf "$INSTALL_PREFIX/bin/asbb" "$INSTALL_PREFIX/bin/asbb"
        print_success "Created asbb symlink"
    fi
}

# Function to install SLURM plugin
install_slurm_plugin() {
    print_status "Installing SLURM plugin..."
    
    local plugin_source="$PROJECT_DIR/build/job_submit_budget.so"
    local plugin_dest="$SLURM_PLUGIN_DIR/job_submit_budget.so"
    
    # Build plugin if it doesn't exist
    if [[ ! -f "$plugin_source" ]]; then
        if [[ "$DRY_RUN" != "true" ]]; then
            cd "$PROJECT_DIR"
            make build-plugin
        fi
    fi
    
    if [[ -f "$plugin_source" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            print_status "Would install SLURM plugin: $plugin_dest"
        else
            mkdir -p "$SLURM_PLUGIN_DIR"
            cp "$plugin_source" "$plugin_dest"
            chmod 755 "$plugin_dest"
            print_success "Installed SLURM plugin: $plugin_dest"
        fi
    else
        print_error "SLURM plugin not found: $plugin_source"
        print_status "Run 'make build-plugin' in the project directory first"
        exit 1
    fi
}

# Function to generate configuration
generate_config() {
    print_status "Generating configuration..."
    
    local config_file="$CONFIG_DIR/config.yaml"
    
    if [[ -f "$config_file" ]] && [[ "$FORCE" != "true" ]]; then
        print_warning "Configuration file exists: $config_file"
        print_status "Use --force to overwrite or edit manually"
        return 0
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "Would generate configuration: $config_file"
        return 0
    fi
    
    # Generate database password if not provided
    if [[ -z "$DB_PASSWORD" ]] && [[ "$SKIP_DB_SETUP" != "true" ]]; then
        DB_PASSWORD=$(openssl rand -base64 32 | tr -d '=/+' | cut -c1-25)
        print_status "Generated database password"
    fi
    
    cat > "$config_file" << EOF
# AWS SLURM Bursting Budget Configuration
# Generated on $(date)

service:
  listen_addr: ":8080"
  read_timeout: "30s"
  write_timeout: "30s"
  shutdown_timeout: "10s"
  enable_metrics: true
  metrics_addr: ":9090"

database:
  driver: "postgres"
  dsn: "postgresql://$DB_USER:$DB_PASSWORD@localhost:5432/$DB_NAME?sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: "5m"
  migrations_path: "/var/lib/asbb/migrations"
  auto_migrate: false

advisor:
  url: "$ADVISOR_URL"
  timeout: "30s"
  retry_attempts: 3
  retry_delay: "1s"
  enable_caching: true
  cache_ttl: "5m"

budget:
  default_hold_percentage: 1.2
  max_hold_percentage: 1.5
  reconciliation_timeout: "24h"
  max_hold_duration: "168h"
  enable_partition_limits: false
  allow_negative_balances: false
  auto_reconciliation: true
  reconciliation_interval: "1h"
  cost_tolerance_percentage: 2.0

slurm:
  bin_path: "/usr/bin"
  plugin_path: "$SLURM_PLUGIN_DIR"
  job_completion_timeout: "5m"
  enable_job_monitoring: true
  monitoring_interval: "30s"

security:
  enable_authentication: false
  enable_tls: false

logging:
  level: "info"
  format: "json"
  output: "stderr"
  enable_file: true
  file_path: "/var/log/asbb/asbb.log"
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true
EOF
    
    chmod 600 "$config_file"
    chown asbb:asbb "$config_file"
    print_success "Generated configuration: $config_file"
    
    # Copy example configurations
    if [[ -d "$PROJECT_DIR/configs" ]]; then
        cp -r "$PROJECT_DIR/configs"/* "$CONFIG_DIR/examples/"
        print_status "Copied example configurations to $CONFIG_DIR/examples/"
    fi
}

# Function to setup database
setup_database() {
    if [[ "$SKIP_DB_SETUP" == "true" ]]; then
        print_status "Skipping database setup"
        return 0
    fi
    
    print_status "Setting up database..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "Would setup database: $DB_NAME with user: $DB_USER"
        return 0
    fi
    
    # Check if PostgreSQL is running
    if ! systemctl is-active --quiet postgresql; then
        print_status "Starting PostgreSQL..."
        systemctl start postgresql
        systemctl enable postgresql
    fi
    
    # Create database user and database
    sudo -u postgres psql << EOF
CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
CREATE DATABASE $DB_NAME OWNER $DB_USER;
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
\q
EOF
    
    print_success "Database setup completed"
    
    # Copy migration files
    if [[ -d "$PROJECT_DIR/migrations" ]]; then
        cp -r "$PROJECT_DIR/migrations" /var/lib/asbb/
        chown -R asbb:asbb /var/lib/asbb/migrations
        print_status "Copied database migrations"
    fi
}

# Function to install systemd service
install_systemd_service() {
    print_status "Installing systemd service..."
    
    local service_file="$SERVICE_DIR/asbb-service.service"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "Would install systemd service: $service_file"
        return 0
    fi
    
    cat > "$service_file" << EOF
[Unit]
Description=AWS SLURM Bursting Budget Service
Documentation=https://github.com/scttfrdmn/aws-slurm-bursting-budget
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=asbb
Group=asbb
ExecStart=$INSTALL_PREFIX/sbin/budget-service
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
LimitNOFILE=65536

# Environment
Environment=ASBB_CONFIG_FILE=$CONFIG_DIR/config.yaml

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/asbb /var/lib/asbb

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    print_success "Installed systemd service"
    
    # Enable but don't start the service yet
    systemctl enable asbb-service
    print_status "Service enabled - start with: systemctl start asbb-service"
}

# Function to configure SLURM
configure_slurm() {
    if [[ "$SKIP_SLURM_CONFIG" == "true" ]]; then
        print_status "Skipping SLURM configuration"
        return 0
    fi
    
    print_status "Configuring SLURM..."
    
    local plugstack_conf="/etc/slurm/plugstack.conf"
    local budget_plugin_line="required $SLURM_PLUGIN_DIR/job_submit_budget.so budget_service_url=http://localhost:8080"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "Would add to $plugstack_conf:"
        print_status "  $budget_plugin_line"
        return 0
    fi
    
    # Create plugstack.conf if it doesn't exist
    if [[ ! -f "$plugstack_conf" ]]; then
        touch "$plugstack_conf"
        print_status "Created $plugstack_conf"
    fi
    
    # Check if plugin is already configured
    if grep -q "job_submit_budget.so" "$plugstack_conf"; then
        print_warning "Budget plugin already configured in $plugstack_conf"
    else
        echo "$budget_plugin_line" >> "$plugstack_conf"
        print_success "Added budget plugin to $plugstack_conf"
    fi
    
    # Add JobSubmitPlugins to slurm.conf if not present
    local slurm_conf="/etc/slurm/slurm.conf"
    if [[ -f "$slurm_conf" ]]; then
        if ! grep -q "JobSubmitPlugins.*budget" "$slurm_conf"; then
            if grep -q "JobSubmitPlugins" "$slurm_conf"; then
                # Add to existing line
                sed -i 's/JobSubmitPlugins=\(.*\)/JobSubmitPlugins=\1,budget/' "$slurm_conf"
            else
                # Add new line
                echo "JobSubmitPlugins=budget" >> "$slurm_conf"
            fi
            print_success "Added JobSubmitPlugins=budget to $slurm_conf"
            print_warning "SLURM daemon restart required: systemctl restart slurmctld"
        fi
    else
        print_warning "SLURM configuration file not found: $slurm_conf"
    fi
}

# Function to run database migrations
run_migrations() {
    print_status "Running database migrations..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_status "Would run database migrations"
        return 0
    fi
    
    # Wait for database to be ready
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if sudo -u asbb ASBB_CONFIG_FILE="$CONFIG_DIR/config.yaml" "$INSTALL_PREFIX/bin/asbb" database migrate; then
            print_success "Database migrations completed"
            return 0
        fi
        
        print_status "Database not ready, attempt $attempt/$max_attempts..."
        sleep 2
        ((attempt++))
    done
    
    print_error "Database migrations failed after $max_attempts attempts"
    return 1
}

# Function to validate installation
validate_installation() {
    print_status "Validating installation..."
    
    local errors=0
    
    # Check binaries
    local binaries=("$INSTALL_PREFIX/bin/asbb" "$INSTALL_PREFIX/sbin/budget-service")
    for binary in "${binaries[@]}"; do
        if [[ ! -x "$binary" ]]; then
            print_error "Binary not found or not executable: $binary"
            ((errors++))
        fi
    done
    
    # Check configuration
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        print_error "Configuration file not found: $CONFIG_DIR/config.yaml"
        ((errors++))
    fi
    
    # Check SLURM plugin
    if [[ ! -f "$SLURM_PLUGIN_DIR/job_submit_budget.so" ]]; then
        print_error "SLURM plugin not found: $SLURM_PLUGIN_DIR/job_submit_budget.so"
        ((errors++))
    fi
    
    # Check systemd service
    if [[ ! -f "$SERVICE_DIR/asbb-service.service" ]]; then
        print_error "Systemd service not found: $SERVICE_DIR/asbb-service.service"
        ((errors++))
    fi
    
    # Test configuration
    if [[ "$DRY_RUN" != "true" ]]; then
        if ! sudo -u asbb ASBB_CONFIG_FILE="$CONFIG_DIR/config.yaml" "$INSTALL_PREFIX/bin/asbb" config validate; then
            print_error "Configuration validation failed"
            ((errors++))
        fi
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "Installation validation passed"
        return 0
    else
        print_error "Installation validation failed with $errors errors"
        return 1
    fi
}

# Function to print installation summary
print_summary() {
    cat << EOF

================================================================================
AWS SLURM Bursting Budget Installation Complete
================================================================================

Installation Summary:
  Install Prefix: $INSTALL_PREFIX
  Configuration: $CONFIG_DIR/config.yaml
  SLURM Plugin: $SLURM_PLUGIN_DIR/job_submit_budget.so
  Log File: $INSTALL_LOG

Next Steps:
  1. Review and customize the configuration:
     vi $CONFIG_DIR/config.yaml

  2. Start the budget service:
     systemctl start asbb-service

  3. Check service status:
     systemctl status asbb-service

  4. Test the installation:
     $INSTALL_PREFIX/bin/asbb account list

  5. Restart SLURM controller to load the plugin:
     systemctl restart slurmctld

  6. Submit a test job to verify budget checking works

For more information, see the documentation at:
https://github.com/scttfrdmn/aws-slurm-bursting-budget

Installation log saved to: $INSTALL_LOG

EOF

    if [[ "$DB_PASSWORD" != "" ]] && [[ "$SKIP_DB_SETUP" != "true" ]]; then
        cat << EOF
Database Information:
  Database: $DB_NAME
  User: $DB_USER
  Password: $DB_PASSWORD

IMPORTANT: Save the database password securely!

EOF
    fi
}

# Main installation function
main() {
    print_status "Starting AWS SLURM Bursting Budget installation..."
    print_status "Log file: $INSTALL_LOG"
    
    # Pre-installation checks
    detect_os
    check_root
    check_prerequisites
    
    # Installation steps
    create_directories
    create_user
    install_binaries
    install_slurm_plugin
    generate_config
    setup_database
    install_systemd_service
    configure_slurm
    
    # Post-installation
    if [[ "$SKIP_DB_SETUP" != "true" ]]; then
        run_migrations
    fi
    
    validate_installation
    print_summary
    
    print_success "Installation completed successfully!"
}

# Error handling
trap 'print_error "Installation failed at line $LINENO"' ERR

# Run main function
main "$@"