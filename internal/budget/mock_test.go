// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package budget

import (
	"context"
	"database/sql"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// MockAccountQueries provides a mock implementation for testing
type MockAccountQueries struct {
	GetAccountByNameFunc func(ctx context.Context, account string) (*api.BudgetAccount, error)
	CreateAccountFunc    func(ctx context.Context, req *api.CreateAccountRequest) (*api.BudgetAccount, error)
	ListAccountsFunc     func(ctx context.Context, req *api.ListAccountsRequest) ([]*api.BudgetAccount, error)
	UpdateAccountFunc    func(ctx context.Context, account string, req *api.UpdateAccountRequest) (*api.BudgetAccount, error)
	DeleteAccountFunc    func(ctx context.Context, account string) error
}

func (m *MockAccountQueries) GetAccountByName(ctx context.Context, account string) (*api.BudgetAccount, error) {
	if m.GetAccountByNameFunc != nil {
		return m.GetAccountByNameFunc(ctx, account)
	}
	return nil, api.NewAccountNotFoundError(account)
}

func (m *MockAccountQueries) CreateAccount(ctx context.Context, req *api.CreateAccountRequest) (*api.BudgetAccount, error) {
	if m.CreateAccountFunc != nil {
		return m.CreateAccountFunc(ctx, req)
	}
	return nil, api.NewBudgetError(api.ErrCodeNotFound, "mock not implemented")
}

func (m *MockAccountQueries) ListAccounts(ctx context.Context, req *api.ListAccountsRequest) ([]*api.BudgetAccount, error) {
	if m.ListAccountsFunc != nil {
		return m.ListAccountsFunc(ctx, req)
	}
	return []*api.BudgetAccount{}, nil
}

func (m *MockAccountQueries) UpdateAccount(ctx context.Context, account string, req *api.UpdateAccountRequest) (*api.BudgetAccount, error) {
	if m.UpdateAccountFunc != nil {
		return m.UpdateAccountFunc(ctx, account, req)
	}
	return nil, api.NewAccountNotFoundError(account)
}

func (m *MockAccountQueries) DeleteAccount(ctx context.Context, account string) error {
	if m.DeleteAccountFunc != nil {
		return m.DeleteAccountFunc(ctx, account)
	}
	return api.NewAccountNotFoundError(account)
}

// MockTransactionQueries provides a mock implementation for testing
type MockTransactionQueries struct {
	CreateTransactionFunc       func(ctx context.Context, tx *sql.Tx, transaction *api.BudgetTransaction) error
	GetTransactionFunc          func(ctx context.Context, transactionID string) (*api.BudgetTransaction, error)
	UpdateTransactionStatusFunc func(ctx context.Context, tx *sql.Tx, transactionID string, status string) error
	ListTransactionsFunc        func(ctx context.Context, req *api.TransactionListRequest) ([]*api.BudgetTransaction, error)
}

func (m *MockTransactionQueries) CreateTransaction(ctx context.Context, tx *sql.Tx, transaction *api.BudgetTransaction) error {
	if m.CreateTransactionFunc != nil {
		return m.CreateTransactionFunc(ctx, tx, transaction)
	}
	return nil
}

func (m *MockTransactionQueries) GetTransaction(ctx context.Context, transactionID string) (*api.BudgetTransaction, error) {
	if m.GetTransactionFunc != nil {
		return m.GetTransactionFunc(ctx, transactionID)
	}
	return nil, api.NewBudgetError(api.ErrCodeNotFound, "transaction not found")
}

func (m *MockTransactionQueries) UpdateTransactionStatus(ctx context.Context, tx *sql.Tx, transactionID string, status string) error {
	if m.UpdateTransactionStatusFunc != nil {
		return m.UpdateTransactionStatusFunc(ctx, tx, transactionID, status)
	}
	return nil
}

func (m *MockTransactionQueries) ListTransactions(ctx context.Context, req *api.TransactionListRequest) ([]*api.BudgetTransaction, error) {
	if m.ListTransactionsFunc != nil {
		return m.ListTransactionsFunc(ctx, req)
	}
	return []*api.BudgetTransaction{}, nil
}

// MockDB provides a mock database for testing
type MockDB struct {
	WithTransactionFunc func(ctx context.Context, fn func(*sql.Tx) error) error
	HealthCheckFunc     func(ctx context.Context) error
}

func (m *MockDB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	if m.WithTransactionFunc != nil {
		return m.WithTransactionFunc(ctx, fn)
	}
	// Simulate successful transaction
	return fn(nil)
}

func (m *MockDB) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}