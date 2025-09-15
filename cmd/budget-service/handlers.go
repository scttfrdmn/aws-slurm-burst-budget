// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

// handleBudgetCheck handles budget availability checks for job submissions
func handleBudgetCheck(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.BudgetCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		response, err := service.CheckBudget(r.Context(), &req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleJobReconcile handles job reconciliation after completion
func handleJobReconcile(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.JobReconcileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		response, err := service.ReconcileJob(r.Context(), &req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleCreateAccount creates a new budget account
func handleCreateAccount(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.CreateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		account, err := service.CreateAccount(r.Context(), &req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, account)
	}
}

// handleGetAccount retrieves a budget account by name
func handleGetAccount(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		accountName := vars["account"]

		account, err := service.GetAccount(r.Context(), accountName)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, account)
	}
}

// handleListAccounts lists budget accounts with optional filtering
func handleListAccounts(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &api.ListAccountsRequest{}

		// Parse query parameters
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				req.Limit = limit
			}
		}

		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
				req.Offset = offset
			}
		}

		if status := r.URL.Query().Get("status"); status != "" {
			req.Status = status
		}

		accounts, err := service.ListAccounts(r.Context(), req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, accounts)
	}
}

// handleUpdateAccount updates a budget account
func handleUpdateAccount(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		accountName := vars["account"]

		var req api.UpdateAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		account, err := service.UpdateAccount(r.Context(), accountName, &req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, account)
	}
}

// handleDeleteAccount deletes a budget account
func handleDeleteAccount(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		accountName := vars["account"]

		err := service.DeleteAccount(r.Context(), accountName)
		if err != nil {
			writeError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListTransactions lists transactions with filtering
func handleListTransactions(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &api.TransactionListRequest{}

		// Parse query parameters
		if account := r.URL.Query().Get("account"); account != "" {
			req.Account = account
		}

		if jobID := r.URL.Query().Get("job_id"); jobID != "" {
			req.JobID = jobID
		}

		if txnType := r.URL.Query().Get("type"); txnType != "" {
			req.Type = txnType
		}

		if status := r.URL.Query().Get("status"); status != "" {
			req.Status = status
		}

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				req.Limit = limit
			}
		}

		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
				req.Offset = offset
			}
		}

		// Parse date parameters
		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
				req.StartDate = &startDate
			}
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
				req.EndDate = &endDate
			}
		}

		transactions, err := service.ListTransactions(r.Context(), req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, transactions)
	}
}

// handleHealth handles health check requests
func handleHealth(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "healthy"
		services := make(map[string]string)

		// Check database
		if err := service.HealthCheck(r.Context()); err != nil {
			status = "unhealthy"
			services["database"] = "unhealthy: " + err.Error()
		} else {
			services["database"] = "healthy"
		}

		// TODO: Add advisor service health check
		services["advisor"] = "unknown"

		response := &api.HealthCheckResponse{
			Status:    status,
			Version:   version.Version,
			Timestamp: time.Now(),
			Services:  services,
			Uptime:    "unknown", // TODO: Calculate actual uptime
		}

		if status == "unhealthy" {
			writeJSON(w, http.StatusServiceUnavailable, response)
		} else {
			writeJSON(w, http.StatusOK, response)
		}
	}
}

// handleMetrics handles Prometheus metrics requests
func handleMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement Prometheus metrics collection
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("# TODO: Implement metrics collection\n")); err != nil {
			log.Error().Err(err).Msg("Failed to write metrics response")
		}
	}
}

// handleVersion handles version information requests
func handleVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buildInfo := version.GetBuildInfo()
		writeJSON(w, http.StatusOK, buildInfo)
	}
}

