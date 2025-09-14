// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBudgetError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      BudgetError
		expected string
	}{
		{
			name: "error with details",
			err: BudgetError{
				Code:    ErrCodeValidation,
				Message: "Invalid input",
				Details: "Field 'name' is required",
			},
			expected: "VALIDATION_ERROR: Invalid input (Field 'name' is required)",
		},
		{
			name: "error without details",
			err: BudgetError{
				Code:    ErrCodeNotFound,
				Message: "Account not found",
			},
			expected: "NOT_FOUND: Account not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestBudgetError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := BudgetError{
		Code:    ErrCodeInternal,
		Message: "Internal error",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
}

func TestBudgetError_HTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{"validation error", ErrCodeValidation, http.StatusBadRequest},
		{"not found", ErrCodeNotFound, http.StatusNotFound},
		{"unauthorized", ErrCodeUnauthorized, http.StatusUnauthorized},
		{"forbidden", ErrCodeForbidden, http.StatusForbidden},
		{"insufficient budget", ErrCodeInsufficientBudget, http.StatusPaymentRequired},
		{"account inactive", ErrCodeAccountInactive, http.StatusPaymentRequired},
		{"account expired", ErrCodeAccountExpired, http.StatusPaymentRequired},
		{"partition exceeded", ErrCodePartitionExceeded, http.StatusPaymentRequired},
		{"duplicate account", ErrCodeDuplicateAccount, http.StatusConflict},
		{"service unavailable", ErrCodeServiceUnavailable, http.StatusServiceUnavailable},
		{"advisor unavailable", ErrCodeAdvisorUnavailable, http.StatusServiceUnavailable},
		{"database error", ErrCodeDatabaseError, http.StatusInternalServerError},
		{"transaction failed", ErrCodeTransactionFailed, http.StatusInternalServerError},
		{"external service", ErrCodeExternalService, http.StatusInternalServerError},
		{"internal error", ErrCodeInternal, http.StatusInternalServerError},
		{"unknown error", "UNKNOWN_ERROR", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BudgetError{Code: tt.code}
			assert.Equal(t, tt.expected, err.HTTPStatus())
		})
	}
}

func TestNewBudgetError(t *testing.T) {
	err := NewBudgetError(ErrCodeValidation, "Test error", "Additional details")

	assert.Equal(t, ErrCodeValidation, err.Code)
	assert.Equal(t, "Test error", err.Message)
	assert.Equal(t, "Additional details", err.Details)
	assert.Nil(t, err.Cause)
}

func TestNewBudgetErrorWithCause(t *testing.T) {
	cause := errors.New("original error")
	err := NewBudgetErrorWithCause(ErrCodeInternal, "Test error", cause, "Additional details")

	assert.Equal(t, ErrCodeInternal, err.Code)
	assert.Equal(t, "Test error", err.Message)
	assert.Equal(t, "Additional details", err.Details)
	assert.Equal(t, cause, err.Cause)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("field_name", "Field is required")

	assert.Equal(t, ErrCodeValidation, err.Code)
	assert.Equal(t, "Field is required", err.Message)
	assert.Equal(t, "field_name", err.Field)
}

func TestNewInsufficientBudgetError(t *testing.T) {
	err := NewInsufficientBudgetError("proj001", 100.0, 50.0)

	assert.Equal(t, ErrCodeInsufficientBudget, err.Code)
	assert.Equal(t, "Insufficient budget for account 'proj001'", err.Message)
	assert.Equal(t, "Required: $100.00, Available: $50.00", err.Details)
}

func TestNewAccountNotFoundError(t *testing.T) {
	err := NewAccountNotFoundError("proj001")

	assert.Equal(t, ErrCodeNotFound, err.Code)
	assert.Equal(t, "Budget account 'proj001' not found", err.Message)
}

func TestNewAccountInactiveError(t *testing.T) {
	err := NewAccountInactiveError("proj001", "suspended")

	assert.Equal(t, ErrCodeAccountInactive, err.Code)
	assert.Equal(t, "Account 'proj001' is not active", err.Message)
	assert.Equal(t, "Current status: suspended", err.Details)
}

func TestNewPartitionLimitError(t *testing.T) {
	err := NewPartitionLimitError("proj001", "gpu", 100.0, 50.0)

	assert.Equal(t, ErrCodePartitionExceeded, err.Code)
	assert.Equal(t, "Partition limit exceeded for 'gpu' in account 'proj001'", err.Message)
	assert.Equal(t, "Required: $100.00, Available: $50.00", err.Details)
}

func TestNewServiceUnavailableError(t *testing.T) {
	cause := errors.New("connection failed")
	err := NewServiceUnavailableError("advisor", cause)

	assert.Equal(t, ErrCodeServiceUnavailable, err.Code)
	assert.Equal(t, "Service 'advisor' is currently unavailable", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestNewDatabaseError(t *testing.T) {
	cause := errors.New("connection timeout")
	err := NewDatabaseError("create account", cause)

	assert.Equal(t, ErrCodeDatabaseError, err.Code)
	assert.Equal(t, "Database error during create account", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestNewTransactionFailedError(t *testing.T) {
	cause := errors.New("insufficient funds")
	err := NewTransactionFailedError("txn_123", cause)

	assert.Equal(t, ErrCodeTransactionFailed, err.Code)
	assert.Equal(t, "Transaction txn_123 failed", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestIsBudgetError(t *testing.T) {
	budgetErr := &BudgetError{Code: ErrCodeValidation, Message: "Test"}
	genericErr := errors.New("generic error")

	assert.True(t, IsBudgetError(budgetErr))
	assert.False(t, IsBudgetError(genericErr))
}

func TestAsBudgetError(t *testing.T) {
	budgetErr := &BudgetError{Code: ErrCodeValidation, Message: "Test"}
	genericErr := errors.New("generic error")

	result, ok := AsBudgetError(budgetErr)
	assert.True(t, ok)
	assert.Equal(t, budgetErr, result)

	result, ok = AsBudgetError(genericErr)
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestWrapError(t *testing.T) {
	t.Run("wrap generic error", func(t *testing.T) {
		genericErr := errors.New("generic error")
		wrappedErr := WrapError(genericErr, ErrCodeInternal, "Wrapped error")

		assert.Equal(t, ErrCodeInternal, wrappedErr.Code)
		assert.Equal(t, "Wrapped error", wrappedErr.Message)
		assert.Equal(t, genericErr, wrappedErr.Cause)
	})

	t.Run("wrap budget error returns same", func(t *testing.T) {
		budgetErr := &BudgetError{Code: ErrCodeValidation, Message: "Original"}
		wrappedErr := WrapError(budgetErr, ErrCodeInternal, "Wrapped error")

		assert.Equal(t, budgetErr, wrappedErr)
	})
}
