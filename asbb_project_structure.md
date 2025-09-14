# AWS SLURM Bursting Budget - Development Guide

This document provides a comprehensive guide for implementing the aws-slurm-bursting-budget tool using Claude Code.

## Project Structure

```
aws-slurm-bursting-budget/
├── README.md                     # Main project documentation
├── LICENSE                       # MIT License
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── Makefile                      # Build system
├── .gitignore                    # Git ignore patterns
├── .air.toml                     # Hot reload configuration for development
├── .golangci.yml                 # Linter configuration
├── Dockerfile                    # Container build instructions
├── docker-compose.yml            # Development environment
│
├── cmd/                          # Main applications
│   ├── asbb/                     # CLI application
│   │   ├── main.go              # Main entry point
│   │   ├── account.go           # Account management commands
│   │   ├── usage.go             # Usage reporting commands
│   │   ├── transactions.go      # Transaction management commands
│   │   ├── forecast.go          # Budget forecasting commands
│   │   ├── config.go            # Configuration commands
│   │   ├── service.go           # Service management commands
│   │   ├── database.go          # Database management commands
│   │   ├── reconcile.go         # Manual reconciliation commands
│   │   └── recover.go           # Recovery commands
│   ├── budget-service/          # Budget service daemon
│   │   ├── main.go              # Service main
│   │   ├── server.go            # HTTP server setup
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── middleware.go        # HTTP middleware
│   │   └── routes.go            # Route definitions
│   └── recovery/                # Recovery utilities
│       ├── main.go              # Recovery tool main
│       └── cleanup.go           # Cleanup operations
│
├── internal/                     # Private application code
│   ├── budget/                  # Budget management logic
│   │   ├── service.go           # Main budget service
│   │   ├── account.go           # Account operations
│   │   ├── transaction.go       # Transaction operations
│   │   ├── reconciliation.go    # Reconciliation logic
│   │   ├── forecast.go          # Forecasting algorithms
│   │   ├── validation.go        # Input validation
│   │   └── metrics.go           # Prometheus metrics
│   ├── database/                # Database layer
│   │   ├── db.go                # Database connection and setup
│   │   ├── migrations.go        # Migration management
│   │   ├── account_queries.go   # Account-related queries
│   │   ├── transaction_queries.go # Transaction-related queries
│   │   └── utils.go             # Database utilities
│   ├── advisor/                 # Advisor client
│   │   ├── client.go            # HTTP client for advisor service
│   │   ├── types.go             # Advisor-specific types
│   │   ├── cache.go             # Response caching
│   │   └── retry.go             # Retry logic
│   ├── slurm/                   # SLURM integration
│   │   ├── client.go            # SLURM command wrapper
│   │   ├── job_monitor.go       # Job monitoring
│   │   ├── epilog.go            # Epilog script integration
│   │   └── parser.go            # SLURM output parsing
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Configuration structure
│   │   ├── validation.go        # Config validation
│   │   └── examples.go          # Example configurations
│   ├── auth/                    # Authentication/authorization
│   │   ├── middleware.go        # Auth middleware
│   │   ├── jwt.go               # JWT handling
│   │   └── permissions.go       # Permission checking
│   └── logging/                 # Logging setup
│       ├── logger.go            # Logger configuration
│       └── middleware.go        # Logging middleware
│
├── pkg/                          # Public API packages
│   ├── api/                     # API types and client
│   │   ├── types.go             # Core data types
│   │   ├── client.go            # HTTP client
│   │   ├── errors.go            # Error types
│   │   └── validation.go        # API validation
│   └── version/                 # Version information
│       └── version.go           # Version constants
│
├── plugins/                      # SLURM plugins
│   └── slurm/                   # SLURM job submit plugin
│       ├── job_submit_budget.c  # C plugin implementation
│       ├── Makefile            # Plugin build configuration
│       └── README.md           # Plugin documentation
│
├── migrations/                   # Database migrations
│   ├── 001_initial_schema.up.sql
│   ├── 001_initial_schema.down.sql
│   ├── 002_add_indexes.up.sql
│   ├── 002_add_indexes.down.sql
│   └── README.md               # Migration documentation
│
├── deployments/                  # Deployment configurations
│   ├── systemd/                # Systemd service files
│   │   └── asbb-service.service
│   ├── docker/                 # Docker configurations
│   │   ├── Dockerfile.service
│   │   └── docker-compose.yml
│   ├── kubernetes/             # Kubernetes manifests
│   │   ├── namespace.yaml
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── configmap.yaml
│   └── ansible/                # Ansible playbooks
│       ├── site.yml
│       └── roles/
│
├── scripts/                      # Utility scripts
│   ├── install.sh              # Installation script
│   ├── uninstall.sh            # Uninstallation script
│   ├── backup.sh               # Database backup script
│   ├── restore.sh              # Database restore script
│   └── dev-setup.sh            # Development environment setup
│
├── configs/                      # Example configurations
│   ├── config.yaml             # Main configuration example
│   ├── local-costs.yaml        # Local cost model example
│   └── slurm/                  # SLURM integration configs
│       └── plugstack.conf.example
│
├── docs/                         # Documentation
│   ├── api/                    # API documentation
│   │   ├── openapi.yaml        # OpenAPI specification
│   │   └── README.md           # API guide
│   ├── installation/           # Installation guides
│   │   ├── quickstart.md
│   │   ├── production.md
│   │   └── troubleshooting.md
│   ├── configuration/          # Configuration guides
│   │   ├── service.md
│   │   ├── database.md
│   │   └── slurm.md
│   └── development/            # Development documentation
│       ├── architecture.md
│       ├── testing.md
│       └── contributing.md
│
├── examples/                     # Example files
│   ├── job_scripts/            # Example SLURM job scripts
│   │   ├── cpu_job.sbatch
│   │   ├── gpu_job.sbatch
│   │   └── mpi_job.sbatch
│   ├── configs/                # Example configurations
│   │   ├── development.yaml
│   │   ├── production.yaml
│   │   └── test.yaml
│   └── scripts/                # Example scripts
│       ├── budget_report.sh
│       └── cost_analysis.py
│
├── test/                         # Test files
│   ├── unit/                   # Unit tests
│   ├── integration/            # Integration tests
│   ├── e2e/                    # End-to-end tests
│   ├── fixtures/               # Test fixtures
│   └── testdata/               # Test data
│
└── build/                        # Build artifacts (gitignored)
    ├── asbb                    # CLI binary
    ├── budget-service          # Service binary
    ├── budget-recovery         # Recovery binary
    └── job_submit_budget.so    # SLURM plugin
```

