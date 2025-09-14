# Claude Code Implementation Setup

This guide provides step-by-step instructions for implementing the aws-slurm-bursting-budget tool using Claude Code.

## Initial Project Setup

### 1. Create the Project Structure

Start by creating the basic directory structure:

```bash
mkdir aws-slurm-bursting-budget
cd aws-slurm-bursting-budget

# Create main directories
mkdir -p cmd/asbb cmd/budget-service cmd/recovery
mkdir -p internal/{budget,database,advisor,slurm,config,auth,logging}
mkdir -p pkg/{api,version}
mkdir -p plugins/slurm
mkdir -p migrations
mkdir -p deployments/{systemd,docker,kubernetes}
mkdir -p scripts configs docs examples test
```

### 2. Initialize Go Module

Create the `go.mod` file with the provided content and initialize:

```bash
# Copy go.mod content from artifact
go mod tidy
```

### 3. Create Essential Files

Copy the following files from the artifacts in this order:

1. **README.md** - Main project documentation
2. **Makefile** - Build system configuration
3. **pkg/api/types.go** - Core data types
4. **internal/config/config.go** - Configuration management
5. **migrations/001_initial_schema.up.sql** - Database schema
6. **cmd/asbb/main.go** - CLI application foundation
7. **internal/budget/service.go** - Budget service foundation
8. **plugins/slurm/job_submit_budget.c** - SLURM plugin

## Implementation Phase 1: Core Foundation

### Step 1: Database Layer Implementation

Create `internal/database/db.go`:

```go
package database

import (
    "database/sql"
    "fmt"
    "time"
    
    _ "github.com/lib/pq"
    _ "github.com/go-sql-driver/mysql"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/file"
    
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/config"
)

type DB struct {
    *sql.DB
    config *config.DatabaseConfig
}

func Connect(cfg *config.DatabaseConfig) (*DB, error) {
    db, err := sql.Open(cfg.Driver, cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
    
    // Test connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    return &DB{DB: db, config: cfg}, nil
}

func (db *DB) Migrate() error {
    // Migration logic here
    return nil
}
```

### Step 2: Advisor Client Implementation

Create `internal/advisor/client.go`:

```go
package advisor

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/config"
)

type Client struct {
    httpClient *http.Client
    baseURL    string
    apiKey     string
}

type CostEstimateRequest struct {
    Account   string `json:"account"`
    Partition string `json:"partition"`
    Nodes     int    `json:"nodes"`
    CPUs      int    `json:"cpus"`
    GPUs      int    `json:"gpus,omitempty"`
    Memory    string `json:"memory,omitempty"`
    WallTime  string `json:"wall_time"`
    JobScript string `json:"job_script,omitempty"`
}

type CostEstimateResponse struct {
    EstimatedCost   float64                `json:"estimated_cost"`
    Recommendation  string                 `json:"recommendation"`
    LocalCost       float64                `json:"local_cost,omitempty"`
    AWSCost         float64                `json:"aws_cost,omitempty"`
    CostBreakdown   map[string]interface{} `json:"cost_breakdown,omitempty"`
    Confidence      float64                `json:"confidence"`
    Raw             map[string]interface{} `json:"-"`
}

func NewClient(cfg *config.AdvisorConfig) *Client {
    return &Client{
        httpClient: &http.Client{Timeout: cfg.Timeout},
        baseURL:    cfg.URL,
        apiKey:     cfg.APIKey,
    }
}

func (c *Client) EstimateCost(ctx context.Context, req *CostEstimateRequest) (*CostEstimateResponse, error) {
    // HTTP request to advisor service
    // Implementation details here
    return nil, nil
}
```

### Step 3: Complete CLI Commands

Create individual command files in `cmd/asbb/`:

