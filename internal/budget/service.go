// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package budget

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/database"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/pricing"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// Transaction types and statuses, matching the budget_transactions CHECK
// constraints in migrations/001_initial_schema.up.sql.
const (
	txnTypeHold   = "hold"
	txnTypeCharge = "charge"
	txnTypeRefund = "refund"

	txnStatusPending   = "pending"
	txnStatusCompleted = "completed"
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
	pricer             pricing.Pricer
	config             *config.BudgetConfig
}

// NewService creates a new budget service. pricer may be nil; the fleet
// admission path (AdmitFleet) requires it and returns an error when absent,
// while the job-shaped CheckBudget path is unaffected.
func NewService(db *database.DB, advisorClient AdvisorClient, pricer pricing.Pricer, cfg *config.BudgetConfig) *Service {
	return &Service{
		db:                 db,
		accountQueries:     database.NewAccountQueries(db),
		transactionQueries: database.NewTransactionQueries(db),
		advisorClient:      advisorClient,
		pricer:             pricer,
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

	// Get cost estimate from advisor with graceful fallback
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
		log.Warn().Err(err).Msg("Advisor service unavailable, using fallback cost estimation")
		// Graceful fallback: use simple cost estimation
		costResp = s.fallbackCostEstimate(req)
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
		Type:          txnTypeHold,
		Amount:        holdAmount,
		Description:   fmt.Sprintf("Budget hold for job on %s partition", req.Partition),
		Status:        txnStatusPending,
	}

	// Store hold transaction in database
	err = s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.transactionQueries.CreateTransaction(ctx, tx, transaction); err != nil {
			return err
		}
		return s.transactionQueries.UpdateTransactionStatus(ctx, tx, transactionID, txnStatusCompleted)
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

// AdmitFleet is the fleet/resume-shaped spend-rate admission gate (issue #6). It
// is the natural integration point for a cloud-bursting scheduler's Slurm
// ResumeProgram, which has a hostlist but no job walltime.
//
// Semantics (issue #10, model (a) — rate-reservation hold): EstimatedCost and
// HoldAmount in the response are per-HOUR figures, not job totals. The hold
// reserves the fleet's hourly spend rate (rate × DefaultHoldPercentage); it is
// closed at teardown by reconciling against actual cost = rate × runtime. The
// transaction is tagged metadata.kind="fleet_hold" so reconciliation can tell a
// rate-shaped hold from a job-total hold.
func (s *Service) AdmitFleet(ctx context.Context, req *api.FleetAdmissionRequest) (*api.BudgetCheckResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if s.pricer == nil {
		return nil, fmt.Errorf("fleet admission unavailable: no pricing source configured")
	}

	account, err := s.accountQueries.GetAccountByName(ctx, req.Account)
	if err != nil {
		return nil, err
	}
	if !account.IsActive() {
		return nil, api.NewAccountInactiveError(req.Account, account.Status)
	}

	// Price the fleet: per-instance $/hr × node count = fleet spend rate.
	unitRate, err := s.pricer.HourlyRate(ctx, req.InstanceType, req.Region, req.CapacityModel)
	if err != nil {
		return nil, fmt.Errorf("price %s in %s (%s): %w", req.InstanceType, req.Region, req.CapacityModel, err)
	}
	hourlyRate := unitRate * float64(req.Count)

	// Hold reserves the hourly rate plus the configured buffer.
	holdAmount := hourlyRate * s.config.DefaultHoldPercentage
	budgetAvailable := account.BudgetAvailable()

	if holdAmount > budgetAvailable {
		return &api.BudgetCheckResponse{
			Available:       false,
			EstimatedCost:   hourlyRate,
			HoldAmount:      holdAmount,
			Message:         "Insufficient budget for fleet spend rate ($/hr)",
			BudgetRemaining: budgetAvailable,
		}, nil
	}

	transactionID := s.generateTransactionID()
	transaction := &api.BudgetTransaction{
		TransactionID: transactionID,
		AccountID:     account.ID,
		Type:          txnTypeHold,
		Amount:        holdAmount,
		Description: fmt.Sprintf("Fleet spend-rate hold: %d×%s (%s) on %s @ $%.4f/hr",
			req.Count, req.InstanceType, capacityModelOrDefault(req.CapacityModel), req.Partition, hourlyRate),
		Metadata: fmt.Sprintf(`{"kind":"fleet_hold","instance_type":%q,"count":%d,"region":%q,"capacity_model":%q,"hourly_rate":%.6f}`,
			req.InstanceType, req.Count, req.Region, capacityModelOrDefault(req.CapacityModel), hourlyRate),
		Status: txnStatusPending,
	}

	err = s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.transactionQueries.CreateTransaction(ctx, tx, transaction); err != nil {
			return err
		}
		return s.transactionQueries.UpdateTransactionStatus(ctx, tx, transactionID, txnStatusCompleted)
	})
	if err != nil {
		return nil, api.NewTransactionFailedError(transactionID, err)
	}

	resp := &api.BudgetCheckResponse{
		Available:       true,
		EstimatedCost:   hourlyRate,
		HoldAmount:      holdAmount,
		TransactionID:   transactionID,
		Message:         "Fleet admitted (spend rate $/hr held)",
		BudgetRemaining: budgetAvailable - holdAmount,
	}
	resp.Details.AccountBalance = budgetAvailable
	resp.Details.CurrentHold = account.BudgetHeld + holdAmount
	resp.Details.HoldPercentage = s.config.DefaultHoldPercentage
	return resp, nil
}

