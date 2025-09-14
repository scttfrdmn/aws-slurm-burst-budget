// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package budget

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/database"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// AdvisorClient defines the interface for cost estimation
type AdvisorClient interface {
	EstimateCost(ctx context.Context, req *CostEstimateRequest) (*CostEstimateResponse, error)
}

// CostEstimateRequest represents a cost estimation request
type CostEstimateRequest struct {
	Account   string            `json:"account"`
	Partition string            `json:"partition"`
	Nodes     int               `json:"nodes"`
	CPUs      int               `json:"cpus"`
	GPUs      int               `json:"gpus,omitempty"`
	Memory    string            `json:"memory,omitempty"`
	WallTime  string            `json:"wall_time"`
	JobScript string            `json:"job_script,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// CostEstimateResponse represents a cost estimation response
type CostEstimateResponse struct {
	EstimatedCost  float64 `json:"estimated_cost"`
	Confidence     float64 `json:"confidence"`
	Recommendation string  `json:"recommendation,omitempty"`
}

// Service provides budget management operations
type Service struct {
	db                 *database.DB
	accountQueries     *database.AccountQueries
	transactionQueries *database.TransactionQueries
	advisorClient      AdvisorClient
	config             *config.BudgetConfig
}

// NewService creates a new budget service
func NewService(db *database.DB, advisorClient AdvisorClient, cfg *config.BudgetConfig) *Service {
	return &Service{
		db:                 db,
		accountQueries:     database.NewAccountQueries(db),
		transactionQueries: database.NewTransactionQueries(db),
		advisorClient:      advisorClient,
		config:             cfg,
	}
}

// CheckBudget checks if a job submission can be accommodated within the budget
func (s *Service) CheckBudget(ctx context.Context, req *api.BudgetCheckRequest) (*api.BudgetCheckResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get account information
	account, err := s.accountQueries.GetAccountByName(ctx, req.Account)
	if err != nil {
		return nil, err
	}

	// Check if account is active
	if !account.IsActive() {
		return nil, api.NewAccountInactiveError(req.Account, account.Status)
	}

	// Get cost estimate from advisor
	costReq := &CostEstimateRequest{
		Account:   req.Account,
		Partition: req.Partition,
		Nodes:     req.Nodes,
		CPUs:      req.CPUs,
		GPUs:      req.GPUs,
		Memory:    req.Memory,
		WallTime:  req.WallTime,
		JobScript: req.JobScript,
	}

	costResp, err := s.advisorClient.EstimateCost(ctx, costReq)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get cost estimate from advisor")
		return nil, api.NewServiceUnavailableError("advisor", err)
	}

	// Calculate hold amount with buffer
	holdAmount := costResp.EstimatedCost * s.config.DefaultHoldPercentage
	budgetAvailable := account.BudgetAvailable()

	// Check if sufficient budget is available
	if holdAmount > budgetAvailable {
		return &api.BudgetCheckResponse{
			Available:       false,
			EstimatedCost:   costResp.EstimatedCost,
			HoldAmount:      holdAmount,
			Message:         "Insufficient budget",
			BudgetRemaining: budgetAvailable,
			Details: struct {
				AccountBalance    float64 `json:"account_balance"`
				CurrentHold       float64 `json:"current_hold"`
				PartitionUsed     float64 `json:"partition_used,omitempty"`
				PartitionLimit    float64 `json:"partition_limit,omitempty"`
				HoldPercentage    float64 `json:"hold_percentage"`
				AdvisorConfidence float64 `json:"advisor_confidence,omitempty"`
			}{
				AccountBalance:    budgetAvailable,
				CurrentHold:       account.BudgetHeld,
				HoldPercentage:    s.config.DefaultHoldPercentage,
				AdvisorConfidence: costResp.Confidence,
			},
		}, nil
	}

	// Create hold transaction
	transactionID := s.generateTransactionID()
	transaction := &api.BudgetTransaction{
		TransactionID: transactionID,
		AccountID:     account.ID,
		Type:          "hold",
		Amount:        holdAmount,
		Description:   fmt.Sprintf("Budget hold for job on %s partition", req.Partition),
		Status:        "pending",
	}

	// Store hold transaction in database
	err = s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.transactionQueries.CreateTransaction(ctx, tx, transaction); err != nil {
			return err
		}
		return s.transactionQueries.UpdateTransactionStatus(ctx, tx, transactionID, "completed")
	})

	if err != nil {
		return nil, api.NewTransactionFailedError(transactionID, err)
	}

	return &api.BudgetCheckResponse{
		Available:       true,
		EstimatedCost:   costResp.EstimatedCost,
		HoldAmount:      holdAmount,
		TransactionID:   transactionID,
		Message:         "Budget check passed",
		BudgetRemaining: budgetAvailable - holdAmount,
		Recommendation:  costResp.Recommendation,
		Details: struct {
			AccountBalance    float64 `json:"account_balance"`
			CurrentHold       float64 `json:"current_hold"`
			PartitionUsed     float64 `json:"partition_used,omitempty"`
			PartitionLimit    float64 `json:"partition_limit,omitempty"`
			HoldPercentage    float64 `json:"hold_percentage"`
			AdvisorConfidence float64 `json:"advisor_confidence,omitempty"`
		}{
			AccountBalance:    budgetAvailable,
			CurrentHold:       account.BudgetHeld + holdAmount,
			HoldPercentage:    s.config.DefaultHoldPercentage,
			AdvisorConfidence: costResp.Confidence,
		},
	}, nil
}

// ReconcileJob reconciles a completed job with actual costs
func (s *Service) ReconcileJob(ctx context.Context, req *api.JobReconcileRequest) (*api.JobReconcileResponse, error) {
	// Get the original hold transaction
	holdTransaction, err := s.transactionQueries.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		return nil, err
	}

	if holdTransaction.Type != "hold" {
		return nil, api.NewBudgetError(api.ErrCodeValidation, "Transaction is not a hold transaction")
	}

	// Calculate refund/additional charge
	actualCost := req.ActualCost
	heldAmount := holdTransaction.Amount
	var refundAmount float64

	if actualCost < heldAmount {
		refundAmount = heldAmount - actualCost
	}
	// Note: additionalCharge not used in current implementation
	// Future versions could handle cases where actual cost exceeds held amount

	err = s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Create charge transaction for actual cost
		chargeID := s.generateTransactionID()
		chargeTransaction := &api.BudgetTransaction{
			TransactionID: chargeID,
			AccountID:     holdTransaction.AccountID,
			JobID:         &req.JobID,
			Type:          "charge",
			Amount:        actualCost,
			Description:   fmt.Sprintf("Actual cost for job %s", req.JobID),
			Status:        "completed",
		}

		if err := s.transactionQueries.CreateTransaction(ctx, tx, chargeTransaction); err != nil {
			return err
		}

		// Create refund transaction if needed
		if refundAmount > 0 {
			refundID := s.generateTransactionID()
			refundTransaction := &api.BudgetTransaction{
				TransactionID: refundID,
				AccountID:     holdTransaction.AccountID,
				JobID:         &req.JobID,
				Type:          "refund",
				Amount:        refundAmount,
				Description:   fmt.Sprintf("Refund for job %s (held: %.2f, actual: %.2f)", req.JobID, heldAmount, actualCost),
				Status:        "completed",
			}

			if err := s.transactionQueries.CreateTransaction(ctx, tx, refundTransaction); err != nil {
				return err
			}
		}

		// Mark original hold as completed
		return s.transactionQueries.UpdateTransactionStatus(ctx, tx, req.TransactionID, "completed")
	})

	if err != nil {
		return nil, api.NewTransactionFailedError(req.TransactionID, err)
	}

	return &api.JobReconcileResponse{
		Success:       true,
		OriginalHold:  heldAmount,
		ActualCharge:  actualCost,
		RefundAmount:  refundAmount,
		TransactionID: req.TransactionID,
		Message:       "Job reconciliation completed successfully",
	}, nil
}

// CreateAccount creates a new budget account
func (s *Service) CreateAccount(ctx context.Context, req *api.CreateAccountRequest) (*api.BudgetAccount, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return s.accountQueries.CreateAccount(ctx, req)
}

// GetAccount retrieves a budget account by name
func (s *Service) GetAccount(ctx context.Context, slurmAccount string) (*api.BudgetAccount, error) {
	return s.accountQueries.GetAccountByName(ctx, slurmAccount)
}

// ListAccounts lists budget accounts
func (s *Service) ListAccounts(ctx context.Context, req *api.ListAccountsRequest) ([]*api.BudgetAccount, error) {
	return s.accountQueries.ListAccounts(ctx, req)
}

// UpdateAccount updates a budget account
func (s *Service) UpdateAccount(ctx context.Context, slurmAccount string, req *api.UpdateAccountRequest) (*api.BudgetAccount, error) {
	return s.accountQueries.UpdateAccount(ctx, slurmAccount, req)
}

// DeleteAccount deletes a budget account
func (s *Service) DeleteAccount(ctx context.Context, slurmAccount string) error {
	return s.accountQueries.DeleteAccount(ctx, slurmAccount)
}

// ListTransactions lists transactions with filtering
func (s *Service) ListTransactions(ctx context.Context, req *api.TransactionListRequest) ([]*api.BudgetTransaction, error) {
	return s.transactionQueries.ListTransactions(ctx, req)
}

// RecoverOrphanedTransactions recovers transactions that may have been orphaned
func (s *Service) RecoverOrphanedTransactions(ctx context.Context) error {
	if !s.config.AutoRecoveryEnabled {
		return nil
	}

	pendingHolds, err := s.transactionQueries.GetPendingHolds(ctx, s.config.ReconciliationTimeout)
	if err != nil {
		return err
	}

	log.Info().Int("count", len(pendingHolds)).Msg("Found orphaned hold transactions for recovery")

	for _, hold := range pendingHolds {
		// In a real implementation, you would check with SLURM if the job completed
		// For now, we'll just log and potentially cancel very old holds
		if time.Since(hold.CreatedAt) > s.config.ReconciliationTimeout*2 {
			log.Warn().Str("transaction_id", hold.TransactionID).Msg("Cancelling very old orphaned hold")

			err := s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
				// Cancel the hold
				if err := s.transactionQueries.UpdateTransactionStatus(ctx, tx, hold.TransactionID, "cancelled"); err != nil {
					return err
				}

				// Create refund transaction
				refundID := s.generateTransactionID()
				refundTransaction := &api.BudgetTransaction{
					TransactionID: refundID,
					AccountID:     hold.AccountID,
					Type:          "refund",
					Amount:        hold.Amount,
					Description:   fmt.Sprintf("Recovery refund for orphaned hold %s", hold.TransactionID),
					Status:        "completed",
				}

				return s.transactionQueries.CreateTransaction(ctx, tx, refundTransaction)
			})

			if err != nil {
				log.Error().Err(err).Str("transaction_id", hold.TransactionID).Msg("Failed to recover orphaned transaction")
			}
		}
	}

	return nil
}

// generateTransactionID generates a unique transaction ID
func (s *Service) generateTransactionID() string {
	return fmt.Sprintf("txn_%d_%d", time.Now().UnixNano(), time.Now().UnixMicro()%1000000)
}

// HealthCheck performs a health check on the service
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.db.HealthCheck(ctx)
}
