// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// DB wraps the database connection with additional functionality
type DB struct {
	*sql.DB
	config *config.DatabaseConfig
}

// Connect establishes a connection to the database
func Connect(cfg *config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, config: cfg}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	return db.MigrateWithPath(db.config.MigrationsPath)
}

// MigrateWithPath runs database migrations from a specific path
func (db *DB) MigrateWithPath(migrationsPath string) error {
	var driver migrate.DatabaseDriver
	var err error

	switch db.config.Driver {
	case "postgres":
		driver, err = postgres.WithInstance(db.DB, &postgres.Config{})
	case "mysql":
		driver, err = mysql.WithInstance(db.DB, &mysql.Config{})
	default:
		return fmt.Errorf("unsupported database driver: %s", db.config.Driver)
	}

	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	sourceDriver, err := (&file.File{}).Open(fmt.Sprintf("file://%s", migrationsPath))
	if err != nil {
		return fmt.Errorf("failed to open migrations directory: %w", err)
	}

	m, err := migrate.NewWithInstance("file", sourceDriver, db.config.Driver, driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// MigrateDown rolls back the latest migration
func (db *DB) MigrateDown() error {
	var driver migrate.DatabaseDriver
	var err error

	switch db.config.Driver {
	case "postgres":
		driver, err = postgres.WithInstance(db.DB, &postgres.Config{})
	case "mysql":
		driver, err = mysql.WithInstance(db.DB, &mysql.Config{})
	default:
		return fmt.Errorf("unsupported database driver: %s", db.config.Driver)
	}

	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	sourceDriver, err := (&file.File{}).Open(fmt.Sprintf("file://%s", db.config.MigrationsPath))
	if err != nil {
		return fmt.Errorf("failed to open migrations directory: %w", err)
	}

	m, err := migrate.NewWithInstance("file", sourceDriver, db.config.Driver, driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	return nil
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// GetStats returns database connection statistics
func (db *DB) GetStats() sql.DBStats {
	return db.DB.Stats()
}