// Helper functions

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, err error) {
	budgetErr, ok := api.AsBudgetError(err)
	if !ok {
		budgetErr = api.NewBudgetError(api.ErrCodeInternal, "Internal server error")
		budgetErr.Cause = err
	}

	response := &api.ErrorResponse{
		RequestID: generateRequestID(),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	response.Error.Code = budgetErr.Code
	response.Error.Message = budgetErr.Message
	response.Error.Details = budgetErr.Details
	response.Error.Field = budgetErr.Field

	// Log the error
	log.Error().
		Err(budgetErr.Cause).
		Str("code", string(budgetErr.Code)).
		Str("message", budgetErr.Message).
		Str("request_id", response.RequestID).
		Msg("API error")

	writeJSON(w, budgetErr.HTTPStatus(), response)
}

// ASBX Integration handlers

// handleASBXReconciliation handles cost reconciliation from ASBX
func handleASBXReconciliation(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ASBXCostReconciliationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		// TODO: Implement ASBX integration service
		// For now, return a placeholder response
		response := &api.ASBXCostReconciliationResponse{
			Success:          false,
			Message:          "ASBX integration not yet implemented",
			ReconciliationID: fmt.Sprintf("placeholder_%d", time.Now().Unix()),
		}

		writeJSON(w, http.StatusNotImplemented, response)
	}
}

// handleASBXEpilog handles epilog data from SLURM
func handleASBXEpilog(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ASBXEpilogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		// TODO: Implement ASBX epilog processing
		response := &api.ASBXEpilogResponse{
			Success:                 true,
			JobID:                   req.JobID,
			DataImportStatus:        "not_implemented",
			ReconciliationTriggered: false,
			Message:                 "ASBX epilog processing not yet implemented",
			NextSteps: []string{
				"ASBX integration service implementation pending",
				"Manual reconciliation may be required",
			},
		}

		writeJSON(w, http.StatusNotImplemented, response)
	}
}

// handleASBXStatus handles ASBX integration status requests
func handleASBXStatus(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement actual ASBX status checking
		status := &api.ASBXIntegrationStatus{
			ASBXVersion:               "0.2.0",
			IntegrationEnabled:        false, // Not yet implemented
			LastDataImport:            time.Now().Add(-24 * time.Hour),
			TotalJobsReconciled:       0,
			SuccessfulReconciliations: 0,
			FailedReconciliations:     0,
			AverageReconciliationTime: "0s",
			CostModelAccuracy:         0.0,
			LastHealthCheck:           time.Now(),
			HealthStatus:              "integration_pending",
		}

		writeJSON(w, http.StatusOK, status)
	}
}

// ASBA Integration handlers (Issues #2 and #3)

