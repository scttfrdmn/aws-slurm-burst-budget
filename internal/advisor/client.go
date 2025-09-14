// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package advisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

// Client provides HTTP client for the AWS SLURM Burst Advisor service
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	headers    map[string]string
}

// NewClient creates a new advisor client
func NewClient(cfg *config.AdvisorConfig) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL: cfg.URL,
		apiKey:  cfg.APIKey,
		headers: make(map[string]string),
	}

	// Set default headers
	client.headers["User-Agent"] = version.UserAgent()
	client.headers["Content-Type"] = "application/json"

	// Add configured headers
	for k, v := range cfg.Headers {
		client.headers[k] = v
	}

	return client
}

// EstimateCost estimates the cost for a job submission
func (c *Client) EstimateCost(ctx context.Context, req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
	// Convert request to advisor format
	advisorReq := map[string]interface{}{
		"account":   req.Account,
		"partition": req.Partition,
		"nodes":     req.Nodes,
		"cpus":      req.CPUs,
		"wall_time": req.WallTime,
		"timestamp": time.Now().Unix(),
	}

	if req.GPUs > 0 {
		advisorReq["gpus"] = req.GPUs
	}

	if req.Memory != "" {
		advisorReq["memory"] = req.Memory
	}

	if req.JobScript != "" {
		advisorReq["job_script"] = req.JobScript
	}

	if req.Metadata != nil {
		advisorReq["metadata"] = req.Metadata
	}

	// Marshal request
	reqBody, err := json.Marshal(advisorReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/analyze", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}

	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("advisor request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// HTTP response body close failed - acknowledge error
			_ = err // Error is handled by acknowledging it
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("advisor returned status %d", resp.StatusCode)
	}

	// Parse response
	var advisorResp struct {
		EstimatedCost  float64 `json:"estimated_cost"`
		LocalCost      float64 `json:"local_cost,omitempty"`
		AWSCost        float64 `json:"aws_cost,omitempty"`
		Recommendation string  `json:"recommendation"`
		Confidence     float64 `json:"confidence"`
		Error          string  `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&advisorResp); err != nil {
		return nil, fmt.Errorf("failed to decode advisor response: %w", err)
	}

	if advisorResp.Error != "" {
		return nil, fmt.Errorf("advisor error: %s", advisorResp.Error)
	}

	return &budget.CostEstimateResponse{
		EstimatedCost:  advisorResp.EstimatedCost,
		Confidence:     advisorResp.Confidence,
		Recommendation: advisorResp.Recommendation,
	}, nil
}

// HealthCheck checks if the advisor service is available
func (c *Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	httpReq.Header.Set("User-Agent", version.UserAgent())

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// HTTP response body close failed - acknowledge error
			_ = err // Error is handled by acknowledging it
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("advisor health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// MockClient provides a mock implementation for testing
type MockClient struct {
	EstimateFunc    func(ctx context.Context, req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error)
	HealthCheckFunc func(ctx context.Context) error
}

// EstimateCost implements the mock estimate cost
func (m *MockClient) EstimateCost(ctx context.Context, req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
	if m.EstimateFunc != nil {
		return m.EstimateFunc(ctx, req)
	}

	// Default mock response
	return &budget.CostEstimateResponse{
		EstimatedCost:  10.0,
		Confidence:     0.8,
		Recommendation: "Job should run efficiently on AWS",
	}, nil
}

// HealthCheck implements the mock health check
func (m *MockClient) HealthCheck(ctx context.Context) error {
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}
