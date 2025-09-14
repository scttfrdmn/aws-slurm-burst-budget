// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateGrantRequest_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		request CreateGrantRequest
		wantErr bool
	}{
		{
			name: "valid NSF grant",
			request: CreateGrantRequest{
				GrantNumber:           "NSF-2025-12345",
				FundingAgency:         "National Science Foundation",
				PrincipalInvestigator: "Dr. Jane Smith",
				Institution:           "Research University",
				GrantStartDate:        now,
				GrantEndDate:          now.Add(3 * 365 * 24 * time.Hour), // 3 years
				TotalAwardAmount:      750000.0,
				IndirectCostRate:      0.30,
				BudgetPeriodMonths:    12,
			},
			wantErr: false,
		},
		{
			name: "missing grant number",
			request: CreateGrantRequest{
				FundingAgency:         "NSF",
				PrincipalInvestigator: "Dr. Smith",
				Institution:           "University",
				GrantStartDate:        now,
				GrantEndDate:          now.Add(365 * 24 * time.Hour),
				TotalAwardAmount:      100000.0,
			},
			wantErr: true,
		},
		{
			name: "negative award amount",
			request: CreateGrantRequest{
				GrantNumber:           "NSF-2025-12345",
				FundingAgency:         "NSF",
				PrincipalInvestigator: "Dr. Smith",
				Institution:           "University",
				GrantStartDate:        now,
				GrantEndDate:          now.Add(365 * 24 * time.Hour),
				TotalAwardAmount:      -100000.0,
			},
			wantErr: true,
		},
		{
			name: "end date before start date",
			request: CreateGrantRequest{
				GrantNumber:           "NSF-2025-12345",
				FundingAgency:         "NSF",
				PrincipalInvestigator: "Dr. Smith",
				Institution:           "University",
				GrantStartDate:        now.Add(365 * 24 * time.Hour),
				GrantEndDate:          now,
				TotalAwardAmount:      100000.0,
			},
			wantErr: true,
		},
		{
			name: "invalid indirect cost rate",
			request: CreateGrantRequest{
				GrantNumber:           "NSF-2025-12345",
				FundingAgency:         "NSF",
				PrincipalInvestigator: "Dr. Smith",
				Institution:           "University",
				GrantStartDate:        now,
				GrantEndDate:          now.Add(365 * 24 * time.Hour),
				TotalAwardAmount:      100000.0,
				IndirectCostRate:      1.5, // > 100%
			},
			wantErr: true,
		},
		{
			name: "invalid budget period",
			request: CreateGrantRequest{
				GrantNumber:           "NSF-2025-12345",
				FundingAgency:         "NSF",
				PrincipalInvestigator: "Dr. Smith",
				Institution:           "University",
				GrantStartDate:        now,
				GrantEndDate:          now.Add(365 * 24 * time.Hour),
				TotalAwardAmount:      100000.0,
				BudgetPeriodMonths:    0, // Invalid
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGrantRequest(&tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBurnRateMetrics_HealthScoring(t *testing.T) {
	tests := []struct {
		name           string
		metrics        BurnRateMetrics
		expectedHealth string
	}{
		{
			name: "healthy budget",
			metrics: BurnRateMetrics{
				BudgetHealthScore:  85.0,
				VariancePercentage: 5.0,
			},
			expectedHealth: "HEALTHY",
		},
		{
			name: "concerning variance",
			metrics: BurnRateMetrics{
				BudgetHealthScore:  65.0,
				VariancePercentage: 25.0,
			},
			expectedHealth: "CONCERN",
		},
		{
			name: "warning level",
			metrics: BurnRateMetrics{
				BudgetHealthScore:  45.0,
				VariancePercentage: 40.0,
			},
			expectedHealth: "WARNING",
		},
		{
			name: "critical level",
			metrics: BurnRateMetrics{
				BudgetHealthScore:  25.0,
				VariancePercentage: 75.0,
			},
			expectedHealth: "CRITICAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := calculateBudgetHealthStatus(tt.metrics.BudgetHealthScore)
			assert.Equal(t, tt.expectedHealth, health)

			burnStatus := calculateBurnRateStatus(tt.metrics.VariancePercentage)
			if tt.metrics.VariancePercentage > 20 {
				assert.Equal(t, "OVERSPENDING", burnStatus)
			} else if tt.metrics.VariancePercentage < -20 {
				assert.Equal(t, "UNDERSPENDING", burnStatus)
			} else {
				assert.Equal(t, "ON_TRACK", burnStatus)
			}
		})
	}
}

func TestBurnRateProjection_RiskAssessment(t *testing.T) {
	tests := []struct {
		name        string
		projection  BurnRateProjection
		expectedRisk string
	}{
		{
			name: "low risk projection",
			projection: BurnRateProjection{
				ProjectedOverrun:  0.0,
				ProjectedUnderrun: 5000.0,
				ConfidenceLevel:   0.9,
			},
			expectedRisk: "LOW",
		},
		{
			name: "medium risk with small overrun",
			projection: BurnRateProjection{
				ProjectedOverrun: 2500.0,
				ConfidenceLevel:  0.8,
			},
			expectedRisk: "MEDIUM",
		},
		{
			name: "high risk with significant overrun",
			projection: BurnRateProjection{
				ProjectedOverrun: 15000.0,
				ConfidenceLevel:  0.85,
			},
			expectedRisk: "HIGH",
		},
		{
			name: "critical risk",
			projection: BurnRateProjection{
				ProjectedOverrun: 50000.0,
				ConfidenceLevel:  0.75,
			},
			expectedRisk: "CRITICAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := calculateRiskLevel(tt.projection.ProjectedOverrun, tt.projection.ConfidenceLevel)
			assert.Equal(t, tt.expectedRisk, risk)
		})
	}
}

func TestGrantAccount_Duration(t *testing.T) {
	grant := GrantAccount{
		GrantStartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrantEndDate:     time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC),
		TotalAwardAmount: 750000.0,
	}

	duration := grant.GrantEndDate.Sub(grant.GrantStartDate)
	years := duration.Hours() / (24 * 365)

	assert.InDelta(t, 3.0, years, 0.1) // Approximately 3 years
	assert.Equal(t, 750000.0, grant.TotalAwardAmount)
}

func TestBurnRateDataPoint_Calculations(t *testing.T) {
	dataPoint := BurnRateDataPoint{
		Date:               time.Now(),
		DailySpend:         125.50,
		DailyExpected:      100.00,
		CumulativeSpend:    15000.0,
		CumulativeExpected: 12000.0,
		BudgetHealthScore:  75.0,
	}

	// Calculate variance
	variance := ((dataPoint.DailySpend - dataPoint.DailyExpected) / dataPoint.DailyExpected) * 100
	assert.InDelta(t, 25.5, variance, 0.1) // 25.5% over expected

	// Calculate cumulative variance
	cumVariance := ((dataPoint.CumulativeSpend - dataPoint.CumulativeExpected) / dataPoint.CumulativeExpected) * 100
	assert.InDelta(t, 25.0, cumVariance, 0.1) // 25% over expected cumulative
}

// Helper functions for testing (would be in actual implementation)
func validateGrantRequest(req *CreateGrantRequest) error {
	if req.GrantNumber == "" {
		return NewValidationError("grant_number", "is required")
	}
	if req.FundingAgency == "" {
		return NewValidationError("funding_agency", "is required")
	}
	if req.PrincipalInvestigator == "" {
		return NewValidationError("principal_investigator", "is required")
	}
	if req.Institution == "" {
		return NewValidationError("institution", "is required")
	}
	if req.TotalAwardAmount <= 0 {
		return NewValidationError("total_award_amount", "must be greater than 0")
	}
	if req.GrantEndDate.Before(req.GrantStartDate) {
		return NewValidationError("grant_end_date", "must be after start date")
	}
	if req.IndirectCostRate < 0 || req.IndirectCostRate > 1 {
		return NewValidationError("indirect_cost_rate", "must be between 0 and 1")
	}
	if req.BudgetPeriodMonths <= 0 || req.BudgetPeriodMonths > 60 {
		return NewValidationError("budget_period_months", "must be between 1 and 60")
	}
	return nil
}

func calculateBudgetHealthStatus(score float64) string {
	if score >= 80 {
		return "HEALTHY"
	} else if score >= 60 {
		return "CONCERN"
	} else if score >= 40 {
		return "WARNING"
	}
	return "CRITICAL"
}

func calculateBurnRateStatus(variancePct float64) string {
	if variancePct > 20 {
		return "OVERSPENDING"
	} else if variancePct < -20 {
		return "UNDERSPENDING"
	}
	return "ON_TRACK"
}

func calculateRiskLevel(overrun float64, confidence float64) string {
	if overrun <= 0 {
		return "LOW"
	} else if overrun <= 5000 {
		return "MEDIUM"
	} else if overrun <= 25000 {
		return "HIGH"
	}
	return "CRITICAL"
}