// capacityModelOrDefault renders an empty capacity model as "on-demand" for
// human-readable descriptions and metadata.
func capacityModelOrDefault(model string) string {
	if model == "" {
		return "on-demand"
	}
	return model
}

// settlementRefund computes the refund that releases the residual hold after the
// actual cost is charged. It is the same for a job-total hold and a fleet-rate
// hold (issue #10): a hold is RESERVED budget to be released in full at
// settlement, while the charge is always the actual $ total. The charge releases
// up to `actualCost` of the held amount (the trigger clamps at zero); this
// refund releases whatever reservation remains. When the actual exceeds the hold
// (common for a fleet rate-reservation that ran many hours), the charge already
// released the whole hold and there is nothing to refund.
//
// Because the hold is always released in full and the actual total is always
// charged, the rate-vs-total dimensional mismatch never produces a double-count:
// the reserved $/hr figure is freed, not charged.
func settlementRefund(heldAmount, actualCost float64) float64 {
	if actualCost < heldAmount {
		return heldAmount - actualCost
	}
	return 0
}

// ReconcileJob reconciles a completed job (or torn-down fleet) with actual costs.
// It books the actual cost as a charge and releases the original hold, linking
// both the charge and any refund to the hold via ParentTransactionID so the
// account-balance trigger actually releases budget_held — without that link the
// hold leaks (issue #10).
func (s *Service) ReconcileJob(ctx context.Context, req *api.JobReconcileRequest) (*api.JobReconcileResponse, error) {
	// Get the original hold transaction
	holdTransaction, err := s.transactionQueries.GetTransaction(ctx, req.TransactionID)
	if err != nil {
		return nil, err
	}

	if holdTransaction.Type != txnTypeHold {
		return nil, api.NewBudgetError(api.ErrCodeValidation, "Transaction is not a hold transaction")
	}

	actualCost := req.ActualCost
	heldAmount := holdTransaction.Amount
	refundAmount := settlementRefund(heldAmount, actualCost)
	parentID := holdTransaction.TransactionID
	heldDescr := "held"
	if isFleetHold(holdTransaction.Metadata) {
		heldDescr = "held rate" // the held figure was a $/hr reservation, not a $ total
	}

	err = s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Create charge transaction for actual cost, linked to the hold so the
		// balance trigger releases held budget rather than treating it as a
		// standalone direct charge.
		chargeID := s.generateTransactionID()
		chargeTransaction := &api.BudgetTransaction{
			TransactionID:       chargeID,
			AccountID:           holdTransaction.AccountID,
			JobID:               &req.JobID,
			Type:                txnTypeCharge,
			Amount:              actualCost,
			Description:         fmt.Sprintf("Actual cost for job %s", req.JobID),
			Status:              txnStatusCompleted,
			ParentTransactionID: &parentID,
		}

		if err := s.transactionQueries.CreateTransaction(ctx, tx, chargeTransaction); err != nil {
			return err
		}

		// Release any residual reservation via a refund linked to the hold.
		if refundAmount > 0 {
			refundID := s.generateTransactionID()
			refundTransaction := &api.BudgetTransaction{
				TransactionID:       refundID,
				AccountID:           holdTransaction.AccountID,
				JobID:               &req.JobID,
				Type:                txnTypeRefund,
				Amount:              refundAmount,
				Description:         fmt.Sprintf("Release for job %s (%s: %.4f, actual: %.4f)", req.JobID, heldDescr, heldAmount, actualCost),
				Status:              txnStatusCompleted,
				ParentTransactionID: &parentID,
			}

			if err := s.transactionQueries.CreateTransaction(ctx, tx, refundTransaction); err != nil {
				return err
			}
		}

		// Mark original hold as completed
		return s.transactionQueries.UpdateTransactionStatus(ctx, tx, req.TransactionID, txnStatusCompleted)
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

// isFleetHold reports whether a hold transaction's metadata marks it as a
// fleet/resume-shaped rate hold (AdmitFleet tags metadata.kind="fleet_hold").
// It is used only for accurate descriptions; the settlement math is identical
// for rate and total holds (see settlementRefund).
func isFleetHold(metadata string) bool {
	return strings.Contains(metadata, `"kind":"fleet_hold"`)
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
			log.Warn().Str("transaction_id", hold.TransactionID).Msg("Canceling very old orphaned hold")

			err := s.db.WithTransaction(ctx, func(tx *sql.Tx) error {
				// Cancel the hold
				if err := s.transactionQueries.UpdateTransactionStatus(ctx, tx, hold.TransactionID, "cancelled"); err != nil {
					return err
				}

				// Create refund transaction, linked to the orphaned hold so the
				// balance trigger releases budget_held (without the parent link it
				// can't tell which bucket to credit — issue #10).
				refundID := s.generateTransactionID()
				parentID := hold.TransactionID
				refundTransaction := &api.BudgetTransaction{
					TransactionID:       refundID,
					AccountID:           hold.AccountID,
					Type:                txnTypeRefund,
					Amount:              hold.Amount,
					Description:         fmt.Sprintf("Recovery refund for orphaned hold %s", hold.TransactionID),
					Status:              txnStatusCompleted,
					ParentTransactionID: &parentID,
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
// txnSeq is a process-wide monotonic counter that guarantees transaction IDs
// are unique even when generated within the same nanosecond (which a bare
// time-based ID cannot — both UnixNano and UnixMicro can repeat on fast hosts).
var txnSeq atomic.Uint64

func (s *Service) generateTransactionID() string {
	return fmt.Sprintf("txn_%d_%d", time.Now().UnixNano(), txnSeq.Add(1))
}

// fallbackCostEstimate provides cost estimation when advisor service is unavailable
func (s *Service) fallbackCostEstimate(req *api.BudgetCheckRequest) *CostEstimateResponse {
	// Simple heuristic-based cost estimation for operational independence
	baseCostPerCPUHour := 0.10 // $0.10/CPU-hour default

	// Parse wall time (simple parsing)
	duration := 1.0 // Default 1 hour
	if strings.Contains(req.WallTime, ":") {
		parts := strings.Split(req.WallTime, ":")
		if len(parts) >= 1 {
			if hours, err := strconv.ParseFloat(parts[0], 64); err == nil {
				duration = hours
				if len(parts) >= 2 {
					if minutes, err := strconv.ParseFloat(parts[1], 64); err == nil {
						duration += minutes / 60.0
					}
				}
			}
		}
	}

	// Calculate base cost
	cpuCost := float64(req.Nodes*req.CPUs) * baseCostPerCPUHour * duration

	// GPU premium
	gpuCost := 0.0
	if req.GPUs > 0 {
		gpuCost = float64(req.GPUs) * baseCostPerCPUHour * 20.0 * duration // 20x premium for GPUs
	}

	// Partition-based adjustments
	partitionMultiplier := 1.0
	partition := strings.ToLower(req.Partition)
	switch {
	case strings.Contains(partition, "gpu"):
		partitionMultiplier = 2.0
	case strings.Contains(partition, "aws"):
		partitionMultiplier = 1.5
	case strings.Contains(partition, "debug"):
		partitionMultiplier = 0.5
	}

	totalCost := (cpuCost + gpuCost) * partitionMultiplier

	// Ensure minimum cost
	if totalCost < 0.01 {
		totalCost = 0.01
	}

	return &CostEstimateResponse{
		EstimatedCost:  totalCost,
		Confidence:     0.6, // Moderate confidence for fallback estimates
		Recommendation: "Fallback cost estimate - advisor service unavailable",
	}
}

// HealthCheck performs a health check on the service
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.db.HealthCheck(ctx)
}
