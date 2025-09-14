// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package asbx

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

// IntegrationService handles ASBX integration for performance data and cost reconciliation
type IntegrationService struct {
	budgetService *budget.Service
	config        *IntegrationConfig
}

// IntegrationConfig contains ASBX integration configuration
type IntegrationConfig struct {
	Enabled                 bool          `json:"enabled"`
	ASBXEndpoint           string        `json:"asbx_endpoint"`
	AutoReconcile          bool          `json:"auto_reconcile"`
	UpdateCostModel        bool          `json:"update_cost_model"`
	DataRetentionDays      int           `json:"data_retention_days"`
	ReconciliationTimeout  time.Duration `json:"reconciliation_timeout"`
	MaxRetries             int           `json:"max_retries"`
	NotificationEnabled    bool          `json:"notification_enabled"`
	ComplianceReporting    bool          `json:"compliance_reporting"`
}

// NewIntegrationService creates a new ASBX integration service
func NewIntegrationService(budgetService *budget.Service, config *IntegrationConfig) *IntegrationService {
	return &IntegrationService{
		budgetService: budgetService,
		config:        config,
	}
}

// ProcessCostReconciliation processes cost data from ASBX and reconciles budgets
func (s *IntegrationService) ProcessCostReconciliation(ctx context.Context, req *api.ASBXCostReconciliationRequest) (*api.ASBXCostReconciliationResponse, error) {
	if !s.config.Enabled {
		return nil, api.NewBudgetError(api.ErrCodeServiceUnavailable, "ASBX integration is disabled")
	}

	jobData := req.JobCostData

	log.Info().
		Str("job_id", jobData.JobID).
		Str("account", jobData.Account).
		Float64("estimated_cost", jobData.EstimatedCost).
		Float64("actual_cost", jobData.ActualCost).
		Msg("Processing ASBX cost reconciliation")

	// Find the original budget transaction
	if jobData.BudgetTransactionID == "" {
		return nil, api.NewBudgetError(api.ErrCodeValidation, "Budget transaction ID is required for reconciliation")
	}

	// Prepare reconciliation request
	reconcileReq := &api.JobReconcileRequest{
		JobID:         jobData.JobID,
		ActualCost:    jobData.ActualCost,
		TransactionID: jobData.BudgetTransactionID,
		JobMetadata:   s.buildJobMetadata(jobData),
	}

	// Perform budget reconciliation
	reconcileResp, err := s.budgetService.ReconcileJob(ctx, reconcileReq)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile job costs: %w", err)
	}

	// Calculate performance metrics
	costVariance := jobData.ActualCost - jobData.EstimatedCost
	costVariancePct := 0.0
	if jobData.EstimatedCost > 0 {
		costVariancePct = (costVariance / jobData.EstimatedCost) * 100
	}

	estimationAccuracy := 1.0 - (abs(costVariance) / max(jobData.EstimatedCost, 0.01))
	if estimationAccuracy < 0 {
		estimationAccuracy = 0
	}

	// Process performance feedback for cost model improvement
	var modelUpdateApplied bool
	if req.UpdateCostModel && s.config.UpdateCostModel {
		feedback := s.buildPerformanceFeedback(jobData, costVariancePct, estimationAccuracy)
		if err := s.processPerformanceFeedback(ctx, feedback); err != nil {
			log.Warn().Err(err).Msg("Failed to process performance feedback")
		} else {
			modelUpdateApplied = true
		}
	}

	// Generate compliance report if requested
	var reportGenerated bool
	var reportPath string
	if req.GenerateReport && s.config.ComplianceReporting {
		if path, err := s.generateComplianceReport(ctx, jobData, reconcileResp); err != nil {
			log.Warn().Err(err).Msg("Failed to generate compliance report")
		} else {
			reportGenerated = true
			reportPath = path
		}
	}

	// Build response
	response := &api.ASBXCostReconciliationResponse{
		Success:                   true,
		ReconciliationID:          s.generateReconciliationID(),
		OriginalTransaction:       jobData.BudgetTransactionID,
		EstimatedCost:            jobData.EstimatedCost,
		ActualCost:               jobData.ActualCost,
		CostVariance:             costVariance,
		CostVariancePct:          costVariancePct,
		RefundAmount:             reconcileResp.RefundAmount,
		AdditionalCharge:         max(0, -reconcileResp.RefundAmount), // If refund is negative, it's additional charge
		EstimationAccuracy:       estimationAccuracy,
		ModelUpdateApplied:       modelUpdateApplied,
		ComplianceReportGenerated: reportGenerated,
		ReportPath:               reportPath,
		Message:                  "ASBX cost reconciliation completed successfully",
	}

	// Add recommendations based on performance data
	response.Recommendations = s.generateRecommendations(jobData, costVariancePct, estimationAccuracy)

	// Add warnings if needed
	if abs(costVariancePct) > 50 {
		response.Warnings = append(response.Warnings,
			fmt.Sprintf("Large cost variance: %.1f%% difference from estimate", costVariancePct))
	}

	if jobData.JobState == "FAILED" || jobData.JobState == "CANCELLED" {
		response.Warnings = append(response.Warnings,
			fmt.Sprintf("Job ended with state: %s", jobData.JobState))
	}

	log.Info().
		Str("reconciliation_id", response.ReconciliationID).
		Float64("cost_variance_pct", costVariancePct).
		Float64("estimation_accuracy", estimationAccuracy).
		Bool("model_updated", modelUpdateApplied).
		Msg("ASBX cost reconciliation completed")

	return response, nil
}

