// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout is the per-request timeout applied when no custom HTTP client
// is supplied. Admission sits in the Slurm resume path, so it must fail fast
// rather than stall node power-on.
const DefaultTimeout = 10 * time.Second

// Client is an HTTP client for the budget service API. It is the single source
// of truth for the wire contract: importers get the request/response types in
// this package for free instead of re-declaring them (see issue #5).
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client (custom timeout, transport,
// or instrumentation).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithTimeout sets the per-request timeout on the default HTTP client. It has no
// effect if WithHTTPClient supplied a client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if c.httpClient != nil {
			c.httpClient.Timeout = d
		}
	}
}

// WithAPIKey sets a bearer token sent on every request.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// NewClient creates a new API client for the budget service at baseURL (e.g.
// "http://budget.internal:8080"). A trailing slash is trimmed.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// doJSON performs an HTTP request with an optional JSON body and decodes a JSON
// response into out (which may be nil to discard the body). A non-2xx response
// is decoded as an ErrorResponse and returned as a *BudgetError, preserving the
// server's error code so callers can branch on it (e.g. ErrCodeInsufficientBudget).
func (c *Client) doJSON(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return NewBudgetErrorWithCause(ErrCodeInternal, "marshal request", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return NewBudgetErrorWithCause(ErrCodeInternal, "build request", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NewBudgetErrorWithCause(ErrCodeServiceUnavailable, fmt.Sprintf("%s %s", method, path), err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return NewBudgetErrorWithCause(ErrCodeExternalService, "decode response", err)
	}
	return nil
}

// decodeError converts a non-2xx response into a *BudgetError. It prefers the
// service's structured ErrorResponse; if the body is not in that shape it falls
// back to a generic error carrying the status and raw body.
func decodeError(resp *http.Response) error {
	raw, _ := io.ReadAll(resp.Body)
	var er ErrorResponse
	if json.Unmarshal(raw, &er) == nil && er.Error.Code != "" {
		return &BudgetError{
			Code:    er.Error.Code,
			Message: er.Error.Message,
			Details: er.Error.Details,
			Field:   er.Error.Field,
		}
	}
	return NewBudgetError(ErrCodeExternalService,
		fmt.Sprintf("budget service returned %s", resp.Status),
		strings.TrimSpace(string(raw)))
}

// --- Budget loop (the contract queuezero consumes) ---

// CheckBudget checks job-shaped budget availability and places a hold.
func (c *Client) CheckBudget(ctx context.Context, req *BudgetCheckRequest) (*BudgetCheckResponse, error) {
	var resp BudgetCheckResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/budget/check", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AdmitFleet performs fleet/resume-shaped spend-rate admission (issue #6). The
// returned EstimatedCost/HoldAmount are per-hour figures.
func (c *Client) AdmitFleet(ctx context.Context, req *FleetAdmissionRequest) (*BudgetCheckResponse, error) {
	var resp BudgetCheckResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/budget/admit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReconcileJob closes a hold against actual cost, charging actual and refunding
// the variance. It is keyed on req.TransactionID.
func (c *Client) ReconcileJob(ctx context.Context, req *JobReconcileRequest) (*JobReconcileResponse, error) {
	var resp JobReconcileResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/budget/reconcile", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Account management ---

// ListAccounts lists budget accounts.
func (c *Client) ListAccounts(ctx context.Context, _ *ListAccountsRequest) ([]*BudgetAccount, error) {
	var resp []*BudgetAccount
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/accounts", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateAccount creates a budget account.
func (c *Client) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*BudgetAccount, error) {
	var resp BudgetAccount
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/accounts", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAccount retrieves a budget account.
func (c *Client) GetAccount(ctx context.Context, account string) (*BudgetAccount, error) {
	var resp BudgetAccount
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/accounts/"+url.PathEscape(account), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Allocation management ---

// ListAllocationSchedules lists allocation schedules.
func (c *Client) ListAllocationSchedules(ctx context.Context, _ *AllocationScheduleRequest) ([]*BudgetAllocationSchedule, error) {
	var resp []*BudgetAllocationSchedule
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/allocations", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ProcessAllocations processes pending allocations.
func (c *Client) ProcessAllocations(ctx context.Context, req *ProcessAllocationsRequest) (*ProcessAllocationsResponse, error) {
	var resp ProcessAllocationsResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/allocations/process", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Grant management ---

// CreateGrant creates a new grant account.
func (c *Client) CreateGrant(ctx context.Context, req *CreateGrantRequest) (*GrantAccount, error) {
	var resp GrantAccount
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/grants", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetGrant retrieves a grant by number.
func (c *Client) GetGrant(ctx context.Context, grantNumber string) (*GrantAccount, error) {
	var resp GrantAccount
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/grants/"+url.PathEscape(grantNumber), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListGrants lists grants with filtering.
func (c *Client) ListGrants(ctx context.Context, _ *GrantListRequest) ([]*GrantAccount, error) {
	var resp []*GrantAccount
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/grants", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetBurnRateAnalysis retrieves burn rate analysis for an account.
func (c *Client) GetBurnRateAnalysis(ctx context.Context, req *BurnRateAnalysisRequest) (*BurnRateAnalysisResponse, error) {
	var resp BurnRateAnalysisResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/burn-rate/"+url.PathEscape(req.Account), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
