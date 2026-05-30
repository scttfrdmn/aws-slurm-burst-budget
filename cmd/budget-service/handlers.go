// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/asbx"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

// Risk/urgency levels surfaced in ASBA decision-support responses.
const (
	riskLow    = "LOW"
	riskMedium = "MEDIUM"
	riskHigh   = "HIGH"
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

// handleFleetAdmission handles fleet/resume-shaped spend-rate admission (issue
// #6). The request has no job walltime; the verdict's estimated_cost/hold_amount
// are per-hour figures (see Service.AdmitFleet).
func handleFleetAdmission(service *budget.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.FleetAdmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		response, err := service.AdmitFleet(r.Context(), &req)
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
		req := parseTransactionListQuery(r.URL.Query())

		transactions, err := service.ListTransactions(r.Context(), req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, transactions)
	}
}

// parseTransactionListQuery builds a TransactionListRequest from URL query
// parameters, ignoring malformed numeric/date values (kept out of the handler
// to bound its cyclomatic complexity).
func parseTransactionListQuery(q url.Values) *api.TransactionListRequest {
	req := &api.TransactionListRequest{
		Account: q.Get("account"),
		JobID:   q.Get("job_id"),
		Type:    q.Get("type"),
		Status:  q.Get("status"),
	}

	if limit, err := strconv.Atoi(q.Get("limit")); err == nil && limit > 0 {
		req.Limit = limit
	}
	if offset, err := strconv.Atoi(q.Get("offset")); err == nil && offset >= 0 {
		req.Offset = offset
	}
	if startDate, err := time.Parse(time.RFC3339, q.Get("start_date")); err == nil {
		req.StartDate = &startDate
	}
	if endDate, err := time.Parse(time.RFC3339, q.Get("end_date")); err == nil {
		req.EndDate = &endDate
	}

	return req
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

// handleASBXReconciliation handles cost reconciliation from ASBX. It closes a
// resume-time budget hold against actual cost via the integration service, which
// keys on ASBXJobCostData.BudgetTransactionID and delegates to ReconcileJob.
func handleASBXReconciliation(integration *asbx.IntegrationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ASBXCostReconciliationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		response, err := integration.ProcessCostReconciliation(r.Context(), &req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleASBXEpilog handles epilog data from SLURM, triggering reconciliation when
// the job has reached a terminal state.
func handleASBXEpilog(integration *asbx.IntegrationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ASBXEpilogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, api.NewValidationError("body", "Invalid JSON format"))
			return
		}

		response, err := integration.ProcessEpilogData(r.Context(), &req)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// handleASBXStatus reports ASBX integration health.
func handleASBXStatus(integration *asbx.IntegrationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := integration.GetIntegrationStatus(r.Context())
		if err != nil {
			writeError(w, err)
			return
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
			RiskLevel:           riskMedium,
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
			BudgetRisk:          riskLow,
			DeadlineRisk:        riskMedium,
			OverallRisk:         riskLow,
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
					Severity:     riskHigh,
					BudgetImpact: "May require intensive compute for final experiments",
					Recommendations: []string{
						"Reserve budget for final experiments",
						"Consider AWS burst for large-scale validation",
					},
				},
			},
			CurrentUrgency:         riskMedium,
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
		urgency := riskMedium
		if req.JobDeadline != nil && req.JobDeadline.Before(time.Now().Add(48*time.Hour)) {
			urgency = riskHigh
		}

		response := &api.BurstDecisionResponse{
			RecommendedAction:  "AWS",
			Confidence:         0.87,
			UrgencyLevel:       urgency,
			BudgetImpact:       (req.EstimatedAWSCost / 5000.00) * 100,
			AffordabilityScore: 0.92,
			TimelinePressure:   0.45,
			DeadlineRisk:       riskMedium,
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
