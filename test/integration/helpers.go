// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/database"
)

const (
	testDSN = "postgresql://asbb_test:test_password@localhost:5433/asbb_test?sslmode=disable"
)

// SetupTestDatabase starts a Docker test database and returns a connected DB instance
func SetupTestDatabase(t *testing.T) *database.DB {
	// Check if docker is available
	_, err := exec.LookPath("docker-compose")
	if err != nil {
		t.Skip("docker-compose not available, skipping integration tests")
	}

	// Start test database
	cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "up", "-d", "--wait")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to start test database: %v", err)
	}

	// Wait for database to be ready
	cfg := &config.DatabaseConfig{
		Driver:          "postgres",
		DSN:             testDSN,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		MigrationsPath:  "../../migrations",
	}

	var db *database.DB
	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		db, err = database.Connect(cfg)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = db.HealthCheck(ctx)
			cancel()
			if err == nil {
				break
			}
			db.Close()
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		t.Fatalf("Failed to connect to test database after 30 seconds: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// TeardownTestDatabase stops the test database and cleans up
func TeardownTestDatabase(t *testing.T, db *database.DB) {
	if db != nil {
		db.Close()
	}

	// Stop and remove test containers
	cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "down", "-v")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to clean up test database: %v", err)
	}
}

// GetTestDSN returns the test database DSN
func GetTestDSN() string {
	return testDSN
}

// IsDockerAvailable checks if Docker is available for testing
func IsDockerAvailable() bool {
	_, err := exec.LookPath("docker-compose")
	return err == nil
}

// SkipIfNoDocker skips the test if Docker is not available
func SkipIfNoDocker(t *testing.T) {
	if !IsDockerAvailable() {
		t.Skip("Docker not available, skipping integration test")
	}

	// Also check if SKIP_INTEGRATION env var is set
	if os.Getenv("SKIP_INTEGRATION") == "true" {
		t.Skip("Integration tests disabled via SKIP_INTEGRATION=true")
	}
}

// SetupTestConfig creates a test configuration for integration tests
func SetupTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Driver:          "postgres",
			DSN:             testDSN,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			MigrationsPath:  "../../migrations",
		},
		Budget: config.BudgetConfig{
			DefaultHoldPercentage:   1.2,
			ReconciliationTimeout:   24 * time.Hour,
			MinBudgetAmount:         0.01,
			MaxBudgetAmount:         1000000.0,
			AllowNegativeBalance:    false,
			AutoRecoveryEnabled:     true,
			RecoveryCheckInterval:   1 * time.Hour,
			TransactionRetention:    2160 * time.Hour,
		},
		Advisor: config.AdvisorConfig{
			URL:           "http://localhost:8081",
			Timeout:       30 * time.Second,
			RetryAttempts: 3,
			RetryDelay:    1 * time.Second,
		},
	}
}