// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared fixture literals, kept as constants to satisfy goconst.
const (
	testPartition = "cpu"
	testRegion    = "us-east-1"
	testInstance  = "c6i.xlarge"
	testAccount   = "acct"
)

func TestClient_AdmitFleet(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	var gotReq FleetAdmissionRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotAuth = r.URL.Path, r.Method, r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		writeTestJSON(w, http.StatusOK, BudgetCheckResponse{
			Available:       true,
			EstimatedCost:   3.40, // $/hr
			HoldAmount:      4.08,
			TransactionID:   "txn_abc",
			BudgetRemaining: 995.92,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithAPIKey("secret"))
	resp, err := c.AdmitFleet(context.Background(), &FleetAdmissionRequest{
		Account: testAccount, Partition: testPartition, Region: testRegion,
		InstanceType: testInstance, CapacityModel: "spot", Count: 4,
	})

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/budget/admit", gotPath)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "Bearer secret", gotAuth)
	assert.Equal(t, 4, gotReq.Count)
	assert.True(t, resp.Available)
	assert.Equal(t, "txn_abc", resp.TransactionID)
	assert.InDelta(t, 3.40, resp.EstimatedCost, 1e-9)
}

func TestClient_ReconcileJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/budget/reconcile", r.URL.Path)
		var req JobReconcileRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeTestJSON(w, http.StatusOK, JobReconcileResponse{
			Success:       true,
			OriginalHold:  4.08,
			ActualCharge:  req.ActualCost,
			RefundAmount:  4.08 - req.ActualCost,
			TransactionID: req.TransactionID,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	resp, err := c.ReconcileJob(context.Background(), &JobReconcileRequest{
		JobID: "job1", ActualCost: 3.0, TransactionID: "txn_abc",
	})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.InDelta(t, 1.08, resp.RefundAmount, 1e-9)
}

func TestClient_ErrorResponseDecoded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mimic the server's writeError shape.
		var er ErrorResponse
		er.Error.Code = ErrCodeInsufficientBudget
		er.Error.Message = "Insufficient budget"
		writeTestJSON(w, http.StatusPaymentRequired, er)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.AdmitFleet(context.Background(), &FleetAdmissionRequest{
		Account: testAccount, Partition: testPartition, Region: testRegion, InstanceType: testInstance, Count: 1,
	})

	require.Error(t, err)
	be, ok := AsBudgetError(err)
	require.True(t, ok, "want a *BudgetError, got %T", err)
	assert.Equal(t, ErrCodeInsufficientBudget, be.Code)
}

func TestClient_TransportError(t *testing.T) {
	// Point at a closed server to force a transport-level failure.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := NewClient(url)
	_, err := c.GetAccount(context.Background(), "acct")
	require.Error(t, err)
	be, ok := AsBudgetError(err)
	require.True(t, ok)
	assert.Equal(t, ErrCodeServiceUnavailable, be.Code)
}

func writeTestJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
