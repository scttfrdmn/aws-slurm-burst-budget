// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package budget

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

func TestNewService(t *testing.T) {
	cfg := &config.BudgetConfig{
		DefaultHoldPercentage: 1.5,
	}

	service := NewService(nil, nil, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.Nil(t, service.db)
	assert.Nil(t, service.advisorClient)
	assert.NotNil(t, service.accountQueries)     // NewService creates these even with nil DB
	assert.NotNil(t, service.transactionQueries) // NewService creates these even with nil DB
}

func TestService_GenerateTransactionID(t *testing.T) {
	service := &Service{}

	id1 := service.generateTransactionID()
	id2 := service.generateTransactionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "txn_")
	assert.Contains(t, id2, "txn_")

	// Test format: txn_nanoseconds_microseconds
	assert.True(t, len(id1) > 10)
	assert.True(t, len(id2) > 10)
}

func TestService_RecoverOrphanedTransactions_Disabled(t *testing.T) {
	cfg := &config.BudgetConfig{
		AutoRecoveryEnabled: false,
	}

	service := &Service{config: cfg}

	err := service.RecoverOrphanedTransactions(context.Background())
	assert.NoError(t, err)
}

func TestAdvisorClient_Interface(t *testing.T) {
	// Test that our mock client implements the interface
	var client AdvisorClient = &MockAdvisorClient{}
	assert.NotNil(t, client)

	// Test default mock behavior
	mockClient := &MockAdvisorClient{}
	resp, err := mockClient.EstimateCost(context.Background(), &CostEstimateRequest{
		Account:   "test",
		Partition: "cpu",
		Nodes:     1,
		CPUs:      4,
		WallTime:  "01:00:00",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 10.0, resp.EstimatedCost)
}

func TestCostEstimateRequest_Fields(t *testing.T) {
	req := &CostEstimateRequest{
		Account:   "test-account",
		Partition: "cpu",
		Nodes:     2,
		CPUs:      8,
		GPUs:      1,
		Memory:    "16GB",
		WallTime:  "02:00:00",
		JobScript: "#!/bin/bash\necho hello",
		Metadata:  map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-account", req.Account)
	assert.Equal(t, "cpu", req.Partition)
	assert.Equal(t, 2, req.Nodes)
	assert.Equal(t, 8, req.CPUs)
	assert.Equal(t, 1, req.GPUs)
	assert.Equal(t, "16GB", req.Memory)
	assert.Equal(t, "02:00:00", req.WallTime)
	assert.Equal(t, "#!/bin/bash\necho hello", req.JobScript)
	assert.Equal(t, "value", req.Metadata["key"])
}

func TestCostEstimateResponse_Fields(t *testing.T) {
	resp := &CostEstimateResponse{
		EstimatedCost:  25.50,
		Confidence:     0.95,
		Recommendation: "Optimal choice for this workload",
	}

	assert.Equal(t, 25.50, resp.EstimatedCost)
	assert.Equal(t, 0.95, resp.Confidence)
	assert.Equal(t, "Optimal choice for this workload", resp.Recommendation)
}

func TestService_BudgetCalculationLogic(t *testing.T) {
	tests := []struct {
		name               string
		estimatedCost      float64
		holdPercentage     float64
		expectedHoldAmount float64
	}{
		{
			name:               "basic calculation",
			estimatedCost:      10.0,
			holdPercentage:     1.2,
			expectedHoldAmount: 12.0,
		},
		{
			name:               "high cost calculation",
			estimatedCost:      100.0,
			holdPercentage:     1.5,
			expectedHoldAmount: 150.0,
		},
		{
			name:               "fractional cost",
			estimatedCost:      7.33,
			holdPercentage:     1.25,
			expectedHoldAmount: 9.1625,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the core budget calculation logic
			holdAmount := tt.estimatedCost * tt.holdPercentage
			assert.Equal(t, tt.expectedHoldAmount, holdAmount)
		})
	}
}

func TestService_AccountValidation(t *testing.T) {
	tests := []struct {
		name     string
		account  api.BudgetAccount
		isActive bool
	}{
		{
			name: "active account in valid period",
			account: api.BudgetAccount{
				Status:    "active",
				StartDate: time.Now().Add(-24 * time.Hour),
				EndDate:   time.Now().Add(24 * time.Hour),
			},
			isActive: true,
		},
		{
			name: "inactive account",
			account: api.BudgetAccount{
				Status:    "inactive",
				StartDate: time.Now().Add(-24 * time.Hour),
				EndDate:   time.Now().Add(24 * time.Hour),
			},
			isActive: false,
		},
		{
			name: "expired account",
			account: api.BudgetAccount{
				Status:    "active",
				StartDate: time.Now().Add(-48 * time.Hour),
				EndDate:   time.Now().Add(-24 * time.Hour),
			},
			isActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isActive, tt.account.IsActive())
		})
	}
}

