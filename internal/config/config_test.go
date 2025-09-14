// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithPath(t *testing.T) {
	// Test loading with a config file
	cfg, err := LoadWithPath("../../test_config.yaml")
	require.NoError(t, err)
	assert.Equal(t, ":9999", cfg.Service.ListenAddr)
	assert.Equal(t, "postgres", cfg.Database.Driver)
	assert.Equal(t, "postgresql://user:pass@localhost/testdb", cfg.Database.DSN)
	assert.Equal(t, "http://localhost:8081", cfg.Advisor.URL)
}

func TestLoad_WithDefaults(t *testing.T) {
	// Test that Load() with defaults still validates (should fail due to missing DSN)
	// This is expected behavior - DSN must be provided
	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database DSN is required")
}

func TestServiceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ServiceConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ServiceConfig{
				ListenAddr: ":8080",
			},
			wantErr: false,
		},
		{
			name: "empty listen addr",
			config: ServiceConfig{
				ListenAddr: "",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled without cert",
			config: ServiceConfig{
				ListenAddr: ":8080",
				TLSEnabled: true,
				TLSCertFile: "",
				TLSKeyFile: "",
			},
			wantErr: true,
		},
		{
			name: "TLS enabled with cert and key",
			config: ServiceConfig{
				ListenAddr: ":8080",
				TLSEnabled: true,
				TLSCertFile: "/path/to/cert.pem",
				TLSKeyFile: "/path/to/key.pem",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name: "valid postgres config",
			config: DatabaseConfig{
				Driver: "postgres",
				DSN:    "postgresql://user:pass@localhost/db",
			},
			wantErr: false,
		},
		{
			name: "valid mysql config",
			config: DatabaseConfig{
				Driver: "mysql",
				DSN:    "user:pass@tcp(localhost:3306)/db",
			},
			wantErr: false,
		},
		{
			name: "empty driver",
			config: DatabaseConfig{
				Driver: "",
				DSN:    "some-dsn",
			},
			wantErr: true,
		},
		{
			name: "empty DSN",
			config: DatabaseConfig{
				Driver: "postgres",
				DSN:    "",
			},
			wantErr: true,
		},
		{
			name: "unsupported driver",
			config: DatabaseConfig{
				Driver: "sqlite",
				DSN:    "test.db",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAdvisorConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AdvisorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AdvisorConfig{
				URL:     "http://localhost:8081",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty URL",
			config: AdvisorConfig{
				URL:     "",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: AdvisorConfig{
				URL:     "http://localhost:8081",
				Timeout: 0,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: AdvisorConfig{
				URL:     "http://localhost:8081",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBudgetConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  BudgetConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: BudgetConfig{
				DefaultHoldPercentage: 1.2,
				MinBudgetAmount:       0.01,
				MaxBudgetAmount:       1000000.0,
			},
			wantErr: false,
		},
		{
			name: "zero hold percentage",
			config: BudgetConfig{
				DefaultHoldPercentage: 0,
				MinBudgetAmount:       0.01,
				MaxBudgetAmount:       1000000.0,
			},
			wantErr: true,
		},
		{
			name: "negative min budget",
			config: BudgetConfig{
				DefaultHoldPercentage: 1.2,
				MinBudgetAmount:       -1.0,
				MaxBudgetAmount:       1000000.0,
			},
			wantErr: true,
		},
		{
			name: "max budget less than min",
			config: BudgetConfig{
				DefaultHoldPercentage: 1.2,
				MinBudgetAmount:       1000.0,
				MaxBudgetAmount:       500.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		config   Config
		expected bool
	}{
		{
			name: "ASBB_ENV development",
			envVars: map[string]string{
				"ASBB_ENV": "development",
			},
			config:   Config{},
			expected: true,
		},
		{
			name: "GO_ENV development",
			envVars: map[string]string{
				"GO_ENV": "development",
			},
			config:   Config{},
			expected: true,
		},
		{
			name: "debug log level",
			config: Config{
				Logging: LoggingConfig{
					Level: "debug",
				},
			},
			expected: true,
		},
		{
			name:     "production environment",
			config:   Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			assert.Equal(t, tt.expected, tt.config.IsDevelopment())
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name: "ASBB_ENV production",
			envVars: map[string]string{
				"ASBB_ENV": "production",
			},
			expected: true,
		},
		{
			name: "GO_ENV production",
			envVars: map[string]string{
				"GO_ENV": "production",
			},
			expected: true,
		},
		{
			name:     "not production",
			envVars:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			config := &Config{}
			assert.Equal(t, tt.expected, config.IsProduction())
		})
	}
}

func TestDatabaseConfig_GetDSNWithoutPassword(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		expected string
	}{
		{
			name:     "postgres DSN with password",
			dsn:      "postgresql://user:secret@localhost/db?sslmode=disable",
			expected: "postgresql:***@localhost/db?sslmode=disable",
		},
		{
			name:     "mysql DSN with password",
			dsn:      "user:password@tcp(localhost:3306)/db",
			expected: "user:***@tcp(localhost:3306)/db",
		},
		{
			name:     "postgres DSN with password parameter",
			dsn:      "host=localhost user=test password=secret dbname=test sslmode=disable",
			expected: "host=localhost user=test password=*** dbname=test sslmode=disable",
		},
		{
			name:     "DSN without password",
			dsn:      "host=localhost user=test dbname=test",
			expected: "host=localhost user=test dbname=test",
		},
		{
			name:     "DSN without colon",
			dsn:      "simple-dsn-format",
			expected: "simple-dsn-format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DatabaseConfig{DSN: tt.dsn}
			assert.Equal(t, tt.expected, config.GetDSNWithoutPassword())
		})
	}
}