// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Service     ServiceConfig     `mapstructure:"service" yaml:"service"`
	Database    DatabaseConfig    `mapstructure:"database" yaml:"database"`
	Advisor     AdvisorConfig     `mapstructure:"advisor" yaml:"advisor"`
	Budget      BudgetConfig      `mapstructure:"budget" yaml:"budget"`
	SLURM       SLURMConfig       `mapstructure:"slurm" yaml:"slurm"`
	Logging     LoggingConfig     `mapstructure:"logging" yaml:"logging"`
	Auth        AuthConfig        `mapstructure:"auth" yaml:"auth"`
	Metrics     MetricsConfig     `mapstructure:"metrics" yaml:"metrics"`
	Integration IntegrationConfig `mapstructure:"integration" yaml:"integration"`
}

// IntegrationConfig contains optional integration settings
type IntegrationConfig struct {
	// ASBX (aws-slurm-burst) integration - OPTIONAL
	ASBXEnabled  bool          `mapstructure:"asbx_enabled" yaml:"asbx_enabled"`
	ASBXEndpoint string        `mapstructure:"asbx_endpoint" yaml:"asbx_endpoint"`
	ASBXTimeout  time.Duration `mapstructure:"asbx_timeout" yaml:"asbx_timeout"`
	ASBXAPIKey   string        `mapstructure:"asbx_api_key" yaml:"asbx_api_key"`

	// ASBA (Academic Slurm Burst Allocation) integration - OPTIONAL
	ASBAEnabled  bool          `mapstructure:"asba_enabled" yaml:"asba_enabled"`
	ASBAEndpoint string        `mapstructure:"asba_endpoint" yaml:"asba_endpoint"`
	ASBATimeout  time.Duration `mapstructure:"asba_timeout" yaml:"asba_timeout"`

	// Advisor service integration - OPTIONAL
	AdvisorEnabled   bool    `mapstructure:"advisor_enabled" yaml:"advisor_enabled"`
	AdvisorFallback  string  `mapstructure:"advisor_fallback" yaml:"advisor_fallback"`     // STATIC, SIMPLE, NONE
	FallbackCostRate float64 `mapstructure:"fallback_cost_rate" yaml:"fallback_cost_rate"` // $/hour when advisor unavailable

	// Feature toggles for optional functionality
	GrantManagementEnabled      bool `mapstructure:"grant_management_enabled" yaml:"grant_management_enabled"`
	BurnRateAnalysisEnabled     bool `mapstructure:"burn_rate_analysis_enabled" yaml:"burn_rate_analysis_enabled"`
	AllocationSchedulingEnabled bool `mapstructure:"allocation_scheduling_enabled" yaml:"allocation_scheduling_enabled"`

	// Graceful degradation settings
	FailureMode           string        `mapstructure:"failure_mode" yaml:"failure_mode"` // STRICT, GRACEFUL, PERMISSIVE
	RetryAttempts         int           `mapstructure:"retry_attempts" yaml:"retry_attempts"`
	CircuitBreakerEnabled bool          `mapstructure:"circuit_breaker_enabled" yaml:"circuit_breaker_enabled"`
	HealthCheckInterval   time.Duration `mapstructure:"health_check_interval" yaml:"health_check_interval"`
}

