// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"fmt"
	"net/http"
)

// Error types for the budget system

// ErrorCode represents different error types in the system
type ErrorCode string

const (
	// General errors
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"

	// Budget specific errors
	ErrCodeInsufficientBudget ErrorCode = "INSUFFICIENT_BUDGET"
	ErrCodeAccountInactive    ErrorCode = "ACCOUNT_INACTIVE"
	ErrCodeAccountExpired     ErrorCode = "ACCOUNT_EXPIRED"
	ErrCodePartitionExceeded  ErrorCode = "PARTITION_LIMIT_EXCEEDED"
	ErrCodeTransactionFailed  ErrorCode = "TRANSACTION_FAILED"
	ErrCodeDuplicateAccount   ErrorCode = "DUPLICATE_ACCOUNT"

	// Service errors
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeAdvisorUnavailable ErrorCode = "ADVISOR_UNAVAILABLE"
	ErrCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrCodeExternalService    ErrorCode = "EXTERNAL_SERVICE_ERROR"
)

// BudgetError represents an error in the budget system
type BudgetError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Field   string    `json:"field,omitempty"`
	Cause   error     `json:"-"`
}

// Error implements the error interface
func (e *BudgetError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *BudgetError) Unwrap() error {
	return e.Cause
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *BudgetError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeInsufficientBudget, ErrCodeAccountInactive, ErrCodeAccountExpired, ErrCodePartitionExceeded:
		return http.StatusPaymentRequired
	case ErrCodeDuplicateAccount:
		return http.StatusConflict
	case ErrCodeServiceUnavailable, ErrCodeAdvisorUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeDatabaseError, ErrCodeTransactionFailed, ErrCodeExternalService:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
		Details string    `json:"details,omitempty"`
		Field   string    `json:"field,omitempty"`
	} `json:"error"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

// NewBudgetError creates a new BudgetError
func NewBudgetError(code ErrorCode, message string, details ...string) *BudgetError {
	err := &BudgetError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewBudgetErrorWithCause creates a new BudgetError with a cause
func NewBudgetErrorWithCause(code ErrorCode, message string, cause error, details ...string) *BudgetError {
	err := &BudgetError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// NewValidationError creates a validation error
func NewValidationError(field, message string) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeValidation,
		Message: message,
		Field:   field,
	}
}

// NewInsufficientBudgetError creates an insufficient budget error
func NewInsufficientBudgetError(account string, required, available float64) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeInsufficientBudget,
		Message: fmt.Sprintf("Insufficient budget for account '%s'", account),
		Details: fmt.Sprintf("Required: $%.2f, Available: $%.2f", required, available),
	}
}

// NewAccountNotFoundError creates an account not found error
func NewAccountNotFoundError(account string) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("Budget account '%s' not found", account),
	}
}

// NewAccountInactiveError creates an account inactive error
func NewAccountInactiveError(account string, status string) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeAccountInactive,
		Message: fmt.Sprintf("Account '%s' is not active", account),
		Details: fmt.Sprintf("Current status: %s", status),
	}
}

// NewPartitionLimitError creates a partition limit exceeded error
func NewPartitionLimitError(account, partition string, required, available float64) *BudgetError {
	return &BudgetError{
		Code:    ErrCodePartitionExceeded,
		Message: fmt.Sprintf("Partition limit exceeded for '%s' in account '%s'", partition, account),
		Details: fmt.Sprintf("Required: $%.2f, Available: $%.2f", required, available),
	}
}

// NewServiceUnavailableError creates a service unavailable error
func NewServiceUnavailableError(service string, cause error) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeServiceUnavailable,
		Message: fmt.Sprintf("Service '%s' is currently unavailable", service),
		Cause:   cause,
	}
}

// NewDatabaseError creates a database error
func NewDatabaseError(operation string, cause error) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeDatabaseError,
		Message: fmt.Sprintf("Database error during %s", operation),
		Cause:   cause,
	}
}

// NewTransactionFailedError creates a transaction failed error
func NewTransactionFailedError(transactionID string, cause error) *BudgetError {
	return &BudgetError{
		Code:    ErrCodeTransactionFailed,
		Message: fmt.Sprintf("Transaction %s failed", transactionID),
		Cause:   cause,
	}
}

// Common error instances
var (
	ErrInternalServer = NewBudgetError(ErrCodeInternal, "Internal server error")
	ErrUnauthorized   = NewBudgetError(ErrCodeUnauthorized, "Unauthorized access")
	ErrForbidden      = NewBudgetError(ErrCodeForbidden, "Access forbidden")
)

// IsBudgetError checks if an error is a BudgetError
func IsBudgetError(err error) bool {
	_, ok := err.(*BudgetError)
	return ok
}

// AsBudgetError attempts to convert an error to a BudgetError
func AsBudgetError(err error) (*BudgetError, bool) {
	budgetErr, ok := err.(*BudgetError)
	return budgetErr, ok
}

// WrapError wraps a generic error as a BudgetError
func WrapError(err error, code ErrorCode, message string) *BudgetError {
	if budgetErr, ok := err.(*BudgetError); ok {
		return budgetErr
	}
	return NewBudgetErrorWithCause(code, message, err)
}