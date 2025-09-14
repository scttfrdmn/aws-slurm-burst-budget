// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// AccountQueries provides database operations for budget accounts
type AccountQueries struct {
	db *DB
}

// NewAccountQueries creates a new AccountQueries instance
func NewAccountQueries(db *DB) *AccountQueries {
	return &AccountQueries{db: db}
}

// GetAccountByID retrieves a budget account by ID
func (q *AccountQueries) GetAccountByID(ctx context.Context, id int64) (*api.BudgetAccount, error) {
	query := `
		SELECT id, slurm_account, name, description, budget_limit,
		       budget_used, budget_held, start_date, end_date, status,
		       created_at, updated_at
		FROM budget_accounts
		WHERE id = $1`

	var account api.BudgetAccount
	err := q.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.SlurmAccount, &account.Name, &account.Description,
		&account.BudgetLimit, &account.BudgetUsed, &account.BudgetHeld,
		&account.StartDate, &account.EndDate, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, api.NewAccountNotFoundError(fmt.Sprintf("ID:%d", id))
		}
		return nil, api.NewDatabaseError("get account by ID", err)
	}

	return &account, nil
}

// GetAccountByName retrieves a budget account by SLURM account name
func (q *AccountQueries) GetAccountByName(ctx context.Context, slurmAccount string) (*api.BudgetAccount, error) {
	query := `
		SELECT id, slurm_account, name, description, budget_limit,
		       budget_used, budget_held, start_date, end_date, status,
		       created_at, updated_at
		FROM budget_accounts
		WHERE slurm_account = $1`

	var account api.BudgetAccount
	err := q.db.QueryRowContext(ctx, query, slurmAccount).Scan(
		&account.ID, &account.SlurmAccount, &account.Name, &account.Description,
		&account.BudgetLimit, &account.BudgetUsed, &account.BudgetHeld,
		&account.StartDate, &account.EndDate, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, api.NewAccountNotFoundError(slurmAccount)
		}
		return nil, api.NewDatabaseError("get account by name", err)
	}

	return &account, nil
}

// ListAccounts retrieves a list of budget accounts with optional filtering
func (q *AccountQueries) ListAccounts(ctx context.Context, req *api.ListAccountsRequest) ([]*api.BudgetAccount, error) {
	baseQuery := `
		SELECT id, slurm_account, name, description, budget_limit,
		       budget_used, budget_held, start_date, end_date, status,
		       created_at, updated_at
		FROM budget_accounts`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add status filter if specified
	if req.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, req.Status)
		argIndex++
	}

	// Build WHERE clause
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY created_at DESC"

	if req.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, req.Limit)
		argIndex++
	}

	if req.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, req.Offset)
	}

	rows, err := q.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, api.NewDatabaseError("list accounts", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Database row close failed - log for debugging
			_ = err // Acknowledge error is handled
		}
	}()

	var accounts []*api.BudgetAccount
	for rows.Next() {
		var account api.BudgetAccount
		err := rows.Scan(
			&account.ID, &account.SlurmAccount, &account.Name, &account.Description,
			&account.BudgetLimit, &account.BudgetUsed, &account.BudgetHeld,
			&account.StartDate, &account.EndDate, &account.Status,
			&account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, api.NewDatabaseError("scan account row", err)
		}
		accounts = append(accounts, &account)
	}

	if err = rows.Err(); err != nil {
		return nil, api.NewDatabaseError("iterate account rows", err)
	}

	return accounts, nil
}

// CreateAccount creates a new budget account
func (q *AccountQueries) CreateAccount(ctx context.Context, req *api.CreateAccountRequest) (*api.BudgetAccount, error) {
	query := `
		INSERT INTO budget_accounts (slurm_account, name, description, budget_limit, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, slurm_account, name, description, budget_limit, budget_used, budget_held,
		          start_date, end_date, status, created_at, updated_at`

	var account api.BudgetAccount
	err := q.db.QueryRowContext(ctx, query,
		req.SlurmAccount, req.Name, req.Description,
		req.BudgetLimit, req.StartDate, req.EndDate,
	).Scan(
		&account.ID, &account.SlurmAccount, &account.Name, &account.Description,
		&account.BudgetLimit, &account.BudgetUsed, &account.BudgetHeld,
		&account.StartDate, &account.EndDate, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, api.NewBudgetError(api.ErrCodeDuplicateAccount,
				fmt.Sprintf("Account '%s' already exists", req.SlurmAccount))
		}
		return nil, api.NewDatabaseError("create account", err)
	}

	return &account, nil
}