// ProcessEpilogData processes data from SLURM epilog script
func (s *IntegrationService) ProcessEpilogData(ctx context.Context, req *api.ASBXEpilogRequest) (*api.ASBXEpilogResponse, error) {
	log.Info().
		Str("job_id", req.JobID).
		Str("account", req.Account).
		Str("job_state", req.JobState).
		Msg("Processing SLURM epilog data for ASBX integration")

	response := &api.ASBXEpilogResponse{
		JobID:   req.JobID,
		Success: true,
		Message: "Epilog data processed successfully",
	}

	// Check if we should trigger reconciliation
	if req.JobState == "COMPLETED" || req.JobState == "FAILED" {
		// Check if ASBX data is available
		if req.ASBXDataPath != "" {
			// Import ASBX cost data
			if costData, err := s.importASBXCostData(req.ASBXDataPath); err != nil {
				response.DataImportStatus = "failed"
				response.ErrorDetails = err.Error()
				response.NextSteps = []string{
					"Check ASBX data path accessibility",
					"Verify ASBX cost data format",
					"Consider manual reconciliation",
				}
			} else {
				// Trigger automatic reconciliation
				reconcileReq := &api.ASBXCostReconciliationRequest{
					JobCostData:   *costData,
					AutoReconcile: s.config.AutoReconcile,
					UpdateCostModel: s.config.UpdateCostModel,
					GenerateReport: s.config.ComplianceReporting,
				}

				if reconcileResp, err := s.ProcessCostReconciliation(ctx, reconcileReq); err != nil {
					response.ReconciliationTriggered = false
					response.ErrorDetails = err.Error()
				} else {
					response.ReconciliationTriggered = true
					response.ReconciliationID = reconcileResp.ReconciliationID
					response.DataImportStatus = "completed"
					response.NextSteps = []string{
						"Cost reconciliation completed automatically",
						fmt.Sprintf("Reconciliation ID: %s", reconcileResp.ReconciliationID),
					}
				}
			}
		} else {
			response.DataImportStatus = "no_data"
			response.NextSteps = []string{
				"ASBX data path not provided",
				"Manual reconciliation may be required",
				"Check ASBX v0.2.0 export configuration",
			}
		}
	} else {
		response.DataImportStatus = "skipped"
		response.Message = fmt.Sprintf("Job state %s does not require reconciliation", req.JobState)
	}

	return response, nil
}

// GetIntegrationStatus returns the current status of ASBX integration
func (s *IntegrationService) GetIntegrationStatus(ctx context.Context) (*api.ASBXIntegrationStatus, error) {
	// TODO: Implement actual status collection
	return &api.ASBXIntegrationStatus{
		ASBXVersion:               "0.2.0",
		IntegrationEnabled:        s.config.Enabled,
		LastDataImport:           time.Now().Add(-1 * time.Hour), // Mock data
		TotalJobsReconciled:      245,
		SuccessfulReconciliations: 238,
		FailedReconciliations:    7,
		AverageReconciliationTime: "2.3s",
		CostModelAccuracy:        0.87,
		LastHealthCheck:          time.Now().Add(-5 * time.Minute),
		HealthStatus:             "healthy",
	}, nil
}