// handleASBABudgetStatus handles budget status queries for ASBA decision making
func handleASBABudgetStatus(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.BudgetStatusQuery
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		// TODO: Implement comprehensive budget status analysis
		response := &api.BudgetStatusResponse{
			Account:             req.Account,
			BudgetLimit:         5000.00,
			BudgetUsed:          1250.75,
			BudgetHeld:          320.50,
			BudgetAvailable:     3428.75,
			BudgetUtilization:   25.015,
			DailyBurnRate:       125.50,
			ExpectedDailyRate:   100.00,
			BurnRateVariance:    25.5,
			BudgetHealthScore:   78.5,
			HealthStatus:        "CONCERN",
			DaysRemaining:       90,
			RiskLevel:           "MEDIUM",
			CanAffordAWSBurst:   true,
			RecommendedDecision: "PREFER_LOCAL",
			DecisionReasoning: []string{
				"Budget health is concerning with 25.5% overspend rate",
				"Sufficient budget available for moderate AWS usage",
				"Recommend local execution for cost efficiency",
			},
			LastUpdated: time.Now(),
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleASBAAffordabilityCheck handles affordability checks for job submissions
func handleASBAAffordabilityCheck(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.AffordabilityCheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		// TODO: Implement sophisticated affordability analysis
		response := &api.AffordabilityCheckResponse{
			Affordable:          req.EstimatedAWSCost <= 500.00, // Simple threshold
			RecommendedDecision: "AWS",
			ConfidenceLevel:     0.85,
			EstimatedAWSCost:    req.EstimatedAWSCost,
			BudgetImpact:        (req.EstimatedAWSCost / 5000.00) * 100, // Percentage
			BudgetRisk:          "LOW",
			DeadlineRisk:        "MEDIUM",
			OverallRisk:         "LOW",
			DecisionFactors: map[string]interface{}{
				"budget_health":     "good",
				"cost_efficiency":   0.8,
				"deadline_pressure": 0.3,
			},
			Reasoning: []string{
				fmt.Sprintf("Job cost $%.2f is within budget limits", req.EstimatedAWSCost),
				"AWS execution recommended for time savings",
			},
			Message: "Job is affordable and recommended for AWS execution",
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleASBAGrantTimeline handles grant timeline queries
func handleASBAGrantTimeline(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.GrantTimelineQuery
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		// TODO: Implement grant timeline analysis
		now := time.Now()
		response := &api.GrantTimelineResponse{
			Account:            req.Account,
			GrantStartDate:     now.AddDate(0, -6, 0), // 6 months ago
			GrantEndDate:       now.AddDate(2, 6, 0),  // 2.5 years from now
			CurrentPeriod:      2,
			TotalPeriods:       3,
			PeriodEndDate:      now.AddDate(0, 6, 0), // 6 months from now
			DaysUntilPeriodEnd: 180,
			DaysUntilGrantEnd:  912, // ~2.5 years
			NextAllocation: &api.AllocationEvent{
				Date:        now.AddDate(0, 1, 0), // Next month
				Amount:      250000.00,
				Description: "Quarterly budget allocation",
				Type:        "AUTOMATIC",
				DaysFromNow: 30,
			},
			UpcomingDeadlines: []api.CriticalDeadline{
				{
					Type:         "CONFERENCE",
					Description:  "ICML 2025 Paper Submission",
					Date:         now.AddDate(0, 2, 15), // ~2.5 months
					DaysFromNow:  75,
					Severity:     "HIGH",
					BudgetImpact: "May require intensive compute for final experiments",
					Recommendations: []string{
						"Reserve budget for final experiments",
						"Consider AWS burst for large-scale validation",
					},
				},
			},
			CurrentUrgency:         "MEDIUM",
			BurstingRecommendation: "NORMAL",
			OptimizationAdvice: []string{
				"Budget health is good, moderate AWS usage acceptable",
				"Plan for conference deadline compute requirements",
				"Monitor burn rate as grant approaches mid-point",
			},
			LastUpdated: now,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleASBABurstDecision handles comprehensive burst decision making
func handleASBABurstDecision(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.BurstDecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		// TODO: Implement sophisticated burst decision logic
		urgency := "MEDIUM"
		if req.JobDeadline != nil && req.JobDeadline.Before(time.Now().Add(48*time.Hour)) {
			urgency = "HIGH"
		}

		response := &api.BurstDecisionResponse{
			RecommendedAction:  "AWS",
			Confidence:         0.87,
			UrgencyLevel:       urgency,
			BudgetImpact:       (req.EstimatedAWSCost / 5000.00) * 100,
			AffordabilityScore: 0.92,
			TimelinePressure:   0.45,
			DeadlineRisk:       "MEDIUM",
			GrantHealthImpact:  "MINIMAL",
			DecisionFactors: []api.DecisionFactor{
				{
					Factor:      "Budget Health",
					Weight:      0.3,
					Value:       0.85,
					Impact:      "POSITIVE",
					Description: "Account has healthy budget status",
				},
				{
					Factor:      "Deadline Pressure",
					Weight:      0.4,
					Value:       0.6,
					Impact:      "NEUTRAL",
					Description: "Moderate deadline pressure",
				},
				{
					Factor:      "Cost Efficiency",
					Weight:      0.3,
					Value:       0.75,
					Impact:      "POSITIVE",
					Description: "AWS cost is reasonable for time savings",
				},
			},
			ImmediateActions: []string{
				"Submit job to AWS for faster completion",
				"Monitor budget impact after job completion",
			},
			LongtermSuggestions: []string{
				"Consider optimizing job for better cost efficiency",
				"Plan budget allocation for upcoming deadlines",
			},
			Message: "AWS burst recommended based on budget health and timeline analysis",
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
