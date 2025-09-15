# AWS SLURM Bursting Budget (ASBB)

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/aws-slurm-burst-budget)](https://goreportcard.com/report/github.com/scttfrdmn/aws-slurm-burst-budget)
[![CI/CD Pipeline](https://github.com/scttfrdmn/aws-slurm-burst-budget/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/aws-slurm-burst-budget/actions/workflows/ci.yml)
[![Coverage Status](https://codecov.io/gh/scttfrdmn/aws-slurm-burst-budget/branch/main/graph/badge.svg)](https://codecov.io/gh/scttfrdmn/aws-slurm-burst-budget)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/v/release/scttfrdmn/aws-slurm-burst-budget)](https://github.com/scttfrdmn/aws-slurm-burst-budget/releases)

**Enterprise-grade budget management system for HPC clusters** that burst workloads to AWS, providing real-money budget enforcement, multi-year grant tracking, and comprehensive cost analytics through SLURM integration.

> **Latest**: v0.1.2 adds comprehensive grant management, long-term burn rate analytics, and ASBX integration for complete academic research computing budget tracking.

## 🌟 Features

### 💰 Core Budget Management
- **Real-time Budget Enforcement**: Check budget availability at job submission time
- **Pre-allocation Model**: Hold estimated costs with configurable buffer, reconcile actual costs upon completion
- **Complete Audit Trail**: Track every budget operation with full transaction history
- **Account-based Budgets**: Map budget accounts to SLURM accounts with flexible limits
- **Partition-specific Limits**: Different budget constraints for CPU vs GPU partitions

### 🆕 Incremental Budget Allocation
- **Scheduled Allocations**: Automatically allocate budget over time (e.g., $600 total allocated at $100/month)
- **Flexible Frequencies**: Daily, weekly, monthly, quarterly, or yearly allocations
- **Automatic Processing**: Background service handles allocations automatically
- **Manual Control**: Override automatic allocations when needed

### 🎓 Grant Management & Long-term Tracking
- **Multi-year Grants**: Support for grants spanning months to years (e.g., 3-year NSF grants)
- **Budget Periods**: Annual or custom budget periods within grants
- **Burn Rate Analytics**: Track spending vs. expected rates over time
- **Compliance Reporting**: Generate agency-required financial reports
- **Performance Tracking**: Monitor if budgets are under/over their burn rates
- **Automated Alerts**: Real-time notifications for budget variances

### Advanced Features
- **Partition-specific Limits**: Different budget limits for CPU vs GPU partitions
- **Cost Estimation Integration**: Works with [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor)
- **Recovery System**: Automatic cleanup of orphaned transactions after outages
- **Comprehensive CLI**: Full command-line interface for budget administration
- **REST API**: Complete HTTP API for integration with external systems
- **Prometheus Metrics**: Built-in monitoring and alerting support

### 🔗 Advanced Integration
- **ASBX Integration**: Seamless integration with aws-slurm-burst v0.2.0 for automatic cost reconciliation
- **ASBA Integration**: Budget-aware decision making APIs for Academic Slurm Burst Allocation
- **Cost Model Learning**: Performance feedback loops to improve estimation accuracy
- **Ecosystem Coordination**: Complete ASBA + ASBX + ASBB workflow integration
- **SLURM Epilog Integration**: Post-job processing with automatic cost tracking
- **Intelligent Decision Making**: Grant timeline optimization and deadline-driven resource allocation

### 🏆 Enterprise Quality
- **Go Report Card Grade A**: Maintains highest code quality standards
- **Production Security**: Complete input validation, audit logging, authentication
- **Cross-platform**: Linux, macOS, Windows binaries via GoReleaser
- **Container Ready**: Docker and Kubernetes deployment manifests
- **Monitoring**: Health checks, metrics collection, alerting integration

## 🏗️ Architecture

```
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ SLURM Daemon    │ │ Budget Service  │ │ Burst Advisor   │
│                 │ │                 │ │                 │
│ ┌─────────────┐ │ │ ┌─────────────┐ │ │ ┌─────────────┐ │
│ │Submit Plugin│ │ │ │Budget Check │ │ │ │Cost Estimate│ │
│ └─────────────┘ │ │ └─────────────┘ │ │ └─────────────┘ │
└─────────────────┘ └─────────────────┘ └─────────────────┘
         │                     │                     │
         └─────────────────────┼─────────────────────┘
                               │
                    ┌─────────────────┐
                    │   Database      │
                    │                 │
                    │ ┌─────────────┐ │
                    │ │Budget Accts │ │
                    │ ├─────────────┤ │
                    │ │Transactions │ │
                    │ ├─────────────┤ │
                    │ │Allocations  │ │
                    │ └─────────────┘ │
                    └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 13+ (or MySQL 8.0+)
- SLURM 22.05+ with development headers
- Access to [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor) service

### Installation

#### Option 1: Download Pre-built Binaries (Recommended)

```bash
# Download latest release for your platform
curl -L https://github.com/scttfrdmn/aws-slurm-burst-budget/releases/latest/download/aws-slurm-burst-budget_Linux_x86_64.tar.gz | tar xz

# Install binaries
sudo mv asbb budget-service recovery /usr/local/bin/

# Install SLURM plugin (requires SLURM development headers)
sudo make install-plugin
```

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/scttfrdmn/aws-slurm-burst-budget.git
cd aws-slurm-burst-budget

# Build all components
make build

# Install system-wide
sudo make install

# Install SLURM plugin
sudo make install-plugin
```

#### Option 3: Homebrew (macOS/Linux)

```bash
# Add tap and install
brew tap scttfrdmn/tap
brew install aws-slurm-burst-budget
```

### Database Setup

```bash
# Create database
createdb asbb

# Run migrations
./build/asbb database migrate
```

### Configuration

```bash
# Generate default configuration
./build/asbb config init --output /etc/asbb/config.yaml

# Edit configuration
sudo vi /etc/asbb/config.yaml
```

### Basic Usage

1. **Create a Budget Account with Incremental Allocation**:
```bash
# Create account with $600 total budget, allocated $100/month
asbb account create \
  --name="ML Research Project" \
  --account=ml-proj-001 \
  --incremental \
  --total-budget=600.00 \
  --allocation-amount=100.00 \
  --allocation-frequency=monthly \
  --start=2025-01-01 \
  --end=2025-12-31
```

2. **Configure SLURM Integration**:
```bash
# Add to /etc/slurm/slurm.conf
JobSubmitPlugins=budget

# Add to /etc/slurm/plugstack.conf
required job_submit_budget.so budget_service_url=http://localhost:8080
```

3. **Start the Service**:
```bash
# Start budget service
systemctl start asbb-service

# Or run directly
./build/budget-service
```

4. **Submit Jobs** (automatic budget checking):
```bash
sbatch --account=ml-proj-001 --partition=gpu-aws my_job.sbatch
```

5. **Monitor Usage**:
```bash
# View account status with allocation schedule
asbb account show ml-proj-001

# View allocation history
asbb allocations list --account=ml-proj-001

# View transaction history
asbb transactions list --account=ml-proj-001 --last=30d
```

## 💰 Incremental Budget Allocation Examples

### Example 1: Monthly Research Budget
```bash
# $1200 annual budget, $100 per month
asbb account create \
  --name="Research Project Alpha" \
  --account=research-alpha \
  --incremental \
  --total-budget=1200.00 \
  --allocation-amount=100.00 \
  --allocation-frequency=monthly \
  --start=2025-01-01 \
  --end=2025-12-31
```

### Example 2: Weekly Compute Allowance
```bash
# $520 annual budget, $10 per week
asbb account create \
  --name="Student Compute" \
  --account=student-cs101 \
  --incremental \
  --total-budget=520.00 \
  --allocation-amount=10.00 \
  --allocation-frequency=weekly \
  --start=2025-01-01 \
  --end=2025-12-31
```

### Example 3: Quarterly Department Budget
```bash
# $4000 annual budget, $1000 per quarter
asbb account create \
  --name="CS Department" \
  --account=cs-dept \
  --incremental \
  --total-budget=4000.00 \
  --allocation-amount=1000.00 \
  --allocation-frequency=quarterly \
  --start=2025-01-01 \
  --end=2025-12-31
```

## 🎓 Grant Management Examples

### Example 1: Multi-year NSF Grant
```bash
# 3-year NSF grant with $750K total funding
asbb grant create \
  --number=NSF-2025-12345 \
  --agency="National Science Foundation" \
  --program="Computer and Information Science and Engineering" \
  --pi="Dr. Jane Smith" \
  --co-pi="Dr. Bob Johnson,Dr. Alice Chen" \
  --institution="Research University" \
  --department="Computer Science" \
  --amount=750000 \
  --start=2025-01-01 \
  --end=2027-12-31 \
  --periods=12 \
  --indirect=0.30 \
  --federal-id="47.070"
```

### Example 2: NIH Medical Research Grant
```bash
# 5-year NIH grant with annual budget periods
asbb grant create \
  --number=NIH-R01-567890 \
  --agency="National Institutes of Health" \
  --program="National Cancer Institute" \
  --pi="Dr. Medical Researcher" \
  --institution="Medical University" \
  --amount=1250000 \
  --start=2025-07-01 \
  --end=2030-06-30 \
  --periods=12 \
  --indirect=0.25
```

### Example 3: DOE Energy Research Grant
```bash
# 2-year Department of Energy grant
asbb grant create \
  --number=DOE-DE-SC-98765 \
  --agency="Department of Energy" \
  --program="Office of Science" \
  --pi="Dr. Energy Expert" \
  --amount=500000 \
  --start=2025-10-01 \
  --end=2027-09-30 \
  --periods=6 \
  --indirect=0.35
```

## 📊 Burn Rate Analysis Examples

### Example 1: Monitor Grant Spending
```bash
# Analyze burn rate for NSF grant over last 3 months
asbb burn-rate NSF-2025-12345 --period=90d --projection

# Quick check for budget alerts
asbb burn-rate NSF-2025-12345 --alerts-only

# Monthly burn rate analysis for all active grants
asbb grant list --active | xargs -I {} asbb burn-rate {} --period=30d
```

### Example 2: Budget Performance Tracking
```bash
# Check if account is overspending
asbb burn-rate ml-research-001 --period=30d
# Output: Daily Rate: $156.43 (Expected: $125.00)
#         Variance: +25.1% (OVERSPENDING)
#         Health Score: 67% (CONCERN)

# Get detailed historical analysis
asbb burn-rate ml-research-001 --period=6m --projection
# Shows 6-month spending trends and projects end-of-grant status
```

## 🔧 CLI Commands

### Account Management
```bash
asbb account list                    # List all budget accounts
asbb account create [options]       # Create new budget account
asbb account show <account>         # Show account details & allocation schedule
asbb account update <account>       # Update account settings
asbb account delete <account>       # Delete account
```

### Grant Management (Long-term Funding)
```bash
asbb grant create [options]         # Create multi-year research grant
asbb grant list                     # List all grants with status
asbb grant show <grant-number>      # Show grant details with burn rate analysis
asbb grant report <grant-number>    # Generate compliance reports
asbb grant periods <grant-number>   # List budget periods for grant
```

### Burn Rate Analytics
```bash
asbb burn-rate <account|grant>      # Analyze spending patterns and projections
asbb burn-rate <target> --period=90d # Historical analysis (7d, 30d, 90d, 6m, 1y)
asbb burn-rate <target> --projection # Include spending projections
asbb burn-rate <target> --alerts-only # Show only active alerts
```

### Allocation Management
```bash
asbb allocations list               # List allocation schedules
asbb allocations show <id>          # Show allocation schedule details
asbb allocations process            # Manually process pending allocations
asbb allocations pause <id>         # Pause allocation schedule
asbb allocations resume <id>        # Resume allocation schedule
```

### Usage Monitoring
```bash
asbb usage show <account>           # Show current usage
asbb usage summary                  # System-wide usage summary
asbb forecast <account>             # Burn rate analysis
```

### Transaction Management
```bash
asbb transactions list              # View transaction history
asbb transactions show <id>         # Show transaction details
asbb reconcile <job_id>             # Manual job reconciliation
asbb recover                        # Cleanup orphaned transactions
```

## 🌐 REST API

The budget service provides a comprehensive REST API:

### Budget Operations
- `POST /api/v1/budget/check` - Check budget availability (used by SLURM plugin)
- `POST /api/v1/budget/reconcile` - Reconcile job costs

### Account Management
- `GET /api/v1/accounts` - List accounts
- `POST /api/v1/accounts` - Create account
- `GET /api/v1/accounts/{account}` - Get account details
- `PUT /api/v1/accounts/{account}` - Update account
- `DELETE /api/v1/accounts/{account}` - Delete account

### Allocation Management
- `GET /api/v1/allocations` - List allocation schedules
- `POST /api/v1/allocations` - Create allocation schedule
- `GET /api/v1/allocations/{id}` - Get allocation schedule
- `PUT /api/v1/allocations/{id}` - Update allocation schedule
- `POST /api/v1/allocations/process` - Process pending allocations

### Grant Management
- `GET /api/v1/grants` - List research grants
- `POST /api/v1/grants` - Create grant account
- `GET /api/v1/grants/{grant_number}` - Get grant details
- `PUT /api/v1/grants/{grant_number}` - Update grant
- `GET /api/v1/grants/{grant_number}/periods` - List budget periods
- `POST /api/v1/grants/{grant_number}/reports` - Generate compliance reports

### Burn Rate Analytics
- `GET /api/v1/burn-rate/{account}` - Get burn rate analysis for account
- `GET /api/v1/burn-rate/grant/{grant_number}` - Get burn rate analysis for grant
- `GET /api/v1/alerts` - List active budget alerts
- `POST /api/v1/alerts/{id}/acknowledge` - Acknowledge alert
- `GET /api/v1/analytics/health-score` - Get budget health scores

### ASBX Integration (aws-slurm-burst)
- `POST /api/v1/asbx/reconcile` - Process ASBX v0.2.0 cost data for reconciliation
- `POST /api/v1/asbx/epilog` - Handle SLURM epilog data from ASBX
- `GET /api/v1/asbx/status` - Get ASBX integration status

### ASBA Integration (Academic Slurm Burst Allocation)
- `POST /api/v1/asba/budget-status` - Budget status for intelligent decision making
- `POST /api/v1/asba/affordability-check` - Job affordability assessment with risk analysis
- `POST /api/v1/asba/grant-timeline` - Grant timeline and deadline optimization
- `POST /api/v1/asba/burst-decision` - Comprehensive burst decision recommendations

### System Endpoints
- `GET /health` - Service health check
- `GET /metrics` - Prometheus metrics
- `GET /version` - Version information

## ⚙️ Configuration

Key configuration options in `/etc/asbb/config.yaml`:

```yaml
# Service configuration
service:
  listen_addr: ":8080"

# Database configuration
database:
  driver: "postgres"
  dsn: "postgresql://asbb:password@localhost/asbb?sslmode=disable"

# Advisor service integration
advisor:
  url: "http://localhost:8081"
  timeout: "30s"

# Budget management settings
budget:
  # Default percentage buffer for holds (1.2 = 20% buffer)
  default_hold_percentage: 1.2

  # How long to wait before auto-reconciling orphaned transactions
  reconciliation_timeout: "24h"

  # Enable automatic recovery of orphaned transactions
  auto_recovery_enabled: true
  recovery_check_interval: "1h"

# Allocation processing
allocations:
  # How often to check for pending allocations
  process_interval: "1h"

  # Enable automatic allocation processing
  auto_process: true
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run integration tests
make test-integration

# Run benchmarks
make bench
```

## 🔒 Security

- Input validation on all API endpoints
- SQL injection prevention with prepared statements
- Audit logging for all budget operations
- Optional JWT authentication for API access
- Rate limiting and CORS support

## 📊 Monitoring

Built-in Prometheus metrics:
- Budget account balances and usage
- Transaction counts and amounts
- Allocation processing statistics
- System health and performance metrics

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run quality checks: `make pre-commit`
5. Submit a pull request

## 📈 Go Report Card

This project maintains a **Grade A** rating on [Go Report Card](https://goreportcard.com/) through:
- Comprehensive test coverage (>80%)
- Consistent code formatting (`gofmt`)
- No golint warnings
- No go vet issues
- Effective code organization
- Minimal cyclomatic complexity

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 📦 Releases

This project uses [GoReleaser](https://goreleaser.com/) for automated releases with:

- **Cross-platform Binaries**: Linux, macOS, Windows (amd64, arm64)
- **Package Managers**: Homebrew, APT, RPM, APK packages
- **Docker Images**: Multi-platform container images
- **Checksums**: SHA256 checksums for all artifacts
- **Release Notes**: Automatically generated from commit history

### Release Process

```bash
# Create a new release
git tag v0.2.0
git push origin v0.2.0

# GoReleaser automatically:
# 1. Builds cross-platform binaries
# 2. Creates GitHub release with notes
# 3. Uploads all artifacts
# 4. Publishes Docker images
# 5. Updates Homebrew formula
```

## 🔗 Related Projects

- [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor) - Intelligent burst decision engine
- [aws-slurm-burst](https://github.com/scttfrdmn/aws-slurm-burst) - AWS workload bursting implementation

## 📚 Complete Documentation

### Core Documentation
- **[API Reference](docs/API_REFERENCE.md)** - Complete REST API documentation with examples
- **[Grant Management Guide](docs/GRANT_MANAGEMENT.md)** - Multi-year grant tracking and burn rate analytics
- **[ASBX Integration Guide](docs/ASBX_INTEGRATION.md)** - Integration with aws-slurm-burst v0.2.0
- **[ASBA Integration Guide](docs/ASBA_INTEGRATION.md)** - Academic Slurm Burst Allocation decision making APIs
- **[Integration Roadmap](INTEGRATION_ROADMAP.md)** - Sister project coordination plan

### Configuration & Deployment
- **[Configuration Example](configs/config.example.yaml)** - Complete configuration reference
- **[Database Migrations](migrations/)** - Schema evolution and setup
- **[Docker Deployment](deployments/docker/)** - Container-based deployment
- **[Kubernetes Manifests](deployments/kubernetes/)** - Production K8s deployment

### Development
- **[CHANGELOG](CHANGELOG.md)** - Detailed release notes and version history
- **[Contributing Guide](CONTRIBUTING.md)** - Development setup and contribution guidelines
- **[Testing Guide](test/)** - Unit and integration testing with Docker

## 📞 Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/scttfrdmn/aws-slurm-burst-budget/issues)
- **Discussions**: [Community discussions](https://github.com/scttfrdmn/aws-slurm-burst-budget/discussions)
- **Wiki**: [Additional documentation](https://github.com/scttfrdmn/aws-slurm-burst-budget/wiki)

---

Copyright © 2025 Scott Friedman. All rights reserved.