// ServiceConfig contains HTTP service configuration
type ServiceConfig struct {
	ListenAddr      string        `mapstructure:"listen_addr" yaml:"listen_addr"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout"`
	TLSEnabled      bool          `mapstructure:"tls_enabled" yaml:"tls_enabled"`
	TLSCertFile     string        `mapstructure:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile      string        `mapstructure:"tls_key_file" yaml:"tls_key_file"`
	CORSEnabled     bool          `mapstructure:"cors_enabled" yaml:"cors_enabled"`
	CORSOrigins     []string      `mapstructure:"cors_origins" yaml:"cors_origins"`
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver" yaml:"driver"`
	DSN             string        `mapstructure:"dsn" yaml:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	MigrationsPath  string        `mapstructure:"migrations_path" yaml:"migrations_path"`
	AutoMigrate     bool          `mapstructure:"auto_migrate" yaml:"auto_migrate"`
}

// AdvisorConfig contains advisor service configuration - OPTIONAL
type AdvisorConfig struct {
	URL           string            `mapstructure:"url" yaml:"url"`
	APIKey        string            `mapstructure:"api_key" yaml:"api_key"`
	Timeout       time.Duration     `mapstructure:"timeout" yaml:"timeout"`
	RetryAttempts int               `mapstructure:"retry_attempts" yaml:"retry_attempts"`
	RetryDelay    time.Duration     `mapstructure:"retry_delay" yaml:"retry_delay"`
	CacheEnabled  bool              `mapstructure:"cache_enabled" yaml:"cache_enabled"`
	CacheTTL      time.Duration     `mapstructure:"cache_ttl" yaml:"cache_ttl"`
	Headers       map[string]string `mapstructure:"headers" yaml:"headers"`
}

// BudgetConfig contains budget management configuration
type BudgetConfig struct {
	DefaultHoldPercentage float64       `mapstructure:"default_hold_percentage" yaml:"default_hold_percentage"`
	ReconciliationTimeout time.Duration `mapstructure:"reconciliation_timeout" yaml:"reconciliation_timeout"`
	MinBudgetAmount       float64       `mapstructure:"min_budget_amount" yaml:"min_budget_amount"`
	MaxBudgetAmount       float64       `mapstructure:"max_budget_amount" yaml:"max_budget_amount"`
	AllowNegativeBalance  bool          `mapstructure:"allow_negative_balance" yaml:"allow_negative_balance"`
	AutoRecoveryEnabled   bool          `mapstructure:"auto_recovery_enabled" yaml:"auto_recovery_enabled"`
	RecoveryCheckInterval time.Duration `mapstructure:"recovery_check_interval" yaml:"recovery_check_interval"`
	TransactionRetention  time.Duration `mapstructure:"transaction_retention" yaml:"transaction_retention"`
}

// SLURMConfig contains SLURM integration configuration
type SLURMConfig struct {
	BinPath           string        `mapstructure:"bin_path" yaml:"bin_path"`
	ConfigPath        string        `mapstructure:"config_path" yaml:"config_path"`
	JobMonitorEnabled bool          `mapstructure:"job_monitor_enabled" yaml:"job_monitor_enabled"`
	MonitorInterval   time.Duration `mapstructure:"monitor_interval" yaml:"monitor_interval"`
	EpilogScript      string        `mapstructure:"epilog_script" yaml:"epilog_script"`
	DefaultPartition  string        `mapstructure:"default_partition" yaml:"default_partition"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level" yaml:"level"`
	Format     string `mapstructure:"format" yaml:"format"` // json or console
	Output     string `mapstructure:"output" yaml:"output"` // stdout, stderr, or file path
	Structured bool   `mapstructure:"structured" yaml:"structured"`
	Sampling   struct {
		Initial    uint32        `mapstructure:"initial" yaml:"initial"`
		Thereafter uint32        `mapstructure:"thereafter" yaml:"thereafter"`
		Tick       time.Duration `mapstructure:"tick" yaml:"tick"`
	} `mapstructure:"sampling" yaml:"sampling"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled"`
	JWTSecret  string        `mapstructure:"jwt_secret" yaml:"jwt_secret"`
	JWTExpiry  time.Duration `mapstructure:"jwt_expiry" yaml:"jwt_expiry"`
	APIKeyAuth bool          `mapstructure:"api_key_auth" yaml:"api_key_auth"`
	APIKeys    []string      `mapstructure:"api_keys" yaml:"api_keys"`
	AdminUsers []string      `mapstructure:"admin_users" yaml:"admin_users"`
}

// MetricsConfig contains metrics/monitoring configuration
type MetricsConfig struct {
	Enabled         bool          `mapstructure:"enabled" yaml:"enabled"`
	Path            string        `mapstructure:"path" yaml:"path"`
	Namespace       string        `mapstructure:"namespace" yaml:"namespace"`
	Subsystem       string        `mapstructure:"subsystem" yaml:"subsystem"`
	CollectInterval time.Duration `mapstructure:"collect_interval" yaml:"collect_interval"`
	PrometheusURL   string        `mapstructure:"prometheus_url" yaml:"prometheus_url"`
}

