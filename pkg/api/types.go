// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"fmt"
	"time"
)

// BudgetAccount represents a budget account in the system
type BudgetAccount struct {
	ID                   int64      `json:"id" db:"id"`
	SlurmAccount         string     `json:"slurm_account" db:"slurm_account"`
	Name                 string     `json:"name" db:"name"`
	Description          string     `json:"description" db:"description"`
	BudgetLimit          float64    `json:"budget_limit" db:"budget_limit"`
	BudgetUsed           float64    `json:"budget_used" db:"budget_used"`
	BudgetHeld           float64    `json:"budget_held" db:"budget_held"`
	HasIncrementalBudget bool       `json:"has_incremental_budget" db:"has_incremental_budget"`
	NextAllocationDate   *time.Time `json:"next_allocation_date,omitempty" db:"next_allocation_date"`
	TotalAllocated       float64    `json:"total_allocated" db:"total_allocated"`
	StartDate            time.Time  `json:"start_date" db:"start_date"`
	EndDate              time.Time  `json:"end_date" db:"end_date"`
	Status               string     `json:"status" db:"status"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// BudgetAvailable returns the available budget amount
func (ba *BudgetAccount) BudgetAvailable() float64 {
	return ba.BudgetLimit - ba.BudgetUsed - ba.BudgetHeld
}

// IsActive returns true if the account is currently active
func (ba *BudgetAccount) IsActive() bool {
	now := time.Now()
	return ba.Status == "active" && now.After(ba.StartDate) && now.Before(ba.EndDate)
}

// BudgetTransaction represents a budget transaction
type BudgetTransaction struct {
	ID            int64      `json:"id" db:"id"`
	AccountID     int64      `json:"account_id" db:"account_id"`
	JobID         *string    `json:"job_id,omitempty" db:"job_id"`
	TransactionID string     `json:"transaction_id" db:"transaction_id"`
	Type          string     `json:"type" db:"type"` // hold, charge, refund, adjustment
	Amount        float64    `json:"amount" db:"amount"`
	Description   string     `json:"description" db:"description"`
	Metadata      string     `json:"metadata,omitempty" db:"metadata"` // JSON metadata
	Status        string     `json:"status" db:"status"`               // pending, completed, failed, cancelled
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// BudgetPartitionLimit represents per-partition budget limits
type BudgetPartitionLimit struct {
	ID        int64   `json:"id" db:"id"`
	AccountID int64   `json:"account_id" db:"account_id"`
	Partition string  `json:"partition" db:"partition"`
	Limit     float64 `json:"limit" db:"limit"`
	Used      float64 `json:"used" db:"used"`
	Held      float64 `json:"held" db:"held"`
}

// Available returns the available budget for the partition
func (bpl *BudgetPartitionLimit) Available() float64 {
	return bpl.Limit - bpl.Used - bpl.Held
}

// BudgetAllocationSchedule represents an incremental budget allocation schedule
type BudgetAllocationSchedule struct {
	ID                  int64      `json:"id" db:"id"`
	AccountID           int64      `json:"account_id" db:"account_id"`
	TotalBudget         float64    `json:"total_budget" db:"total_budget"`
	AllocationAmount    float64    `json:"allocation_amount" db:"allocation_amount"`
	AllocationFrequency string     `json:"allocation_frequency" db:"allocation_frequency"` // daily, weekly, monthly, quarterly, yearly
	StartDate           time.Time  `json:"start_date" db:"start_date"`
	EndDate             *time.Time `json:"end_date,omitempty" db:"end_date"`
	NextAllocationDate  time.Time  `json:"next_allocation_date" db:"next_allocation_date"`
	AllocatedToDate     float64    `json:"allocated_to_date" db:"allocated_to_date"`
	RemainingBudget     float64    `json:"remaining_budget" db:"remaining_budget"`
	Status              string     `json:"status" db:"status"` // active, paused, completed, cancelled
	AutoAllocate        bool       `json:"auto_allocate" db:"auto_allocate"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// BudgetAllocation represents a single budget allocation event
type BudgetAllocation struct {
	ID               int64     `json:"id" db:"id"`
	ScheduleID       int64     `json:"schedule_id" db:"schedule_id"`
	AccountID        int64     `json:"account_id" db:"account_id"`
	AllocationAmount float64   `json:"allocation_amount" db:"allocation_amount"`
	AllocatedDate    time.Time `json:"allocated_date" db:"allocated_date"`
	TransactionID    string    `json:"transaction_id" db:"transaction_id"`
	Notes            string    `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// AllocationScheduleSummary provides summary information about allocation schedules
type AllocationScheduleSummary struct {
	TotalBudget          float64    `json:"total_budget"`
	AllocatedToDate      float64    `json:"allocated_to_date"`
	RemainingBudget      float64    `json:"remaining_budget"`
	NextAllocationDate   *time.Time `json:"next_allocation_date,omitempty"`
	NextAllocationAmount float64    `json:"next_allocation_amount"`
	AllocationFrequency  string     `json:"allocation_frequency,omitempty"`
}

// Request and Response Types

// CreateAccountRequest represents a request to create a new budget account
type CreateAccountRequest struct {
	SlurmAccount         string                           `json:"slurm_account" validate:"required"`
	Name                 string                           `json:"name" validate:"required"`
	Description          string                           `json:"description"`
	BudgetLimit          float64                          `json:"budget_limit" validate:"required,min=0"`
	StartDate            time.Time                        `json:"start_date" validate:"required"`
	EndDate              time.Time                        `json:"end_date" validate:"required,gtfield=StartDate"`
	HasIncrementalBudget bool                             `json:"has_incremental_budget"`
	AllocationSchedule   *CreateAllocationScheduleRequest `json:"allocation_schedule,omitempty"`
}

// CreateAllocationScheduleRequest represents a request to create an allocation schedule
type CreateAllocationScheduleRequest struct {
	TotalBudget         float64    `json:"total_budget" validate:"required,min=0"`
	AllocationAmount    float64    `json:"allocation_amount" validate:"required,min=0"`
	AllocationFrequency string     `json:"allocation_frequency" validate:"required,oneof=daily weekly monthly quarterly yearly"`
	StartDate           time.Time  `json:"start_date" validate:"required"`
	EndDate             *time.Time `json:"end_date,omitempty"`
	AutoAllocate        bool       `json:"auto_allocate"`
}

// UpdateAccountRequest represents a request to update a budget account
type UpdateAccountRequest struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	BudgetLimit *float64   `json:"budget_limit,omitempty" validate:"omitempty,min=0"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Status      *string    `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
}

// ListAccountsRequest represents a request to list budget accounts
type ListAccountsRequest struct {
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset int    `json:"offset,omitempty" validate:"omitempty,min=0"`
	Status string `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
}

// BudgetCheckRequest represents a request to check budget availability
type BudgetCheckRequest struct {
	Account    string            `json:"account" validate:"required"`
	Partition  string            `json:"partition" validate:"required"`
	Nodes      int               `json:"nodes" validate:"required,min=1"`
	CPUs       int               `json:"cpus" validate:"required,min=1"`
	GPUs       int               `json:"gpus,omitempty" validate:"omitempty,min=0"`
	Memory     string            `json:"memory,omitempty"`
	WallTime   string            `json:"wall_time" validate:"required"`
	JobScript  string            `json:"job_script,omitempty"`
	UserID     string            `json:"user_id,omitempty"`
	JobDetails map[string]string `json:"job_details,omitempty"`
}

// BudgetCheckResponse represents a response to budget check request
type BudgetCheckResponse struct {
	Available       bool    `json:"available"`
	EstimatedCost   float64 `json:"estimated_cost"`
	HoldAmount      float64 `json:"hold_amount"`
	TransactionID   string  `json:"transaction_id,omitempty"`
	Message         string  `json:"message,omitempty"`
	BudgetRemaining float64 `json:"budget_remaining"`
	Recommendation  string  `json:"recommendation,omitempty"`
	Details         struct {
		AccountBalance    float64 `json:"account_balance"`
		CurrentHold       float64 `json:"current_hold"`
		PartitionUsed     float64 `json:"partition_used,omitempty"`
		PartitionLimit    float64 `json:"partition_limit,omitempty"`
		HoldPercentage    float64 `json:"hold_percentage"`
		AdvisorConfidence float64 `json:"advisor_confidence,omitempty"`
	} `json:"details,omitempty"`
}

// JobReconcileRequest represents a request to reconcile a completed job
type JobReconcileRequest struct {
	JobID         string  `json:"job_id" validate:"required"`
	ActualCost    float64 `json:"actual_cost" validate:"required,min=0"`
	TransactionID string  `json:"transaction_id" validate:"required"`
	JobMetadata   string  `json:"job_metadata,omitempty"` // JSON metadata
}

// JobReconcileResponse represents a response to job reconciliation
type JobReconcileResponse struct {
	Success       bool    `json:"success"`
	OriginalHold  float64 `json:"original_hold"`
	ActualCharge  float64 `json:"actual_charge"`
	RefundAmount  float64 `json:"refund_amount"`
	TransactionID string  `json:"transaction_id"`
	Message       string  `json:"message,omitempty"`
}

// UsageReportRequest represents a request for usage reporting
type UsageReportRequest struct {
	Account   string     `json:"account,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Partition string     `json:"partition,omitempty"`
	GroupBy   string     `json:"group_by,omitempty" validate:"omitempty,oneof=day week month partition user"`
}

// UsageReportResponse represents usage report data
type UsageReportResponse struct {
	Account   string               `json:"account"`
	Period    string               `json:"period"`
	Summary   UsageSummary         `json:"summary"`
	Breakdown []UsageBreakdownItem `json:"breakdown,omitempty"`
	Forecast  *UsageForecast       `json:"forecast,omitempty"`
}

// UsageSummary provides summary statistics
type UsageSummary struct {
	TotalSpent     float64 `json:"total_spent"`
	TotalHeld      float64 `json:"total_held"`
	TotalJobs      int64   `json:"total_jobs"`
	AvgCostPerJob  float64 `json:"avg_cost_per_job"`
	BudgetUtilized float64 `json:"budget_utilized"` // percentage
}

// UsageBreakdownItem represents a breakdown item in usage reports
type UsageBreakdownItem struct {
	Category   string  `json:"category"`
	Label      string  `json:"label"`
	Amount     float64 `json:"amount"`
	JobCount   int64   `json:"job_count"`
	Percentage float64 `json:"percentage"`
}

// UsageForecast provides budget forecasting information
type UsageForecast struct {
	ProjectedSpend     float64   `json:"projected_spend"`
	ProjectedDepletion time.Time `json:"projected_depletion,omitempty"`
	BurnRate           float64   `json:"burn_rate"` // per day
	Confidence         float64   `json:"confidence"`
	Recommendation     string    `json:"recommendation,omitempty"`
}

// TransactionListRequest represents a request to list transactions
type TransactionListRequest struct {
	Account   string     `json:"account,omitempty"`
	JobID     string     `json:"job_id,omitempty"`
	Type      string     `json:"type,omitempty" validate:"omitempty,oneof=hold charge refund adjustment allocation"`
	Status    string     `json:"status,omitempty" validate:"omitempty,oneof=pending completed failed cancelled"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Limit     int        `json:"limit,omitempty" validate:"omitempty,min=1,max=1000"`
	Offset    int        `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// AllocationScheduleRequest represents a request to list allocation schedules
type AllocationScheduleRequest struct {
	Account string `json:"account,omitempty"`
	Status  string `json:"status,omitempty" validate:"omitempty,oneof=active paused completed cancelled"`
	Limit   int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset  int    `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// UpdateAllocationScheduleRequest represents a request to update an allocation schedule
type UpdateAllocationScheduleRequest struct {
	AllocationAmount    *float64   `json:"allocation_amount,omitempty" validate:"omitempty,min=0"`
	AllocationFrequency *string    `json:"allocation_frequency,omitempty" validate:"omitempty,oneof=daily weekly monthly quarterly yearly"`
	EndDate             *time.Time `json:"end_date,omitempty"`
	Status              *string    `json:"status,omitempty" validate:"omitempty,oneof=active paused completed cancelled"`
	AutoAllocate        *bool      `json:"auto_allocate,omitempty"`
}

// ProcessAllocationsRequest represents a request to manually process allocations
type ProcessAllocationsRequest struct {
	AccountID  *int64 `json:"account_id,omitempty"`
	ScheduleID *int64 `json:"schedule_id,omitempty"`
	DryRun     bool   `json:"dry_run"`
}

// ProcessAllocationsResponse represents the response from processing allocations
type ProcessAllocationsResponse struct {
	ProcessedCount int64                 `json:"processed_count"`
	TotalAllocated float64               `json:"total_allocated"`
	Allocations    []ProcessedAllocation `json:"allocations,omitempty"`
	DryRun         bool                  `json:"dry_run"`
}

// Grant Management Request/Response Types

// CreateGrantRequest represents a request to create a new grant account
type CreateGrantRequest struct {
	GrantNumber            string    `json:"grant_number" validate:"required"`
	FundingAgency          string    `json:"funding_agency" validate:"required"`
	AgencyProgram          string    `json:"agency_program,omitempty"`
	PrincipalInvestigator  string    `json:"principal_investigator" validate:"required"`
	CoInvestigators        []string  `json:"co_investigators,omitempty"`
	Institution            string    `json:"institution" validate:"required"`
	Department             string    `json:"department,omitempty"`
	GrantStartDate         time.Time `json:"grant_start_date" validate:"required"`
	GrantEndDate           time.Time `json:"grant_end_date" validate:"required,gtfield=GrantStartDate"`
	TotalAwardAmount       float64   `json:"total_award_amount" validate:"required,min=0"`
	IndirectCostRate       float64   `json:"indirect_cost_rate" validate:"min=0,max=1"`
	BudgetPeriodMonths     int       `json:"budget_period_months" validate:"min=1,max=60"`
	ComplianceRequirements string    `json:"compliance_requirements,omitempty"`
	FederalAwardID         string    `json:"federal_award_id,omitempty"`
	InternalProjectCode    string    `json:"internal_project_code,omitempty"`
	CostCenter             string    `json:"cost_center,omitempty"`
}

// BurnRateAnalysisRequest represents a request for burn rate analysis
type BurnRateAnalysisRequest struct {
	Account           string     `json:"account,omitempty"`
	GrantNumber       string     `json:"grant_number,omitempty"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	AnalysisPeriod    string     `json:"analysis_period,omitempty" validate:"omitempty,oneof=7d 30d 90d 6m 1y"`
	IncludeProjection bool       `json:"include_projection"`
	IncludeAlerts     bool       `json:"include_alerts"`
}

// BurnRateAnalysisResponse represents burn rate analysis results
type BurnRateAnalysisResponse struct {
	Account         string              `json:"account"`
	GrantNumber     string              `json:"grant_number,omitempty"`
	AnalysisPeriod  string              `json:"analysis_period"`
	TimeRange       TimeRange           `json:"time_range"`
	CurrentMetrics  BurnRateMetrics     `json:"current_metrics"`
	HistoricalData  []BurnRateDataPoint `json:"historical_data"`
	Projection      *BurnRateProjection `json:"projection,omitempty"`
	Alerts          []BudgetAlert       `json:"alerts,omitempty"`
	Recommendations []string            `json:"recommendations"`
}

// TimeRange represents a time period for analysis
type TimeRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Days      int       `json:"days"`
}

// BurnRateMetrics represents current burn rate metrics
type BurnRateMetrics struct {
	DailySpendRate         float64 `json:"daily_spend_rate"`
	DailyExpectedRate      float64 `json:"daily_expected_rate"`
	VariancePercentage     float64 `json:"variance_percentage"`
	Rolling7DayAverage     float64 `json:"rolling_7day_average"`
	Rolling30DayAverage    float64 `json:"rolling_30day_average"`
	CumulativeSpend        float64 `json:"cumulative_spend"`
	CumulativeExpected     float64 `json:"cumulative_expected"`
	CumulativeVariancePct  float64 `json:"cumulative_variance_pct"`
	BudgetHealthScore      float64 `json:"budget_health_score"`
	BudgetRemainingAmount  float64 `json:"budget_remaining_amount"`
	BudgetRemainingPercent float64 `json:"budget_remaining_percent"`
	TimeRemainingDays      int     `json:"time_remaining_days"`
	BurnRateStatus         string  `json:"burn_rate_status"`     // OVERSPENDING, UNDERSPENDING, ON_TRACK
	BudgetHealthStatus     string  `json:"budget_health_status"` // HEALTHY, CONCERN, WARNING, CRITICAL
}

// BurnRateDataPoint represents a single data point in burn rate analysis
type BurnRateDataPoint struct {
	Date               time.Time `json:"date"`
	DailySpend         float64   `json:"daily_spend"`
	DailyExpected      float64   `json:"daily_expected"`
	VariancePercentage float64   `json:"variance_percentage"`
	CumulativeSpend    float64   `json:"cumulative_spend"`
	CumulativeExpected float64   `json:"cumulative_expected"`
	BudgetHealthScore  float64   `json:"budget_health_score"`
}

// BurnRateProjection represents future spending projections
type BurnRateProjection struct {
	ProjectedEndDate       time.Time  `json:"projected_end_date"`
	ProjectedDepletionDate *time.Time `json:"projected_depletion_date,omitempty"`
	ProjectedFinalSpend    float64    `json:"projected_final_spend"`
	ProjectedOverrun       float64    `json:"projected_overrun"`
	ProjectedUnderrun      float64    `json:"projected_underrun"`
	ConfidenceLevel        float64    `json:"confidence_level"`
	ProjectionMethod       string     `json:"projection_method"`
	RiskLevel              string     `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
}

// GrantReportRequest represents a request for grant compliance reports
type GrantReportRequest struct {
	GrantNumber    string     `json:"grant_number" validate:"required"`
	ReportType     string     `json:"report_type" validate:"required,oneof=financial technical compliance annual"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	BudgetPeriod   *int       `json:"budget_period,omitempty"`
	Format         string     `json:"format" validate:"oneof=json csv pdf"`
	IncludeDetails bool       `json:"include_details"`
}

// AlertAcknowledgeRequest represents a request to acknowledge an alert
type AlertAcknowledgeRequest struct {
	AlertID        int64  `json:"alert_id" validate:"required"`
	AcknowledgedBy string `json:"acknowledged_by" validate:"required"`
	Notes          string `json:"notes,omitempty"`
}

// ProcessedAllocation represents a single processed allocation
type ProcessedAllocation struct {
	ScheduleID      int64   `json:"schedule_id"`
	AccountID       int64   `json:"account_id"`
	AllocatedAmount float64 `json:"allocated_amount"`
	TransactionID   string  `json:"transaction_id"`
}

// GrantListRequest represents a request to list grants
type GrantListRequest struct {
	Status        string     `json:"status,omitempty" validate:"omitempty,oneof=pending active suspended completed cancelled"`
	FundingAgency string     `json:"funding_agency,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	ActiveOnly    bool       `json:"active_only"`
	Limit         int        `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset        int        `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// Grant Management Types

// GrantAccount represents a research grant with long-term funding
type GrantAccount struct {
	ID                     int64     `json:"id" db:"id"`
	GrantNumber            string    `json:"grant_number" db:"grant_number"`
	FundingAgency          string    `json:"funding_agency" db:"funding_agency"`
	AgencyProgram          string    `json:"agency_program,omitempty" db:"agency_program"`
	PrincipalInvestigator  string    `json:"principal_investigator" db:"principal_investigator"`
	CoInvestigators        []string  `json:"co_investigators,omitempty" db:"co_investigators"`
	Institution            string    `json:"institution" db:"institution"`
	Department             string    `json:"department,omitempty" db:"department"`
	GrantStartDate         time.Time `json:"grant_start_date" db:"grant_start_date"`
	GrantEndDate           time.Time `json:"grant_end_date" db:"grant_end_date"`
	TotalAwardAmount       float64   `json:"total_award_amount" db:"total_award_amount"`
	DirectCosts            float64   `json:"direct_costs" db:"direct_costs"`
	IndirectCostRate       float64   `json:"indirect_cost_rate" db:"indirect_cost_rate"`
	IndirectCosts          float64   `json:"indirect_costs" db:"indirect_costs"`
	BudgetPeriodMonths     int       `json:"budget_period_months" db:"budget_period_months"`
	CurrentBudgetPeriod    int       `json:"current_budget_period" db:"current_budget_period"`
	Status                 string    `json:"status" db:"status"`
	ComplianceRequirements string    `json:"compliance_requirements,omitempty" db:"compliance_requirements"`
	FederalAwardID         string    `json:"federal_award_id,omitempty" db:"federal_award_id"`
	InternalProjectCode    string    `json:"internal_project_code,omitempty" db:"internal_project_code"`
	CostCenter             string    `json:"cost_center,omitempty" db:"cost_center"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

// GrantBudgetPeriod represents a budget period within a multi-year grant
type GrantBudgetPeriod struct {
	ID                    int64     `json:"id" db:"id"`
	GrantID               int64     `json:"grant_id" db:"grant_id"`
	PeriodNumber          int       `json:"period_number" db:"period_number"`
	PeriodStartDate       time.Time `json:"period_start_date" db:"period_start_date"`
	PeriodEndDate         time.Time `json:"period_end_date" db:"period_end_date"`
	PeriodBudgetAmount    float64   `json:"period_budget_amount" db:"period_budget_amount"`
	PeriodSpentAmount     float64   `json:"period_spent_amount" db:"period_spent_amount"`
	PeriodCommittedAmount float64   `json:"period_committed_amount" db:"period_committed_amount"`
	ExpectedBurnRate      float64   `json:"expected_burn_rate" db:"expected_burn_rate"`
	ActualBurnRate        float64   `json:"actual_burn_rate" db:"actual_burn_rate"`
	BurnRateVariance      float64   `json:"burn_rate_variance" db:"burn_rate_variance"`
	Status                string    `json:"status" db:"status"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// BudgetBurnRate represents daily burn rate tracking
type BudgetBurnRate struct {
	ID                     int64      `json:"id" db:"id"`
	AccountID              int64      `json:"account_id" db:"account_id"`
	MeasurementDate        time.Time  `json:"measurement_date" db:"measurement_date"`
	DailySpendAmount       float64    `json:"daily_spend_amount" db:"daily_spend_amount"`
	DailyExpectedAmount    float64    `json:"daily_expected_amount" db:"daily_expected_amount"`
	DailyVariancePct       float64    `json:"daily_variance_pct" db:"daily_variance_pct"`
	Rolling7DayAvg         float64    `json:"rolling_7day_avg" db:"rolling_7day_avg"`
	Rolling30DayAvg        float64    `json:"rolling_30day_avg" db:"rolling_30day_avg"`
	CumulativeSpend        float64    `json:"cumulative_spend" db:"cumulative_spend"`
	CumulativeExpected     float64    `json:"cumulative_expected" db:"cumulative_expected"`
	CumulativeVariancePct  float64    `json:"cumulative_variance_pct" db:"cumulative_variance_pct"`
	ProjectedEndDate       *time.Time `json:"projected_end_date,omitempty" db:"projected_end_date"`
	ProjectedDepletionDate *time.Time `json:"projected_depletion_date,omitempty" db:"projected_depletion_date"`
	BudgetHealthScore      float64    `json:"budget_health_score" db:"budget_health_score"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
}

// BudgetAlert represents automated budget alerts
type BudgetAlert struct {
	ID             int64      `json:"id" db:"id"`
	AccountID      int64      `json:"account_id" db:"account_id"`
	GrantID        *int64     `json:"grant_id,omitempty" db:"grant_id"`
	AlertType      string     `json:"alert_type" db:"alert_type"`
	Severity       string     `json:"severity" db:"severity"`
	ThresholdValue float64    `json:"threshold_value" db:"threshold_value"`
	ActualValue    float64    `json:"actual_value" db:"actual_value"`
	Message        string     `json:"message" db:"message"`
	Details        string     `json:"details,omitempty" db:"details"`
	TriggeredAt    time.Time  `json:"triggered_at" db:"triggered_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy string     `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	Status         string     `json:"status" db:"status"`
}

// HealthCheckResponse represents service health status
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Uptime    string            `json:"uptime"`
}

// MetricsResponse represents Prometheus metrics endpoint response
type MetricsResponse struct {
	Metrics []MetricFamily `json:"metrics"`
}

// MetricFamily represents a Prometheus metric family
type MetricFamily struct {
	Name    string   `json:"name"`
	Help    string   `json:"help"`
	Type    string   `json:"type"`
	Metrics []Metric `json:"metrics"`
}

// Metric represents a single metric value
type Metric struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

// Validation helpers

// Validate performs basic validation on CreateAccountRequest
func (car *CreateAccountRequest) Validate() error {
	if car.SlurmAccount == "" {
		return NewValidationError("slurm_account", "is required")
	}
	if car.Name == "" {
		return NewValidationError("name", "is required")
	}
	if car.BudgetLimit <= 0 {
		return NewValidationError("budget_limit", "must be greater than 0")
	}
	if car.EndDate.Before(car.StartDate) {
		return NewValidationError("end_date", "must be after start_date")
	}
	return nil
}

// Validate performs basic validation on BudgetCheckRequest
func (bcr *BudgetCheckRequest) Validate() error {
	if bcr.Account == "" {
		return NewValidationError("account", "is required")
	}
	if bcr.Partition == "" {
		return NewValidationError("partition", "is required")
	}
	if bcr.Nodes < 1 {
		return NewValidationError("nodes", "must be at least 1")
	}
	if bcr.CPUs < 1 {
		return NewValidationError("cpus", "must be at least 1")
	}
	if bcr.WallTime == "" {
		return NewValidationError("wall_time", "is required")
	}
	return nil
}

// String returns a string representation of the account
func (ba *BudgetAccount) String() string {
	return fmt.Sprintf("BudgetAccount{Account: %s, Name: %s, Limit: %.2f, Used: %.2f, Available: %.2f}",
		ba.SlurmAccount, ba.Name, ba.BudgetLimit, ba.BudgetUsed, ba.BudgetAvailable())
}

// String returns a string representation of the transaction
func (bt *BudgetTransaction) String() string {
	return fmt.Sprintf("BudgetTransaction{ID: %s, Type: %s, Amount: %.2f, Status: %s}",
		bt.TransactionID, bt.Type, bt.Amount, bt.Status)
}
