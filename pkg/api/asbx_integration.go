// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"time"
)

// ASBX Integration Types for aws-slurm-burst performance data

// ASBXJobCostData represents cost data exported from ASBX v0.2.0
type ASBXJobCostData struct {
	// Job identification
	JobID      string `json:"job_id"`
	SlurmJobID string `json:"slurm_job_id"`
	Account    string `json:"account"`
	Partition  string `json:"partition"`
	UserID     string `json:"user_id"`

	// Job execution details
	SubmittedAt time.Time `json:"submitted_at"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	JobState    string    `json:"job_state"` // COMPLETED, FAILED, CANCELLED, TIMEOUT
	ExitCode    int       `json:"exit_code"`

	// Resource usage
	RequestedNodes  int    `json:"requested_nodes"`
	UsedNodes       int    `json:"used_nodes"`
	RequestedCPUs   int    `json:"requested_cpus"`
	UsedCPUs        int    `json:"used_cpus"`
	RequestedGPUs   int    `json:"requested_gpus,omitempty"`
	UsedGPUs        int    `json:"used_gpus,omitempty"`
	RequestedMemory string `json:"requested_memory,omitempty"`
	UsedMemory      string `json:"used_memory,omitempty"`
	WallTimeLimit   string `json:"wall_time_limit"`
	ActualWallTime  string `json:"actual_wall_time"`

	// Cost breakdown from ASBX
	EstimatedCost float64            `json:"estimated_cost"`
	ActualCost    float64            `json:"actual_cost"`
	LocalCost     float64            `json:"local_cost,omitempty"`
	AWSCost       float64            `json:"aws_cost,omitempty"`
	CostBreakdown map[string]float64 `json:"cost_breakdown,omitempty"`

	// Performance metrics
	CPUEfficiency    float64                `json:"cpu_efficiency,omitempty"`
	MemoryEfficiency float64                `json:"memory_efficiency,omitempty"`
	IOMetrics        map[string]interface{} `json:"io_metrics,omitempty"`
	NetworkMetrics   map[string]interface{} `json:"network_metrics,omitempty"`

	// ASBX-specific data
	BurstDecision    string   `json:"burst_decision"` // LOCAL, AWS, HYBRID
	BurstReason      string   `json:"burst_reason,omitempty"`
	InstanceTypes    []string `json:"instance_types,omitempty"`
	AvailabilityZone string   `json:"availability_zone,omitempty"`

	// Metadata and context
	JobMetadata        map[string]interface{} `json:"job_metadata,omitempty"`
	PerformanceProfile string                 `json:"performance_profile,omitempty"`
	ResearchDomain     string                 `json:"research_domain,omitempty"`

	// ASBB transaction tracking
	BudgetTransactionID  string `json:"budget_transaction_id,omitempty"`
	ReconciliationStatus string `json:"reconciliation_status,omitempty"`
}

// ASBXCostReconciliationRequest represents a request to reconcile ASBX cost data
type ASBXCostReconciliationRequest struct {
	JobCostData        ASBXJobCostData `json:"job_cost_data"`
	AutoReconcile      bool            `json:"auto_reconcile"`
	UpdateCostModel    bool            `json:"update_cost_model"`
	GenerateReport     bool            `json:"generate_report"`
	NotifyStakeholders bool            `json:"notify_stakeholders"`
}

// ASBXCostReconciliationResponse represents the response from ASBX cost reconciliation
type ASBXCostReconciliationResponse struct {
	Success             bool   `json:"success"`
	ReconciliationID    string `json:"reconciliation_id"`
	OriginalTransaction string `json:"original_transaction"`

	// Cost reconciliation details
	EstimatedCost   float64 `json:"estimated_cost"`
	ActualCost      float64 `json:"actual_cost"`
	CostVariance    float64 `json:"cost_variance"`
	CostVariancePct float64 `json:"cost_variance_pct"`

	// Budget impact
	RefundAmount     float64 `json:"refund_amount"`
	AdditionalCharge float64 `json:"additional_charge"`

	// Performance learning
	EstimationAccuracy float64 `json:"estimation_accuracy"`
	ModelUpdateApplied bool    `json:"model_update_applied"`

	// Reporting
	ComplianceReportGenerated bool   `json:"compliance_report_generated,omitempty"`
	ReportPath                string `json:"report_path,omitempty"`

	Message         string   `json:"message"`
	Warnings        []string `json:"warnings,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// ASBXPerformanceFeedback represents performance data to improve cost estimation
type ASBXPerformanceFeedback struct {
	JobID     string `json:"job_id"`
	Account   string `json:"account"`
	Partition string `json:"partition"`

	// Resource utilization efficiency
	CPUEfficiency    float64 `json:"cpu_efficiency"`
	MemoryEfficiency float64 `json:"memory_efficiency"`
	GPUEfficiency    float64 `json:"gpu_efficiency,omitempty"`

	// Performance characteristics
	ComputeIntensive bool `json:"compute_intensive"`
	MemoryIntensive  bool `json:"memory_intensive"`
	IOIntensive      bool `json:"io_intensive"`
	NetworkIntensive bool `json:"network_intensive"`

	// Cost model learning data
	ActualVsEstimatedRatio    float64  `json:"actual_vs_estimated_ratio"`
	PerformanceProfile        string   `json:"performance_profile"`
	OptimizationOpportunities []string `json:"optimization_opportunities,omitempty"`

	// Context for future estimates
	SimilarJobPatterns      map[string]interface{} `json:"similar_job_patterns,omitempty"`
	ResourceRecommendations map[string]string      `json:"resource_recommendations,omitempty"`
}

// ASBXIntegrationStatus represents the status of ASBX integration
type ASBXIntegrationStatus struct {
	ASBXVersion               string    `json:"asbx_version"`
	IntegrationEnabled        bool      `json:"integration_enabled"`
	LastDataImport            time.Time `json:"last_data_import"`
	TotalJobsReconciled       int64     `json:"total_jobs_reconciled"`
	SuccessfulReconciliations int64     `json:"successful_reconciliations"`
	FailedReconciliations     int64     `json:"failed_reconciliations"`
	AverageReconciliationTime string    `json:"average_reconciliation_time"`
	CostModelAccuracy         float64   `json:"cost_model_accuracy"`
	LastHealthCheck           time.Time `json:"last_health_check"`
	HealthStatus              string    `json:"health_status"`
}

// ASBXEpilogRequest represents data from SLURM epilog script
type ASBXEpilogRequest struct {
	JobID     string `json:"job_id"`
	Account   string `json:"account"`
	Partition string `json:"partition"`
	UserID    string `json:"user_id"`
	JobState  string `json:"job_state"`
	ExitCode  int    `json:"exit_code"`

	// Timing information
	SubmitTime int64 `json:"submit_time"` // Unix timestamp
	StartTime  int64 `json:"start_time"`  // Unix timestamp
	EndTime    int64 `json:"end_time"`    // Unix timestamp

	// Resource usage from SLURM accounting
	AllocatedNodes  int    `json:"allocated_nodes"`
	AllocatedCPUs   int    `json:"allocated_cpus"`
	AllocatedGPUs   int    `json:"allocated_gpus,omitempty"`
	RequestedMemory string `json:"requested_memory,omitempty"`
	MaxRSS          string `json:"max_rss,omitempty"`
	MaxVMSize       string `json:"max_vm_size,omitempty"`

	// ASBX performance data path
	ASBXDataPath  string `json:"asbx_data_path,omitempty"`
	ASBXJobReport string `json:"asbx_job_report,omitempty"`

	// Additional SLURM accounting data
	SlurmAccountingData map[string]string `json:"slurm_accounting_data,omitempty"`
}

// ASBXEpilogResponse represents response from epilog processing
type ASBXEpilogResponse struct {
	Success                 bool     `json:"success"`
	JobID                   string   `json:"job_id"`
	ReconciliationTriggered bool     `json:"reconciliation_triggered"`
	ReconciliationID        string   `json:"reconciliation_id,omitempty"`
	Message                 string   `json:"message"`
	NextSteps               []string `json:"next_steps,omitempty"`
	DataImportStatus        string   `json:"data_import_status"`
	ErrorDetails            string   `json:"error_details,omitempty"`
}