// UpdateAccount updates an existing budget account
func (q *AccountQueries) UpdateAccount(ctx context.Context, slurmAccount string, req *api.UpdateAccountRequest) (*api.BudgetAccount, error) {
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}

	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *req.Description)
		argIndex++
	}

	if req.BudgetLimit != nil {
		setParts = append(setParts, fmt.Sprintf("budget_limit = $%d", argIndex))
		args = append(args, *req.BudgetLimit)
		argIndex++
	}

	if req.StartDate != nil {
		setParts = append(setParts, fmt.Sprintf("start_date = $%d", argIndex))
		args = append(args, *req.StartDate)
		argIndex++
	}

	if req.EndDate != nil {
		setParts = append(setParts, fmt.Sprintf("end_date = $%d", argIndex))
		args = append(args, *req.EndDate)
		argIndex++
	}

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	if len(setParts) == 0 {
		return q.GetAccountByName(ctx, slurmAccount)
	}

	// Always update updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, "NOW()")
	argIndex++

	query := fmt.Sprintf(`
		UPDATE budget_accounts
		SET %s
		WHERE slurm_account = $%d
		RETURNING id, slurm_account, name, description, budget_limit, budget_used, budget_held,
		          start_date, end_date, status, created_at, updated_at`,
		strings.Join(setParts, ", "), argIndex)

	args = append(args, slurmAccount)

	var account api.BudgetAccount
	err := q.db.QueryRowContext(ctx, query, args...).Scan(
		&account.ID, &account.SlurmAccount, &account.Name, &account.Description,
		&account.BudgetLimit, &account.BudgetUsed, &account.BudgetHeld,
		&account.StartDate, &account.EndDate, &account.Status,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, api.NewAccountNotFoundError(slurmAccount)
		}
		return nil, api.NewDatabaseError("update account", err)
	}

	return &account, nil
}

// DeleteAccount deletes a budget account
func (q *AccountQueries) DeleteAccount(ctx context.Context, slurmAccount string) error {
	query := `DELETE FROM budget_accounts WHERE slurm_account = $1`

	result, err := q.db.ExecContext(ctx, query, slurmAccount)
	if err != nil {
		return api.NewDatabaseError("delete account", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return api.NewDatabaseError("get affected rows", err)
	}

	if rowsAffected == 0 {
		return api.NewAccountNotFoundError(slurmAccount)
	}

	return nil
}

// GetAccountSummary gets budget summary using the database function
func (q *AccountQueries) GetAccountSummary(ctx context.Context, accountID int64) (*api.BudgetAccount, error) {
	query := `SELECT * FROM get_account_budget_summary($1)`

	var summary struct {
		AccountID       int64
		BudgetLimit     float64
		BudgetUsed      float64
		BudgetHeld      float64
		BudgetAvailable float64
	}

	err := q.db.QueryRowContext(ctx, query, accountID).Scan(
		&summary.AccountID, &summary.BudgetLimit,
		&summary.BudgetUsed, &summary.BudgetHeld, &summary.BudgetAvailable,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, api.NewAccountNotFoundError(fmt.Sprintf("ID:%d", accountID))
		}
		return nil, api.NewDatabaseError("get account summary", err)
	}

	// Get full account details
	return q.GetAccountByID(ctx, accountID)
}

// UpdateAccountBalance updates account balances - called by triggers but available for manual use
func (q *AccountQueries) UpdateAccountBalance(ctx context.Context, accountID int64, budgetUsed, budgetHeld float64) error {
	query := `
		UPDATE budget_accounts
		SET budget_used = $2, budget_held = $3, updated_at = NOW()
		WHERE id = $1`

	result, err := q.db.ExecContext(ctx, query, accountID, budgetUsed, budgetHeld)
	if err != nil {
		return api.NewDatabaseError("update account balance", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return api.NewDatabaseError("get affected rows", err)
	}

	if rowsAffected == 0 {
		return api.NewAccountNotFoundError(fmt.Sprintf("ID:%d", accountID))
	}

	return nil
}
