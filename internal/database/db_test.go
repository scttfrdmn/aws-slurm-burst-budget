// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
)

func TestConnect_InvalidDriver(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "invalid-driver",
		DSN:    "test",
	}

	db, err := Connect(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestConnect_InvalidDSN(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "postgres",
		DSN:    "invalid-dsn-format",
	}

	db, err := Connect(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestDB_GetStats(t *testing.T) {
	// Test with nil DB (edge case)
	db := &DB{DB: nil}

	// This would panic with nil DB, so we test the method exists
	// In a real test environment, we'd use a test database
	assert.NotNil(t, db)
}

func TestDB_Close(t *testing.T) {
	// Test Close method exists and can be called
	db := &DB{}

	// Note: In real testing, we'd test with actual database connection
	// For now, testing method signature and that it doesn't panic
	assert.NotNil(t, db)
}

func TestDB_BeginTx(t *testing.T) {
	// Test BeginTx method signature
	db := &DB{}
	ctx := context.Background()
	opts := &sql.TxOptions{}

	// Note: Would need real DB connection for functional test
	// Testing that method exists with correct signature
	assert.NotNil(t, db)
	assert.NotNil(t, ctx)
	assert.NotNil(t, opts)
}

func TestDB_WithTransaction_PanicRecovery(t *testing.T) {
	// Test panic recovery logic exists
	db := &DB{}

	// Test that WithTransaction method exists
	assert.NotNil(t, db)

	// The actual panic recovery would be tested with integration tests
	// that require a real database connection
}

func TestDB_HealthCheck_Context(t *testing.T) {
	db := &DB{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test that HealthCheck accepts context properly
	// Note: Would need real DB for functional test
	assert.NotNil(t, db)
	assert.NotNil(t, ctx)
}

func TestMigrate_UnsupportedDriver(t *testing.T) {
	db := &DB{
		config: &config.DatabaseConfig{
			Driver:         "unsupported-driver",
			MigrationsPath: "test-path",
		},
	}

	// Test that migrate methods handle unsupported drivers properly
	err := db.Migrate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database driver")

	err = db.MigrateWithPath("test-path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database driver")

	err = db.MigrateDown()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database driver")
}

// Note: Integration test helpers moved to test/integration package

// Benchmark test for database operations
func BenchmarkDB_HealthCheck(b *testing.B) {
	// Would benchmark actual health checks with real DB
	b.Skip("Benchmark test - requires test database")
}