**cmd/asbb/account.go**:
```go
package main

import (
    "fmt"
    "time"
    
    "github.com/spf13/cobra"
    "github.com/scttfrdmn/aws-slurm-bursting-budget/pkg/api"
)

func init() {
    accountCmd.AddCommand(accountListCmd)
    accountCmd.AddCommand(accountCreateCmd)
    accountCmd.AddCommand(accountShowCmd)
    accountCmd.AddCommand(accountUpdateCmd)
    accountCmd.AddCommand(accountDeleteCmd)
}

var accountListCmd = &cobra.Command{
    Use:   "list",
    Short: "List budget accounts",
    RunE: func(cmd *cobra.Command, args []string) error {
        client, err := getAPIClient()
        if err != nil {
            return err
        }
        
        accounts, err := client.ListAccounts(cmd.Context(), &api.ListAccountsRequest{})
        if err != nil {
            return err
        }
        
        // Display accounts in table format
        fmt.Printf("%-15s %-20s %-10s %-10s %-10s %-10s\n", 
            "ACCOUNT", "NAME", "LIMIT", "USED", "HELD", "AVAILABLE")
        for _, account := range accounts {
            fmt.Printf("%-15s %-20s $%-9.2f $%-9.2f $%-9.2f $%-9.2f\n",
                account.SlurmAccount, account.Name, account.BudgetLimit,
                account.BudgetUsed, account.BudgetHeld, 
                account.BudgetLimit-account.BudgetUsed-account.BudgetHeld)
        }
        
        return nil
    },
}

var accountCreateCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new budget account",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation for creating accounts
        return nil
    },
}

// Additional commands...
```

## Implementation Phase 2: Service Development

### Step 4: HTTP Service Implementation

Create `cmd/budget-service/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/rs/zerolog/log"
    
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/budget"
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/config"
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/database"
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/advisor"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to load configuration")
    }
    
    // Connect to database
    db, err := database.Connect(&cfg.Database)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to connect to database")
    }
    defer db.Close()
    
    // Initialize advisor client
    advisorClient := advisor.NewClient(&cfg.Advisor)
    
    // Initialize budget service
    budgetService := budget.NewService(db, advisorClient, &cfg.Budget)
    
    // Setup HTTP server
    router := mux.NewRouter()
    setupRoutes(router, budgetService)
    
    server := &http.Server{
        Addr:         cfg.Service.ListenAddr,
        Handler:      router,
        ReadTimeout:  cfg.Service.ReadTimeout,
        WriteTimeout: cfg.Service.WriteTimeout,
    }
    
    // Start server
    go func() {
        log.Info().Str("addr", cfg.Service.ListenAddr).Msg("Starting budget service")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal().Err(err).Msg("Server failed")
        }
    }()
    
    // Wait for interrupt signal
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), cfg.Service.ShutdownTimeout)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Error().Err(err).Msg("Server shutdown failed")
    }
    
    log.Info().Msg("Server stopped")
}

func setupRoutes(router *mux.Router, service *budget.Service) {
    api := router.PathPrefix("/api/v1").Subrouter()
    
    // Budget check endpoint (used by SLURM plugin)
    api.HandleFunc("/budget/check", handleBudgetCheck(service)).Methods("POST")
    
    // Job reconciliation endpoint
    api.HandleFunc("/budget/reconcile", handleJobReconcile(service)).Methods("POST")
    
    // Account management endpoints
    api.HandleFunc("/accounts", handleListAccounts(service)).Methods("GET")
    api.HandleFunc("/accounts", handleCreateAccount(service)).Methods("POST")
    api.HandleFunc("/accounts/{account}", handleGetAccount(service)).Methods("GET")
    
    // Health check
    api.HandleFunc("/health", handleHealth()).Methods("GET")
}
```

### Step 5: Database Query Implementation

Complete the database layer with specific query functions:

**internal/database/account_queries.go**:
```go
package database

import (
    "context"
    "database/sql"
    
    "github.com/scttfrdmn/aws-slurm-bursting-budget/pkg/api"
)

func (db *DB) GetAccountByName(ctx context.Context, account string) (*api.BudgetAccount, error) {
    query := `
        SELECT id, slurm_account, name, description, budget_limit, 
               budget_used, budget_held, start_date, end_date, status,
               created_at, updated_at
        FROM budget_accounts 
        WHERE slurm_account = $1`
    
    var acc api.BudgetAccount
    err := db.QueryRowContext(ctx, query, account).Scan(
        &acc.ID, &acc.SlurmAccount, &acc.Name, &acc.Description,
        &acc.BudgetLimit, &acc.BudgetUsed, &acc.BudgetHeld,
        &acc.StartDate, &acc.EndDate, &acc.Status,
        &acc.CreatedAt, &acc.UpdatedAt,
    )
    
    if err != nil {
        return nil, err
    }
    
    return &acc, nil
}

func (db *DB) CreateAccount(ctx context.Context, req *api.CreateAccountRequest) (*api.BudgetAccount, error) {
    query := `
        INSERT INTO budget_accounts (slurm_account, name, description, budget_limit, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, slurm_account, name, description, budget_limit, budget_used, budget_held,
                  start_date, end_date, status, created_at, updated_at`
    
    var acc api.BudgetAccount
    err := db.QueryRowContext(ctx, query,
        req.SlurmAccount, req.Name, req.Description,
        req.BudgetLimit, req.StartDate, req.EndDate,
    ).Scan(
        &acc.ID, &acc.SlurmAccount, &acc.Name, &acc.Description,
        &acc.BudgetLimit, &acc.BudgetUsed, &acc.BudgetHeld,
        &acc.StartDate, &acc.EndDate, &acc.Status,
        &acc.CreatedAt, &acc.UpdatedAt,
    )
    
    return &acc, err
}

// Additional query functions...
```

## Testing Strategy

### Step 6: Create Test Structure

```bash
mkdir -p test/{unit,integration,e2e}
```

**test/unit/budget_test.go**:
```go
package budget_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/scttfrdmn/aws-slurm-bursting-budget/internal/budget"
    "github.com/scttfrdmn/aws-slurm-bursting-budget/pkg/api"
)

func TestBudgetCheck(t *testing.T) {
    // Mock dependencies
    mockDB := &MockDB{}
    mockAdvisor := &MockAdvisor{}
    
    service := budget.NewService(mockDB, mockAdvisor, &config.BudgetConfig{
        DefaultHoldPercentage: 1.2,
    })
    
    // Test cases
    t.Run("sufficient budget", func(t *testing.T) {
        req := &api.BudgetCheckRequest{
            Account:   "test-account",
            Partition: "cpu",
            Nodes:     1,
            CPUs:      4,
            WallTime:  "01:00:00",
        }
        
        // Set up mocks
        mockAdvisor.On("EstimateCost", mock.Anything, mock.Anything).Return(&advisor.CostEstimateResponse{
            EstimatedCost: 10.0,
        }, nil)
        
        mockDB.On("GetAccountByName", mock.Anything, "test-account").Return(&api.BudgetAccount{
            ID:          1,
            BudgetLimit: 1000.0,
            BudgetUsed:  100.0,
            BudgetHeld:  50.0,
        }, nil)
        
        resp, err := service.CheckBudget(context.Background(), req)
        
        assert.NoError(t, err)
        assert.True(t, resp.Available)
        assert.Equal(t, 12.0, resp.HoldAmount) // 10.0 * 1.2
    })
}
```

## Build and Deployment

### Step 7: Build System Usage

```bash
# Check environment
make check-env

# Build all components
make build

# Run tests
make test

# Install locally for testing
sudo make install

# Configure SLURM plugin
sudo make install-plugin
```

### Step 8: Configuration

Create configuration files:

```bash
# Generate default config
./build/asbb config init --output /etc/asbb/config.yaml

# Edit configuration
sudo vi /etc/asbb/config.yaml
```

Example configuration:

```yaml
service:
  listen_addr: ":8080"

database:
  driver: "postgres"
  dsn: "postgresql://asbb:password@localhost/asbb?sslmode=disable"

advisor:
  url: "http://localhost:8081"
  timeout: "30s"

budget:
  default_hold_percentage: 1.2
  reconciliation_timeout: "24h"

slurm:
  bin_path: "/usr/bin"
```

## Integration Testing

### Step 9: End-to-End Testing

1. **Start Services**:
   ```bash
   # Start database
   sudo systemctl start postgresql
   
   # Run migrations
   ./build/asbb database migrate
   
   # Start budget service
   ./build/budget-service
   ```

2. **Test CLI Operations**:
   ```bash
   # Create test account
   ./build/asbb account create --name=test-proj --account=proj001 --budget=1000 --start=2025-01-01 --end=2025-12-31
   
   # Check account
   ./build/asbb account show proj001
   
   # Test budget check
   curl -X POST http://localhost:8080/api/v1/budget/check \
     -H "Content-Type: application/json" \
     -d '{"account":"proj001","partition":"cpu","nodes":1,"cpus":4,"wall_time":"01:00:00"}'
   ```

3. **Test SLURM Integration**:
   ```bash
   # Submit test job
   sbatch --account=proj001 test_job.sbatch
   ```

## Monitoring and Maintenance

### Step 10: Production Deployment

1. **Systemd Service**:
   ```bash
   sudo systemctl enable asbb-service
   sudo systemctl start asbb-service
   ```

2. **Monitoring**:
   ```bash
   # Check service status
   sudo systemctl status asbb-service
   
   # View logs
   journalctl -u asbb-service -f
   
   # Check budget status
   ./build/asbb usage summary
   ```

This setup provides a complete foundation for implementing the aws-slurm-bursting-budget tool with Claude Code, with clear implementation phases and testing strategies.