// Load loads configuration from multiple sources
func Load() (*Config, error) {
	return LoadWithPath("")
}

// LoadWithPath loads configuration from a specific path
func LoadWithPath(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file search paths
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/asbb")
		v.AddConfigPath("$HOME/.asbb")
	}

	// Environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("ASBB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Service defaults
	v.SetDefault("service.listen_addr", ":8080")
	v.SetDefault("service.read_timeout", "30s")
	v.SetDefault("service.write_timeout", "30s")
	v.SetDefault("service.shutdown_timeout", "30s")
	v.SetDefault("service.tls_enabled", false)
	v.SetDefault("service.cors_enabled", false)
	v.SetDefault("service.cors_origins", []string{"*"})

	// Database defaults (REQUIRED - core functionality)
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.migrations_path", "migrations")
	v.SetDefault("database.auto_migrate", false)

	// Advisor defaults (OPTIONAL - graceful degradation)
	v.SetDefault("advisor.url", "http://localhost:8081")
	v.SetDefault("advisor.timeout", "30s")
	v.SetDefault("advisor.retry_attempts", 3)
	v.SetDefault("advisor.retry_delay", "1s")
	v.SetDefault("advisor.cache_enabled", true)
	v.SetDefault("advisor.cache_ttl", "5m")

	// Integration defaults (ALL OPTIONAL)
	v.SetDefault("integration.asbx_enabled", false)
	v.SetDefault("integration.asbx_endpoint", "http://localhost:8082")
	v.SetDefault("integration.asbx_timeout", "30s")

	v.SetDefault("integration.asba_enabled", false)
	v.SetDefault("integration.asba_endpoint", "http://localhost:8083")
	v.SetDefault("integration.asba_timeout", "30s")

	v.SetDefault("integration.advisor_enabled", true)      // Default enabled but graceful fallback
	v.SetDefault("integration.advisor_fallback", "SIMPLE") // STATIC, SIMPLE, NONE
	v.SetDefault("integration.fallback_cost_rate", 0.10)   // $0.10/hour default

	v.SetDefault("integration.grant_management_enabled", true)
	v.SetDefault("integration.burn_rate_analysis_enabled", true)
	v.SetDefault("integration.allocation_scheduling_enabled", true)

	v.SetDefault("integration.failure_mode", "GRACEFUL") // STRICT, GRACEFUL, PERMISSIVE
	v.SetDefault("integration.retry_attempts", 3)
	v.SetDefault("integration.circuit_breaker_enabled", true)
	v.SetDefault("integration.health_check_interval", "60s")

	// Budget defaults
	v.SetDefault("budget.default_hold_percentage", 1.2)
	v.SetDefault("budget.reconciliation_timeout", "24h")
	v.SetDefault("budget.min_budget_amount", 0.01)
	v.SetDefault("budget.max_budget_amount", 1000000.0)
	v.SetDefault("budget.allow_negative_balance", false)
	v.SetDefault("budget.auto_recovery_enabled", true)
	v.SetDefault("budget.recovery_check_interval", "1h")
	v.SetDefault("budget.transaction_retention", "2160h") // 90 days

	// SLURM defaults
	v.SetDefault("slurm.bin_path", "/usr/bin")
	v.SetDefault("slurm.config_path", "/etc/slurm")
	v.SetDefault("slurm.job_monitor_enabled", true)
	v.SetDefault("slurm.monitor_interval", "30s")
	v.SetDefault("slurm.default_partition", "cpu")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.structured", true)
	v.SetDefault("logging.sampling.initial", 100)
	v.SetDefault("logging.sampling.thereafter", 100)
	v.SetDefault("logging.sampling.tick", "1s")

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.jwt_expiry", "24h")
	v.SetDefault("auth.api_key_auth", false)

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.namespace", "asbb")
	v.SetDefault("metrics.subsystem", "budget")
	v.SetDefault("metrics.collect_interval", "15s")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.Service.Validate(); err != nil {
		return fmt.Errorf("service config: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	// Advisor validation only if enabled
	if c.Integration.AdvisorEnabled {
		if err := c.Advisor.Validate(); err != nil {
			return fmt.Errorf("advisor config: %w", err)
		}
	}
	if err := c.Budget.Validate(); err != nil {
		return fmt.Errorf("budget config: %w", err)
	}
	return nil
}

// Validate validates ServiceConfig
func (sc *ServiceConfig) Validate() error {
	if sc.ListenAddr == "" {
		return fmt.Errorf("listen_addr is required")
	}
	if sc.TLSEnabled && (sc.TLSCertFile == "" || sc.TLSKeyFile == "") {
		return fmt.Errorf("TLS cert and key files are required when TLS is enabled")
	}
	return nil
}

// Validate validates DatabaseConfig
func (dc *DatabaseConfig) Validate() error {
	if dc.Driver == "" {
		return fmt.Errorf("database driver is required")
	}
	if dc.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}
	if dc.Driver != "postgres" && dc.Driver != "mysql" {
		return fmt.Errorf("unsupported database driver: %s", dc.Driver)
	}
	return nil
}

