// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package advisor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.AdvisorConfig{
		URL:     "http://localhost:8081",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
		Headers: map[string]string{
			"Custom-Header": "test-value",
		},
	}

	client := NewClient(cfg)

	assert.NotNil(t, client)
	assert.Equal(t, cfg.URL, client.baseURL)
	assert.Equal(t, cfg.APIKey, client.apiKey)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.headers)

	// Check that default headers are set
	assert.Contains(t, client.headers, "User-Agent")
	assert.Contains(t, client.headers, "Content-Type")
	assert.Equal(t, "application/json", client.headers["Content-Type"])

	// Check that custom headers are included
	assert.Equal(t, "test-value", client.headers["Custom-Header"])
}

func TestClient_EstimateCost_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/analyze", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{
			"estimated_cost": 15.50,
			"confidence": 0.85,
			"recommendation": "Good choice for this workload"
		}`)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := &config.AdvisorConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg)

	req := &budget.CostEstimateRequest{
		Account:   "test-account",
		Partition: "cpu",
		Nodes:     2,
		CPUs:      8,
		WallTime:  "02:00:00",
	}

	resp, err := client.EstimateCost(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, 15.50, resp.EstimatedCost)
	assert.Equal(t, 0.85, resp.Confidence)
	assert.Equal(t, "Good choice for this workload", resp.Recommendation)
}

func TestClient_EstimateCost_ServerError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.AdvisorConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg)

	req := &budget.CostEstimateRequest{
		Account:   "test-account",
		Partition: "cpu",
		Nodes:     1,
		CPUs:      4,
		WallTime:  "01:00:00",
	}

	resp, err := client.EstimateCost(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "advisor returned status 500")
}

func TestClient_EstimateCost_AdvisorError(t *testing.T) {
	// Create mock server that returns advisor error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{
			"error": "Invalid job configuration"
		}`)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := &config.AdvisorConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg)

	req := &budget.CostEstimateRequest{
		Account:   "test-account",
		Partition: "invalid-partition",
		Nodes:     1,
		CPUs:      4,
		WallTime:  "01:00:00",
	}

	resp, err := client.EstimateCost(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "advisor error: Invalid job configuration")
}

func TestClient_HealthCheck_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/health", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status": "healthy"}`)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	cfg := &config.AdvisorConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg)

	err := client.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestClient_HealthCheck_Failure(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := &config.AdvisorConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}

	client := NewClient(cfg)

	err := client.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "advisor health check failed with status 503")
}

func TestMockClient_EstimateCost(t *testing.T) {
	mock := &MockClient{}

	req := &budget.CostEstimateRequest{
		Account:   "test",
		Partition: "cpu",
		Nodes:     1,
		CPUs:      4,
		WallTime:  "01:00:00",
	}

	// Test default mock response
	resp, err := mock.EstimateCost(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 10.0, resp.EstimatedCost)
	assert.Equal(t, 0.8, resp.Confidence)
	assert.Equal(t, "Job should run efficiently on AWS", resp.Recommendation)
}

func TestMockClient_HealthCheck(t *testing.T) {
	mock := &MockClient{}

	err := mock.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestMockClient_CustomFunctions(t *testing.T) {
	mock := &MockClient{
		EstimateFunc: func(ctx context.Context, req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
			return &budget.CostEstimateResponse{
				EstimatedCost:  50.0,
				Confidence:     0.9,
				Recommendation: "Custom recommendation",
			}, nil
		},
		HealthCheckFunc: func(ctx context.Context) error {
			return assert.AnError
		},
	}

	// Test custom estimate function
	req := &budget.CostEstimateRequest{
		Account: "test",
	}
	resp, err := mock.EstimateCost(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 50.0, resp.EstimatedCost)
	assert.Equal(t, "Custom recommendation", resp.Recommendation)

	// Test custom health check function
	err = mock.HealthCheck(context.Background())
	assert.Error(t, err)
}
