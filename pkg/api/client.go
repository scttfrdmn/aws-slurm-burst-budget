// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"context"
	"fmt"
)

// Client provides HTTP client for the budget service API
type Client struct {
	baseURL    string
	httpClient interface{} // TODO: Use proper HTTP client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

// Placeholder implementations - TODO: Implement actual HTTP calls

// ListAccounts lists budget accounts
func (c *Client) ListAccounts(ctx context.Context, req *ListAccountsRequest) ([]*BudgetAccount, error) {
	return nil, fmt.Errorf("not implemented")
}

// CreateAccount creates a budget account
func (c *Client) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*BudgetAccount, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetAccount retrieves a budget account
func (c *Client) GetAccount(ctx context.Context, account string) (*BudgetAccount, error) {
	return nil, fmt.Errorf("not implemented")
}

// ListAllocationSchedules lists allocation schedules
func (c *Client) ListAllocationSchedules(ctx context.Context, req *AllocationScheduleRequest) ([]*BudgetAllocationSchedule, error) {
	return nil, fmt.Errorf("not implemented")
}

// ProcessAllocations processes pending allocations
func (c *Client) ProcessAllocations(ctx context.Context, req *ProcessAllocationsRequest) (*ProcessAllocationsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}
