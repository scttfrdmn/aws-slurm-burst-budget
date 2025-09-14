# AWS SLURM Bursting Budget (ASBB)

[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/aws-slurm-burst-budget)](https://goreportcard.com/report/github.com/scttfrdmn/aws-slurm-burst-budget)
[![CI/CD Pipeline](https://github.com/scttfrdmn/aws-slurm-burst-budget/actions/workflows/ci.yml/badge.svg)](https://github.com/scttfrdmn/aws-slurm-burst-budget/actions/workflows/ci.yml)
[![Coverage Status](https://codecov.io/gh/scttfrdmn/aws-slurm-burst-budget/branch/main/graph/badge.svg)](https://codecov.io/gh/scttfrdmn/aws-slurm-burst-budget)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive budget management system for HPC clusters that burst workloads to AWS, providing real-money budget enforcement and cost tracking through SLURM integration.

## 🌟 Features

### Core Budget Management
- **Real-time Budget Enforcement**: Check budget availability at job submission time
- **Pre-allocation Model**: Hold estimated costs with configurable buffer, reconcile actual costs upon completion
- **Complete Audit Trail**: Track every budget operation with full transaction history
- **Account-based Budgets**: Map budget accounts to SLURM accounts with flexible limits

### 🆕 Incremental Budget Allocation
- **Scheduled Allocations**: Automatically allocate budget over time (e.g., $600 total allocated at $100/month)
- **Flexible Frequencies**: Daily, weekly, monthly, quarterly, or yearly allocations
- **Automatic Processing**: Background service handles allocations automatically
- **Manual Control**: Override automatic allocations when needed

### Advanced Features
- **Partition-specific Limits**: Different budget limits for CPU vs GPU partitions
- **Cost Estimation Integration**: Works with [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor)
- **Recovery System**: Automatic cleanup of orphaned transactions after outages
- **Comprehensive CLI**: Full command-line interface for budget administration
- **REST API**: Complete HTTP API for integration with external systems
- **Prometheus Metrics**: Built-in monitoring and alerting support

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

## 🔧 CLI Commands

### Account Management
```bash
asbb account list                    # List all budget accounts
asbb account create [options]       # Create new budget account
asbb account show <account>         # Show account details & allocation schedule
asbb account update <account>       # Update account settings
asbb account delete <account>       # Delete account
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

## 🔗 Related Projects

- [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor) - Intelligent burst decision engine
- [aws-slurm-burst](https://github.com/scttfrdmn/aws-slurm-burst) - AWS workload bursting implementation

## 📞 Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/scttfrdmn/aws-slurm-burst-budget/issues)
- **Documentation**: [Wiki pages](https://github.com/scttfrdmn/aws-slurm-burst-budget/wiki)

---

Copyright © 2025 Scott Friedman. All rights reserved.