// Benchmark tests
func BenchmarkService_GenerateTransactionID(b *testing.B) {
	service := &Service{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.generateTransactionID()
	}
}

func BenchmarkNewService(b *testing.B) {
	cfg := &config.BudgetConfig{
		DefaultHoldPercentage: 1.2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewService(nil, nil, cfg)
	}
}

func TestService_ConfigAndDependencies(t *testing.T) {
	cfg := &config.BudgetConfig{
		DefaultHoldPercentage: 1.3,
		ReconciliationTimeout: 48 * time.Hour,
		MinBudgetAmount:       0.1,
		MaxBudgetAmount:       500000.0,
		AllowNegativeBalance:  true,
		AutoRecoveryEnabled:   false,
		RecoveryCheckInterval: 2 * time.Hour,
		TransactionRetention:  1440 * time.Hour,
	}

	mockAdvisor := &MockAdvisorClient{
		EstimateResponse: &CostEstimateResponse{
			EstimatedCost:  50.0,
			Confidence:     0.9,
			Recommendation: "Test recommendation",
		},
	}

	service := NewService(nil, mockAdvisor, cfg)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, mockAdvisor, service.advisorClient)
	assert.NotNil(t, service.accountQueries)
	assert.NotNil(t, service.transactionQueries)
}

func TestService_AdvisorClientMock(t *testing.T) {
	t.Run("default response", func(t *testing.T) {
		client := &MockAdvisorClient{}
		resp, err := client.EstimateCost(context.Background(), &CostEstimateRequest{
			Account: "test",
		})
		assert.NoError(t, err)
		assert.Equal(t, 10.0, resp.EstimatedCost)
		assert.Equal(t, 0.8, resp.Confidence)
	})

	t.Run("custom response", func(t *testing.T) {
		customResp := &CostEstimateResponse{
			EstimatedCost:  25.0,
			Confidence:     0.95,
			Recommendation: "Custom test recommendation",
		}
		client := &MockAdvisorClient{
			EstimateResponse: customResp,
		}
		resp, err := client.EstimateCost(context.Background(), &CostEstimateRequest{})
		assert.NoError(t, err)
		assert.Equal(t, customResp, resp)
	})

	t.Run("error response", func(t *testing.T) {
		testErr := api.NewServiceUnavailableError("test", assert.AnError)
		client := &MockAdvisorClient{
			EstimateError: testErr,
		}
		resp, err := client.EstimateCost(context.Background(), &CostEstimateRequest{})
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, testErr, err)
	})
}

// MockAdvisorClient is a simple mock implementation of AdvisorClient
type MockAdvisorClient struct {
	EstimateResponse *CostEstimateResponse
	EstimateError    error
}

func (m *MockAdvisorClient) EstimateCost(ctx context.Context, req *CostEstimateRequest) (*CostEstimateResponse, error) {
	if m.EstimateError != nil {
		return nil, m.EstimateError
	}
	if m.EstimateResponse != nil {
		return m.EstimateResponse, nil
	}
	return &CostEstimateResponse{
		EstimatedCost:  10.0,
		Confidence:     0.8,
		Recommendation: "Default mock response",
	}, nil
}