## Implementation Priority

When implementing with Claude Code, follow this priority order:

### Phase 1: Core Foundation
1. **Project Setup**
   - Initialize Go module (`go.mod`)
   - Set up basic project structure
   - Create `.gitignore` and basic configuration files

2. **Core Types and Configuration**
   - Implement `pkg/api/types.go` - All core data structures
   - Implement `internal/config/config.go` - Configuration management
   - Create example configuration files

3. **Database Layer**
   - Create database migration files (`migrations/001_initial_schema.up.sql`)
   - Implement `internal/database/db.go` - Database connection
   - Implement basic query functions

### Phase 2: Core Services
4. **Budget Service Core**
   - Implement `internal/budget/service.go` - Main budget logic
   - Implement `internal/budget/account.go` - Account operations
   - Implement `internal/budget/transaction.go` - Transaction handling

5. **Advisor Integration**
   - Implement `internal/advisor/client.go` - HTTP client for advisor
   - Add retry logic and caching
   - Create mock client for testing

6. **CLI Foundation**
   - Implement `cmd/asbb/main.go` - Main CLI entry point
   - Implement `cmd/asbb/account.go` - Account management commands
   - Add basic command structure

### Phase 3: API and Service
7. **HTTP Service**
   - Implement `cmd/budget-service/main.go` - Service daemon
   - Implement HTTP handlers and routes
   - Add middleware for logging and authentication

8. **Advanced CLI Features**
   - Implement usage reporting commands
   - Implement transaction history commands
   - Add forecasting functionality

### Phase 4: SLURM Integration
9. **SLURM Plugin**
   - Compile and test the C plugin
   - Create installation and configuration scripts
   - Test integration with SLURM

10. **Job Monitoring**
    - Implement job completion monitoring
    - Add automatic reconciliation
    - Create recovery utilities

### Phase 5: Production Features
11. **Testing and Quality**
    - Add comprehensive unit tests
    - Add integration tests
    - Set up CI/CD pipeline

12. **Documentation and Deployment**
    - Complete API documentation
    - Create installation guides
    - Add deployment configurations

## Key Implementation Files

### Essential Files to Start With:

1. **go.mod** - Already provided
2. **pkg/api/types.go** - Already provided (all core types)
3. **internal/config/config.go** - Already provided (configuration structure)
4. **migrations/001_initial_schema.up.sql** - Already provided (database schema)
5. **cmd/asbb/main.go** - Already provided (CLI foundation)

### Critical Business Logic Files:

1. **internal/budget/service.go** - Partially provided (needs completion)
2. **internal/database/db.go** - Needs implementation
3. **internal/advisor/client.go** - Needs implementation
4. **cmd/budget-service/main.go** - Needs implementation

## Development Environment Setup

1. **Prerequisites**
   ```bash
   # Install Go 1.21+
   # Install PostgreSQL
   # Install SLURM development headers
   # Install required C libraries (libcurl, json-c)
   ```

2. **Initial Setup**
   ```bash
   git clone <repository>
   cd aws-slurm-bursting-budget
   make dev-setup
   ```

3. **Database Setup**
   ```bash
   # Create database
   createdb asbb
   
   # Run migrations
   make migrate-up
   ```

4. **Build and Test**
   ```bash
   make build
   make test
   ```

## Integration Points

### With aws-slurm-burst-advisor
- HTTP client calls to advisor service for cost estimation
- Budget context passed to advisor for enhanced recommendations
- Shared configuration for consistent cost models

### With aws-slurm-burst
- Budget information flows through job metadata
- Reconciliation triggers from job completion events
- Shared understanding of AWS instance costs

### With SLURM
- Job submit plugin for budget enforcement
- Epilog scripts for job completion handling
- Account association mapping
- Job monitoring through SLURM commands

## Testing Strategy

1. **Unit Tests**: Test individual components in isolation
2. **Integration Tests**: Test database operations and HTTP APIs
3. **End-to-End Tests**: Test complete workflows with mock SLURM
4. **Performance Tests**: Test with high job submission rates
5. **Recovery Tests**: Test system recovery after failures

## Security Considerations

1. **Database Security**: Use connection pooling and prepared statements
2. **API Security**: Implement JWT authentication for production
3. **SLURM Integration**: Validate user permissions through SLURM
4. **Audit Trail**: Complete logging of all budget operations
5. **Input Validation**: Sanitize all user inputs

## Monitoring and Observability

1. **Metrics**: Prometheus metrics for all operations
2. **Logging**: Structured logging with correlation IDs
3. **Health Checks**: Database and advisor service health
4. **Alerting**: Budget exhaustion and system failure alerts

This structure provides a solid foundation for implementing the aws-slurm-bursting-budget tool with Claude Code, with clear priorities and well-defined interfaces between components.
