# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **GoReleaser Integration**: Automated release process with cross-platform binaries
- **Package Management**: Homebrew, APT, RPM, APK package support
- **Multi-platform Docker**: Automated Docker image builds for amd64 and arm64
- **Recovery Tool**: Standalone recovery utility for orphaned transactions

### Improved
- **Go Report Card Grade A**: Fixed all code quality issues (gofmt, go vet, golint, ineffassign, misspell)
- **Documentation**: Enhanced installation options and release process documentation
- **CI/CD Pipeline**: Updated GitHub Actions workflow for GoReleaser integration

## [0.1.0] - 2025-09-13

### Added

#### Core Budget Management
- **Budget Accounts**: Create and manage budget accounts mapped to SLURM accounts
- **Real-time Budget Checking**: SLURM job submit plugin enforces budget limits at submission time
- **Pre-allocation Model**: Hold estimated costs with configurable buffer, reconcile actual costs on completion
- **Cost Estimation Integration**: Seamless integration with aws-slurm-burst-advisor for intelligent cost prediction
- **Complete Audit Trail**: Full transaction logging for financial accountability

#### 🆕 Incremental Budget Allocation
- **Scheduled Allocations**: Automatically allocate budget over time (e.g., $600 total allocated at $100/month)
- **Flexible Frequencies**: Support for daily, weekly, monthly, quarterly, and yearly allocations
- **Automatic Processing**: Background service handles allocations automatically based on schedule
- **Manual Control**: Override automatic allocations when needed
- **Allocation History**: Track all allocation events with complete audit trail

#### API and Integration
- **REST API**: Comprehensive HTTP API for all budget operations
- **SLURM Plugin**: C-based job submit plugin for budget enforcement
- **CLI Application**: Full-featured command-line interface for budget administration
- **Database Support**: PostgreSQL and MySQL support with migrations

#### Monitoring and Operations
- **Health Checks**: Service health monitoring endpoints
- **Prometheus Metrics**: Built-in metrics collection for monitoring
- **Recovery System**: Automatic cleanup of orphaned transactions after outages
- **Partition Limits**: Per-partition budget constraints

#### Development and Quality
- **Go Report Card Grade A**: Maintains highest code quality standards
- **Comprehensive Testing**: >80% test coverage with unit, integration, and benchmark tests
- **CI/CD Pipeline**: GitHub Actions workflow with quality gates
- **Git Hooks**: Pre-commit hooks ensure code quality
- **Docker Support**: Complete containerization with multi-stage builds
- **Documentation**: Comprehensive README and API documentation

### Technical Details

#### Database Schema
- `budget_accounts` - Budget account definitions and current usage
- `budget_partition_limits` - Partition-specific budget constraints
- `budget_transactions` - Complete audit trail of all operations
- `budget_allocation_schedules` - Incremental allocation configurations
- `budget_allocations` - History of allocation events
- `job_submissions` - Job tracking for reconciliation

#### API Endpoints
- `POST /api/v1/budget/check` - Budget availability checking
- `POST /api/v1/budget/reconcile` - Job cost reconciliation
- `GET|POST|PUT|DELETE /api/v1/accounts/*` - Account management
- `GET|POST|PUT /api/v1/allocations/*` - Allocation schedule management
- `GET /api/v1/transactions` - Transaction history
- `GET /health` - Service health status
- `GET /metrics` - Prometheus metrics

#### CLI Commands
- `asbb account` - Account management operations
- `asbb allocations` - Allocation schedule management
- `asbb usage` - Usage reporting and analysis
- `asbb transactions` - Transaction history and reconciliation
- `asbb config` - Configuration management
- `asbb database` - Database operations and migrations

#### Configuration
- YAML-based configuration with environment variable support
- Flexible database connection options
- Advisor service integration settings
- Budget policy configuration (hold percentages, timeouts)
- Allocation processing settings

### Security
- Input validation on all API endpoints
- SQL injection prevention with prepared statements
- Audit logging for all budget operations
- Optional authentication and authorization
- Rate limiting and CORS support

### Performance
- Database connection pooling
- Efficient indexing strategy
- Background processing for allocations
- Prometheus metrics for monitoring
- Health check endpoints

### Examples

#### Creating Account with Monthly Allocation
```bash
asbb account create \
  --name="Research Project" \
  --account=research-001 \
  --incremental \
  --total-budget=1200.00 \
  --allocation-amount=100.00 \
  --allocation-frequency=monthly \
  --start=2025-01-01 \
  --end=2025-12-31
```

#### SLURM Integration
```bash
# Add to /etc/slurm/slurm.conf
JobSubmitPlugins=budget

# Add to /etc/slurm/plugstack.conf
required job_submit_budget.so budget_service_url=http://localhost:8080
```

### Dependencies
- Go 1.21+
- PostgreSQL 13+ or MySQL 8.0+
- SLURM 22.05+ with development headers
- aws-slurm-burst-advisor service

---

Copyright © 2025 Scott Friedman. All rights reserved.