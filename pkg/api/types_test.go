// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBudgetAccount_BudgetAvailable(t *testing.T) {
	tests := []struct {
		name     string
		account  BudgetAccount
		expected float64
	}{
		{
			name: "positive available budget",
			account: BudgetAccount{
				BudgetLimit: 1000.0,
				BudgetUsed:  300.0,
				BudgetHeld:  200.0,
			},
			expected: 500.0,
		},
		{
			name: "zero available budget",
			account: BudgetAccount{
				BudgetLimit: 1000.0,
				BudgetUsed:  600.0,
				BudgetHeld:  400.0,
			},
			expected: 0.0,
		},
		{
			name: "negative available budget",
			account: BudgetAccount{
				BudgetLimit: 1000.0,
				BudgetUsed:  700.0,
				BudgetHeld:  400.0,
			},
			expected: -100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.account.BudgetAvailable())
		})
	}
}

func TestBudgetAccount_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		account  BudgetAccount
		expected bool
	}{
		{
			name: "active account within date range",
			account: BudgetAccount{
				Status:    "active",
				StartDate: now.Add(-24 * time.Hour),
				EndDate:   now.Add(24 * time.Hour),
			},
			expected: true,
		},
		{
			name: "inactive account",
			account: BudgetAccount{
				Status:    "inactive",
				StartDate: now.Add(-24 * time.Hour),
				EndDate:   now.Add(24 * time.Hour),
			},
			expected: false,
		},
		{
			name: "account before start date",
			account: BudgetAccount{
				Status:    "active",
				StartDate: now.Add(24 * time.Hour),
				EndDate:   now.Add(48 * time.Hour),
			},
			expected: false,
		},
		{
			name: "account after end date",
			account: BudgetAccount{
				Status:    "active",
				StartDate: now.Add(-48 * time.Hour),
				EndDate:   now.Add(-24 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.account.IsActive())
		})
	}
}

func TestBudgetPartitionLimit_Available(t *testing.T) {
	limit := BudgetPartitionLimit{
		Limit: 500.0,
		Used:  200.0,
		Held:  100.0,
	}

	assert.Equal(t, 200.0, limit.Available())
}

func TestCreateAccountRequest_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		request CreateAccountRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: CreateAccountRequest{
				SlurmAccount: "proj001",
				Name:         "Test Project",
				Description:  "Test project description",
				BudgetLimit:  1000.0,
				StartDate:    now,
				EndDate:      now.Add(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "missing SLURM account",
			request: CreateAccountRequest{
				Name:        "Test Project",
				BudgetLimit: 1000.0,
				StartDate:   now,
				EndDate:     now.Add(24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "missing name",
			request: CreateAccountRequest{
				SlurmAccount: "proj001",
				BudgetLimit:  1000.0,
				StartDate:    now,
				EndDate:      now.Add(24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "zero budget limit",
			request: CreateAccountRequest{
				SlurmAccount: "proj001",
				Name:         "Test Project",
				BudgetLimit:  0.0,
				StartDate:    now,
				EndDate:      now.Add(24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "negative budget limit",
			request: CreateAccountRequest{
				SlurmAccount: "proj001",
				Name:         "Test Project",
				BudgetLimit:  -100.0,
				StartDate:    now,
				EndDate:      now.Add(24 * time.Hour),
			},
			wantErr: true,
		},
		{
			name: "end date before start date",
			request: CreateAccountRequest{
				SlurmAccount: "proj001",
				Name:         "Test Project",
				BudgetLimit:  1000.0,
				StartDate:    now.Add(24 * time.Hour),
				EndDate:      now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBudgetCheckRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request BudgetCheckRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: BudgetCheckRequest{
				Account:   "proj001",
				Partition: "cpu",
				Nodes:     1,
				CPUs:      4,
				WallTime:  "01:00:00",
			},
			wantErr: false,
		},
		{
			name: "missing account",
			request: BudgetCheckRequest{
				Partition: "cpu",
				Nodes:     1,
				CPUs:      4,
				WallTime:  "01:00:00",
			},
			wantErr: true,
		},
		{
			name: "missing partition",
			request: BudgetCheckRequest{
				Account:  "proj001",
				Nodes:    1,
				CPUs:     4,
				WallTime: "01:00:00",
			},
			wantErr: true,
		},
		{
			name: "zero nodes",
			request: BudgetCheckRequest{
				Account:   "proj001",
				Partition: "cpu",
				Nodes:     0,
				CPUs:      4,
				WallTime:  "01:00:00",
			},
			wantErr: true,
		},
		{
			name: "zero CPUs",
			request: BudgetCheckRequest{
				Account:   "proj001",
				Partition: "cpu",
				Nodes:     1,
				CPUs:      0,
				WallTime:  "01:00:00",
			},
			wantErr: true,
		},
		{
			name: "missing wall time",
			request: BudgetCheckRequest{
				Account:   "proj001",
				Partition: "cpu",
				Nodes:     1,
				CPUs:      4,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBudgetAccount_String(t *testing.T) {
	account := BudgetAccount{
		SlurmAccount: "proj001",
		Name:         "Test Project",
		BudgetLimit:  1000.0,
		BudgetUsed:   300.0,
		BudgetHeld:   200.0,
	}

	expected := "BudgetAccount{Account: proj001, Name: Test Project, Limit: 1000.00, Used: 300.00, Available: 500.00}"
	assert.Equal(t, expected, account.String())
}

func TestBudgetTransaction_String(t *testing.T) {
	transaction := BudgetTransaction{
		TransactionID: "txn_123",
		Type:          "hold",
		Amount:        100.0,
		Status:        "completed",
	}

	expected := "BudgetTransaction{ID: txn_123, Type: hold, Amount: 100.00, Status: completed}"
	assert.Equal(t, expected, transaction.String())
}

// Benchmark tests
func BenchmarkBudgetAccount_BudgetAvailable(b *testing.B) {
	account := BudgetAccount{
		BudgetLimit: 1000.0,
		BudgetUsed:  300.0,
		BudgetHeld:  200.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = account.BudgetAvailable()
	}
}

func BenchmarkBudgetAccount_IsActive(b *testing.B) {
	now := time.Now()
	account := BudgetAccount{
		Status:    "active",
		StartDate: now.Add(-24 * time.Hour),
		EndDate:   now.Add(24 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = account.IsActive()
	}
}