// Validate validates AdvisorConfig (only called if advisor enabled)
func (ac *AdvisorConfig) Validate() error {
	if ac.URL == "" {
		return fmt.Errorf("advisor URL is required when advisor is enabled")
	}
	if ac.Timeout <= 0 {
		return fmt.Errorf("advisor timeout must be positive")
	}
	return nil
}

// Validate validates BudgetConfig
func (bc *BudgetConfig) Validate() error {
	if bc.DefaultHoldPercentage <= 0 {
		return fmt.Errorf("default_hold_percentage must be positive")
	}
	if bc.MinBudgetAmount < 0 {
		return fmt.Errorf("min_budget_amount cannot be negative")
	}
	if bc.MaxBudgetAmount <= bc.MinBudgetAmount {
		return fmt.Errorf("max_budget_amount must be greater than min_budget_amount")
	}
	return nil
}

// IsStandalone returns true if running in standalone mode (no integrations)
func (c *Config) IsStandalone() bool {
	return !c.Integration.AdvisorEnabled &&
		!c.Integration.ASBXEnabled &&
		!c.Integration.ASBAEnabled
}

// HasIntegrations returns true if any external integrations are enabled
func (c *Config) HasIntegrations() bool {
	return c.Integration.AdvisorEnabled ||
		c.Integration.ASBXEnabled ||
		c.Integration.ASBAEnabled
}

// CanFallback returns true if graceful fallback is enabled
func (c *Config) CanFallback() bool {
	return c.Integration.FailureMode == "GRACEFUL" || c.Integration.FailureMode == "PERMISSIVE"
}

// IsStrictMode returns true if strict mode is enabled (fail on integration errors)
func (c *Config) IsStrictMode() bool {
	return c.Integration.FailureMode == "STRICT"
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return os.Getenv("ASBB_ENV") == "development" ||
		os.Getenv("GO_ENV") == "development" ||
		c.Logging.Level == "debug"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return os.Getenv("ASBB_ENV") == "production" ||
		os.Getenv("GO_ENV") == "production"
}

// GetDSNWithoutPassword returns DSN with password masked for logging
func (dc *DatabaseConfig) GetDSNWithoutPassword() string {
	dsn := dc.DSN
	// Simple password masking - in production you might want more sophisticated logic
	if strings.Contains(dsn, "password=") {
		parts := strings.Split(dsn, " ")
		for i, part := range parts {
			if strings.HasPrefix(part, "password=") {
				parts[i] = "password=***"
			}
		}
		return strings.Join(parts, " ")
	}
	if strings.Contains(dsn, ":") && strings.Contains(dsn, "@") {
		// Handle mysql-style DSNs: user:password@host
		at := strings.LastIndex(dsn, "@")
		if at > 0 {
			userPass := dsn[:at]
			host := dsn[at:]
			if colon := strings.Index(userPass, ":"); colon > 0 {
				return userPass[:colon+1] + "***" + host
			}
		}
	}
	return dsn
}
