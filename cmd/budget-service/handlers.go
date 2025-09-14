// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
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
		w.Write([]byte("# TODO: Implement metrics collection\n"))
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

// generateRequestID generates a simple request ID
func generateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
