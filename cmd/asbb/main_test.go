// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAPIClient_NotImplemented(t *testing.T) {
	client, err := getAPIClient()
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestRootCommand_Exists(t *testing.T) {
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "asbb", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "AWS SLURM Bursting Budget")
}

func TestVersionCommand_Exists(t *testing.T) {
	assert.NotNil(t, versionCmd)
	assert.Equal(t, "version", versionCmd.Use)
	assert.Contains(t, versionCmd.Short, "Show version information")
}

func TestAccountCommand_Exists(t *testing.T) {
	assert.NotNil(t, accountCmd)
	assert.Equal(t, "account", accountCmd.Use)
	assert.Contains(t, accountCmd.Short, "Manage budget accounts")
}

func TestAllocationsCommand_Exists(t *testing.T) {
	assert.NotNil(t, allocationsCmd)
	assert.Equal(t, "allocations", allocationsCmd.Use)
	assert.Contains(t, allocationsCmd.Short, "Manage incremental budget allocations")
}

func TestUsageCommand_Exists(t *testing.T) {
	assert.NotNil(t, usageCmd)
	assert.Equal(t, "usage", usageCmd.Use)
	assert.Contains(t, usageCmd.Short, "View usage reports")
}

func TestTransactionCommand_Exists(t *testing.T) {
	assert.NotNil(t, transactionCmd)
	assert.Equal(t, "transactions", transactionCmd.Use)
	assert.Contains(t, transactionCmd.Short, "Manage transactions")
}

func TestConfigCommand_Exists(t *testing.T) {
	assert.NotNil(t, configCmd)
	assert.Equal(t, "config", configCmd.Use)
	assert.Contains(t, configCmd.Short, "Configuration management")
}

func TestServiceCommand_Exists(t *testing.T) {
	assert.NotNil(t, serviceCmd)
	assert.Equal(t, "service", serviceCmd.Use)
	assert.Contains(t, serviceCmd.Short, "Service management")
}

func TestDatabaseCommand_Exists(t *testing.T) {
	assert.NotNil(t, databaseCmd)
	assert.Equal(t, "database", databaseCmd.Use)
	assert.Contains(t, databaseCmd.Short, "Database management")
}