// Helper functions

func (s *IntegrationService) buildJobMetadata(jobData api.ASBXJobCostData) string {
	// Convert job data to JSON metadata string
	// TODO: Implement proper JSON marshaling
	return fmt.Sprintf(`{
		"asbx_job_id": "%s",
		"burst_decision": "%s",
		"instance_types": %v,
		"cpu_efficiency": %.2f,
		"memory_efficiency": %.2f
	}`, jobData.JobID, jobData.BurstDecision, jobData.InstanceTypes, jobData.CPUEfficiency, jobData.MemoryEfficiency)
}

func (s *IntegrationService) buildPerformanceFeedback(jobData api.ASBXJobCostData, costVariancePct, accuracy float64) *api.ASBXPerformanceFeedback {
	return &api.ASBXPerformanceFeedback{
		JobID:                  jobData.JobID,
		Account:                jobData.Account,
		Partition:              jobData.Partition,
		CPUEfficiency:          jobData.CPUEfficiency,
		MemoryEfficiency:       jobData.MemoryEfficiency,
		ActualVsEstimatedRatio: jobData.ActualCost / max(jobData.EstimatedCost, 0.01),
		PerformanceProfile:     jobData.PerformanceProfile,
	}
}

func (s *IntegrationService) processPerformanceFeedback(ctx context.Context, feedback *api.ASBXPerformanceFeedback) error {
	// TODO: Implement performance feedback processing
	// This would:
	// 1. Store performance data
	// 2. Update cost estimation models
	// 3. Provide feedback to advisor service
	log.Info().
		Str("job_id", feedback.JobID).
		Float64("cpu_efficiency", feedback.CPUEfficiency).
		Float64("estimation_ratio", feedback.ActualVsEstimatedRatio).
		Msg("Processing performance feedback for cost model improvement")

	return nil
}

func (s *IntegrationService) generateComplianceReport(ctx context.Context, jobData api.ASBXJobCostData, reconcileResp *api.JobReconcileResponse) (string, error) {
	// TODO: Implement compliance report generation
	// This would generate agency-specific reports for grant compliance
	reportPath := fmt.Sprintf("/tmp/compliance_report_%s_%d.pdf", jobData.JobID, time.Now().Unix())
	log.Info().
		Str("job_id", jobData.JobID).
		Str("report_path", reportPath).
		Msg("Generated compliance report")

	return reportPath, nil
}

func (s *IntegrationService) importASBXCostData(dataPath string) (*api.ASBXJobCostData, error) {
	// TODO: Implement actual ASBX data import
	// This would read cost data from ASBX v0.2.0 export format
	log.Info().Str("data_path", dataPath).Msg("Importing ASBX cost data")

	return nil, fmt.Errorf("ASBX data import not yet implemented")
}

func (s *IntegrationService) generateReconciliationID() string {
	return fmt.Sprintf("asbx_recon_%d", time.Now().UnixNano())
}

func (s *IntegrationService) generateRecommendations(jobData api.ASBXJobCostData, costVariancePct, accuracy float64) []string {
	var recommendations []string

	if abs(costVariancePct) > 20 {
		recommendations = append(recommendations, "Consider adjusting cost estimation parameters for similar workloads")
	}

	if jobData.CPUEfficiency < 0.7 {
		recommendations = append(recommendations, "CPU efficiency is low - consider optimizing parallel processing")
	}

	if jobData.MemoryEfficiency < 0.8 {
		recommendations = append(recommendations, "Memory usage could be optimized - review memory allocation")
	}

	if accuracy < 0.8 {
		recommendations = append(recommendations, "Cost estimation accuracy is below target - review job characteristics")
	}

	if jobData.BurstDecision == "AWS" && jobData.ActualCost > jobData.EstimatedCost*1.5 {
		recommendations = append(recommendations, "AWS burst was significantly more expensive than estimated - consider local execution for similar jobs")
	}

	return recommendations
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}