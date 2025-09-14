// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// TransactionQueries provides database operations for budget transactions
type TransactionQueries struct {
	db *DB
}

// NewTransactionQueries creates a new TransactionQueries instance
func NewTransactionQueries(db *DB) *TransactionQueries {
	return &TransactionQueries{db: db}
}

// CreateTransaction creates a new budget transaction
func (q *TransactionQueries) CreateTransaction(ctx context.Context, tx *sql.Tx, transaction *api.BudgetTransaction) error {
	query := `
		INSERT INTO budget_transactions (transaction_id, account_id, job_id, type, amount, description, metadata, status, parent_transaction_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	var execer interface {
		QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	}

	if tx != nil {
		execer = tx
	} else {
		execer = q.db
	}

	err := execer.QueryRowContext(ctx, query,
		transaction.TransactionID,
		transaction.AccountID,
		transaction.JobID,
		transaction.Type,
		transaction.Amount,
		transaction.Description,
		transaction.Metadata,
		transaction.Status,
		nil, // parent_transaction_id - set separately if needed
	).Scan(&transaction.ID, &transaction.CreatedAt)

	if err != nil {
		return api.NewDatabaseError("create transaction", err)
	}

	return nil
}

// GetTransaction retrieves a transaction by ID
func (q *TransactionQueries) GetTransaction(ctx context.Context, transactionID string) (*api.BudgetTransaction, error) {
	query := `
		SELECT id, transaction_id, account_id, job_id, type, amount, description, metadata, status, created_at, completed_at
		FROM budget_transactions
		WHERE transaction_id = $1`

	var transaction api.BudgetTransaction
	err := q.db.QueryRowContext(ctx, query, transactionID).Scan(
		&transaction.ID,
		&transaction.TransactionID,
		&transaction.AccountID,
		&transaction.JobID,
		&transaction.Type,
		&transaction.Amount,
		&transaction.Description,
		&transaction.Metadata,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, api.NewBudgetError(api.ErrCodeNotFound, fmt.Sprintf("Transaction %s not found", transactionID))
		}
		return nil, api.NewDatabaseError("get transaction", err)
	}

	return &transaction, nil
}

// UpdateTransactionStatus updates a transaction's status
func (q *TransactionQueries) UpdateTransactionStatus(ctx context.Context, tx *sql.Tx, transactionID string, status string) error {
	query := `
		UPDATE budget_transactions
		SET status = $2, completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END
		WHERE transaction_id = $1`

	var execer interface {
		ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	}

	if tx != nil {
		execer = tx
	} else {
		execer = q.db
	}

	result, err := execer.ExecContext(ctx, query, transactionID, status)
	if err != nil {
		return api.NewDatabaseError("update transaction status", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return api.NewDatabaseError("get affected rows", err)
	}

	if rowsAffected == 0 {
		return api.NewBudgetError(api.ErrCodeNotFound, fmt.Sprintf("Transaction %s not found", transactionID))
	}

	return nil
}

// ListTransactions retrieves transactions with filtering
func (q *TransactionQueries) ListTransactions(ctx context.Context, req *api.TransactionListRequest) ([]*api.BudgetTransaction, error) {
	baseQuery := `
		SELECT bt.id, bt.transaction_id, bt.account_id, bt.job_id, bt.type, bt.amount,
		       bt.description, bt.metadata, bt.status, bt.created_at, bt.completed_at
		FROM budget_transactions bt`

	var joins []string
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Join with accounts if filtering by account name
	if req.Account != "" {
		joins = append(joins, "JOIN budget_accounts ba ON bt.account_id = ba.id")
		conditions = append(conditions, fmt.Sprintf("ba.slurm_account = $%d", argIndex))
		args = append(args, req.Account)
		argIndex++
	}

	if req.JobID != "" {
		conditions = append(conditions, fmt.Sprintf("bt.job_id = $%d", argIndex))
		args = append(args, req.JobID)
		argIndex++
	}

	if req.Type != "" {
		conditions = append(conditions, fmt.Sprintf("bt.type = $%d", argIndex))
		args = append(args, req.Type)
		argIndex++
	}

	if req.Status != "" {
		conditions = append(conditions, fmt.Sprintf("bt.status = $%d", argIndex))
		args = append(args, req.Status)
		argIndex++
	}

	if req.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("bt.created_at >= $%d", argIndex))
		args = append(args, *req.StartDate)
		argIndex++
	}

	if req.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("bt.created_at <= $%d", argIndex))
		args = append(args, *req.EndDate)
		argIndex++
	}

	// Build final query
	if len(joins) > 0 {
		baseQuery += " " + strings.Join(joins, " ")
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY bt.created_at DESC"

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
		return nil, api.NewDatabaseError("list transactions", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Database row close failed - log for debugging
			_ = err // Acknowledge error is handled
		}
	}()

	var transactions []*api.BudgetTransaction
	for rows.Next() {
		var transaction api.BudgetTransaction
		err := rows.Scan(
			&transaction.ID,
			&transaction.TransactionID,
			&transaction.AccountID,
			&transaction.JobID,
			&transaction.Type,
			&transaction.Amount,
			&transaction.Description,
			&transaction.Metadata,
			&transaction.Status,
			&transaction.CreatedAt,
			&transaction.CompletedAt,
		)
		if err != nil {
			return nil, api.NewDatabaseError("scan transaction row", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, nil
}

// GetPendingHolds retrieves pending hold transactions for reconciliation
func (q *TransactionQueries) GetPendingHolds(ctx context.Context, olderThan time.Duration) ([]*api.BudgetTransaction, error) {
	query := `
		SELECT id, transaction_id, account_id, job_id, type, amount, description, metadata, status, created_at, completed_at
		FROM budget_transactions
		WHERE type = 'hold' AND status = 'pending' AND created_at < $1`

	cutoff := time.Now().Add(-olderThan)

	rows, err := q.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, api.NewDatabaseError("get pending holds", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Database row close failed - log for debugging
			_ = err // Acknowledge error is handled
		}
	}()

	var transactions []*api.BudgetTransaction
	for rows.Next() {
		var transaction api.BudgetTransaction
		err := rows.Scan(
			&transaction.ID,
			&transaction.TransactionID,
			&transaction.AccountID,
			&transaction.JobID,
			&transaction.Type,
			&transaction.Amount,
			&transaction.Description,
			&transaction.Metadata,
			&transaction.Status,
			&transaction.CreatedAt,
			&transaction.CompletedAt,
		)
		if err != nil {
			return nil, api.NewDatabaseError("scan pending hold", err)
		}
		transactions = append(transactions, &transaction)
	}

	return transactions, nil
}
