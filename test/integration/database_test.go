// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/database"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

func TestDatabase_AccountOperations(t *testing.T) {
	SkipIfNoDocker(t)

	db := SetupTestDatabase(t)
	defer TeardownTestDatabase(t, db)

	accountQueries := database.NewAccountQueries(db)
	ctx := context.Background()

	t.Run("CreateAccount", func(t *testing.T) {
		req := &api.CreateAccountRequest{
			SlurmAccount: "test-account-1",
			Name:         "Test Account 1",
			Description:  "Test account for integration testing",
			BudgetLimit:  1000.0,
			StartDate:    time.Now().Add(-24 * time.Hour),
			EndDate:      time.Now().Add(365 * 24 * time.Hour),
		}

		account, err := accountQueries.CreateAccount(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, req.SlurmAccount, account.SlurmAccount)
		assert.Equal(t, req.Name, account.Name)
		assert.Equal(t, req.Description, account.Description)
		assert.Equal(t, req.BudgetLimit, account.BudgetLimit)
		assert.Equal(t, 0.0, account.BudgetUsed)
		assert.Equal(t, 0.0, account.BudgetHeld)
		assert.Equal(t, "active", account.Status)
		assert.True(t, account.IsActive())
	})

	t.Run("GetAccountByName", func(t *testing.T) {
		account, err := accountQueries.GetAccountByName(ctx, "test-account-1")
		require.NoError(t, err)

		assert.Equal(t, "test-account-1", account.SlurmAccount)
		assert.Equal(t, "Test Account 1", account.Name)
		assert.Equal(t, 1000.0, account.BudgetLimit)
	})

	t.Run("GetAccountByName_NotFound", func(t *testing.T) {
		account, err := accountQueries.GetAccountByName(ctx, "nonexistent-account")
		assert.Error(t, err)
		assert.Nil(t, account)

		budgetErr, ok := api.AsBudgetError(err)
		require.True(t, ok)
		assert.Equal(t, api.ErrCodeNotFound, budgetErr.Code)
	})

	t.Run("ListAccounts", func(t *testing.T) {
		req := &api.ListAccountsRequest{
			Limit:  10,
			Offset: 0,
		}

		accounts, err := accountQueries.ListAccounts(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, len(accounts), 0)

		// Find our test account
		found := false
		for _, account := range accounts {
			if account.SlurmAccount == "test-account-1" {
				found = true
				break
			}
		}
		assert.True(t, found, "Test account should be found in list")
	})

	t.Run("UpdateAccount", func(t *testing.T) {
		newName := "Updated Test Account"
		newDescription := "Updated description"
		newBudgetLimit := 2000.0

		req := &api.UpdateAccountRequest{
			Name:        &newName,
			Description: &newDescription,
			BudgetLimit: &newBudgetLimit,
		}

		account, err := accountQueries.UpdateAccount(ctx, "test-account-1", req)
		require.NoError(t, err)

		assert.Equal(t, newName, account.Name)
		assert.Equal(t, newDescription, account.Description)
		assert.Equal(t, newBudgetLimit, account.BudgetLimit)
	})

	t.Run("DeleteAccount", func(t *testing.T) {
		err := accountQueries.DeleteAccount(ctx, "test-account-1")
		require.NoError(t, err)

		// Verify account is deleted
		account, err := accountQueries.GetAccountByName(ctx, "test-account-1")
		assert.Error(t, err)
		assert.Nil(t, account)
	})
}

func TestDatabase_TransactionOperations(t *testing.T) {
	SkipIfNoDocker(t)

	db := SetupTestDatabase(t)
	defer TeardownTestDatabase(t, db)

	accountQueries := database.NewAccountQueries(db)
	transactionQueries := database.NewTransactionQueries(db)
	ctx := context.Background()

	// Create test account first
	accountReq := &api.CreateAccountRequest{
		SlurmAccount: "test-account-txn",
		Name:         "Test Account for Transactions",
		BudgetLimit:  1000.0,
		StartDate:    time.Now().Add(-24 * time.Hour),
		EndDate:      time.Now().Add(365 * 24 * time.Hour),
	}

	account, err := accountQueries.CreateAccount(ctx, accountReq)
	require.NoError(t, err)

	t.Run("CreateTransaction", func(t *testing.T) {
		transaction := &api.BudgetTransaction{
			TransactionID: "test-txn-001",
			AccountID:     account.ID,
			Type:          "hold",
			Amount:        100.0,
			Description:   "Test hold transaction",
			Status:        "pending",
		}

		err := transactionQueries.CreateTransaction(ctx, nil, transaction)
		require.NoError(t, err)
		assert.Greater(t, transaction.ID, int64(0))
	})

	t.Run("GetTransaction", func(t *testing.T) {
		transaction, err := transactionQueries.GetTransaction(ctx, "test-txn-001")
		require.NoError(t, err)

		assert.Equal(t, "test-txn-001", transaction.TransactionID)
		assert.Equal(t, account.ID, transaction.AccountID)
		assert.Equal(t, "hold", transaction.Type)
		assert.Equal(t, 100.0, transaction.Amount)
		assert.Equal(t, "pending", transaction.Status)
	})

	t.Run("UpdateTransactionStatus", func(t *testing.T) {
		err := transactionQueries.UpdateTransactionStatus(ctx, nil, "test-txn-001", "completed")
		require.NoError(t, err)

		// Verify status was updated
		transaction, err := transactionQueries.GetTransaction(ctx, "test-txn-001")
		require.NoError(t, err)
		assert.Equal(t, "completed", transaction.Status)
		assert.NotNil(t, transaction.CompletedAt)
	})

	t.Run("ListTransactions", func(t *testing.T) {
		req := &api.TransactionListRequest{
			Account: "test-account-txn",
			Limit:   10,
		}

		transactions, err := transactionQueries.ListTransactions(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, len(transactions), 0)

		// Find our test transaction
		found := false
		for _, txn := range transactions {
			if txn.TransactionID == "test-txn-001" {
				found = true
				assert.Equal(t, "completed", txn.Status)
				break
			}
		}
		assert.True(t, found, "Test transaction should be found in list")
	})
}

func TestDatabase_MigrationOperations(t *testing.T) {
	SkipIfNoDocker(t)

	db := SetupTestDatabase(t)
	defer TeardownTestDatabase(t, db)

	ctx := context.Background()

	t.Run("HealthCheck", func(t *testing.T) {
		err := db.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := db.GetStats()
		assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)
	})

	t.Run("WithTransaction", func(t *testing.T) {
		executed := false
		err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("WithTransaction_Rollback", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
			return testErr
		})

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})
}