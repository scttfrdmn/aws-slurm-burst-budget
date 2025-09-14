# AWS SLURM Bursting Budget (asbb)

A SLURM plugin system for managing real-money budgets across HPC workload bursting to AWS, integrating with the aws-slurm-burst-advisor ecosystem.

## Overview

The AWS SLURM Bursting Budget tool provides budget management and enforcement for HPC clusters that burst workloads to AWS. It implements a hybrid pre-allocation model where estimated costs are held at job submission and reconciled upon completion, similar to SLURM's existing SU (Service Unit) exhaustion behavior.

## Architecture

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
                    │ └─────────────┘ │
                    └─────────────────┘
```

## Features

- **Budget Management**: Create and manage budget accounts mapped to SLURM accounts
- **Cost Estimation**: Integration with aws-slurm-burst-advisor for intelligent cost prediction
- **Pre-allocation**: Hold estimated funds at job submission with configurable buffer
- **Reconciliation**: Charge actual costs and refund differences upon job completion
- **Audit Trail**: Complete transaction logging for financial accountability
- **CLI Tools**: Command-line interface for budget administration and reporting
- **Recovery**: Automatic cleanup of orphaned transactions after outages

## Components

### Core Services
- **Budget Service**: REST API for budget operations and cost checks
- **SLURM Plugin**: Job submit plugin for budget enforcement
- **CLI Tools**: Administrative interface for budget management
- **Recovery Manager**: Handles cleanup after system outages

### Database Schema
- **budget_accounts**: Budget allocations and current usage
- **budget_partition_limits**: Partition-specific budget constraints  
- **budget_transactions**: Complete audit trail of all budget operations

## Installation

### Prerequisites
- Go 1.21+
- SLURM 22.05+ with development headers
- PostgreSQL/MySQL database
- Access to aws-slurm-burst-advisor service

### Build and Install
```bash
# Clone repository
git clone https://github.com/scttfrdmn/aws-slurm-bursting-budget.git
cd aws-slurm-bursting-budget

# Build all components
make build

# Install system-wide
sudo make install

# Install SLURM plugin
sudo make install-plugin
```

### Configuration
```bash
# Initialize configuration
asbb config init

# Setup database
asbb database migrate

# Configure SLURM integration
asbb slurm configure
```

## Quick Start

### 1. Create Budget Account
```bash
# Create a budget account mapped to SLURM account
asbb account create --name=proj001 --budget=50000 --start=2025-01-01 --end=2025-12-31

# Set partition-specific limits
asbb account set-limit proj001 --partition=gpu-aws --limit=10000
```

### 2. Configure SLURM
```bash
# Add to /etc/slurm/slurm.conf
JobSubmitPlugins=budget

# Add to /etc/slurm/plugstack.conf  
required job_submit_budget.so budget_service_url=http://localhost:8080
```

### 3. Submit Jobs
```bash
# Jobs will automatically check budget before submission
sbatch --account=proj001 --partition=gpu-aws my_job.sbatch
```

### 4. Monitor Usage
```bash
# View account status
asbb usage show proj001

# View transaction history
asbb transactions proj001 --last=30d

# Forecast burn rate
asbb forecast proj001
```

## CLI Reference

### Account Management
- `asbb account list` - List all budget accounts
- `asbb account create` - Create new budget account
- `asbb account show <account>` - Show account details
- `asbb account update <account>` - Update account settings
- `asbb account delete <account>` - Delete account

### Usage Monitoring
- `asbb usage show <account>` - Show current usage
- `asbb usage summary` - System-wide usage summary
- `asbb forecast <account>` - Burn rate analysis

### Transaction Management
- `asbb transactions <account>` - View transaction history
- `asbb reconcile <job_id>` - Manual job reconciliation
- `asbb recover` - Cleanup orphaned transactions

### System Administration
- `asbb database migrate` - Run database migrations
- `asbb service start` - Start budget service
- `asbb config validate` - Validate configuration

## Configuration

### Service Configuration (`/etc/asbb/config.yaml`)
```yaml
service:
  listen_addr: ":8080"
  log_level: "info"

database:
  driver: "postgres"
  dsn: "postgresql://user:pass@localhost/asbb"

advisor:
  url: "http://localhost:8081"
  timeout: "30s"

budget:
  default_hold_percentage: 1.2  # 20% buffer
  reconciliation_timeout: "24h"
```

### SLURM Plugin Configuration
```bash
# /etc/slurm/plugstack.conf
required job_submit_budget.so budget_service_url=http://localhost:8080
```

## Integration with Existing Tools

### aws-slurm-burst-advisor Integration
The budget service calls the advisor for cost estimation:
```bash
# Advisor provides cost estimates for budget holds
curl -X POST http://advisor:8081/analyze \
  -d '{"account": "proj001", "partition": "gpu-aws", "nodes": 2}'
```

### aws-slurm-burst Integration
Budget information flows to the burst plugin:
```bash
# Budget context included in burst decisions
asbb job-context --job-id=12345 | aws-slurm-burst resume
```

## Development

### Project Structure
```
aws-slurm-bursting-budget/
├── cmd/
│   ├── asbb/              # Main CLI application
│   ├── budget-service/    # Budget service daemon
│   └── recovery/          # Recovery tools
├── internal/
│   ├── budget/           # Budget management logic
│   ├── database/         # Database operations
│   ├── slurm/           # SLURM integration
│   └── advisor/         # Advisor client
├── pkg/
│   └── api/             # API types and client
├── plugins/
│   └── slurm/           # SLURM job submit plugin
├── deployments/         # Deployment configurations
├── docs/               # Documentation
└── examples/           # Example configurations
```

### Building from Source
```bash
# Development build
make dev

# Run tests
make test

# Integration tests (requires test database)
make test-integration

# Lint and format
make lint
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run quality checks: `make check`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Related Projects

- [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor) - Intelligent burst decision engine
- [aws-slurm-burst](https://github.com/scttfrdmn/aws-slurm-burst) - AWS workload bursting implementation

## Support

- GitHub Issues: [Report bugs and request features](https://github.com/scttfrdmn/aws-slurm-bursting-budget/issues)
- Documentation: [Wiki pages](https://github.com/scttfrdmn/aws-slurm-bursting-budget/wiki)
