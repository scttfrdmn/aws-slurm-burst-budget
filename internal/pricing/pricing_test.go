// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package pricing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatic_HourlyRate(t *testing.T) {
	ctx := context.Background()
	p := Static{OnDemand: 2.0, SpotDiscount: 0.7}

	tests := []struct {
		name  string
		model string
		want  float64
	}{
		{"on-demand explicit", "on-demand", 2.0},
		{"on-demand alias ondemand", "ondemand", 2.0},
		{"empty defaults to on-demand", "", 2.0},
		{"spot applies discount", "spot", 0.6}, // 2.0 * (1 - 0.7)
		{"reserved maps to on-demand baseline", "reserved", 2.0},
		{"unknown maps to on-demand", "bogus", 2.0},
		{"case-insensitive spot", "SPOT", 0.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.HourlyRate(ctx, "c6i.large", "us-east-1", tt.model)
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestStatic_HourlyRate_Errors(t *testing.T) {
	ctx := context.Background()

	// No configured rate.
	_, err := Static{}.HourlyRate(ctx, "c6i.large", "us-east-1", "on-demand")
	assert.Error(t, err)

	// Spot discount of 1.0 yields a non-positive rate.
	_, err = Static{OnDemand: 1.0, SpotDiscount: 1.0}.HourlyRate(ctx, "c6i.large", "us-east-1", "spot")
	assert.Error(t, err)
}

func TestNormalizeModel(t *testing.T) {
	assert.Equal(t, ModelOnDemand, normalizeModel(""))
	assert.Equal(t, ModelOnDemand, normalizeModel("On-Demand"))
	assert.Equal(t, ModelOnDemand, normalizeModel("on_demand"))
	assert.Equal(t, ModelSpot, normalizeModel("spot"))
	assert.Equal(t, ModelSpot, normalizeModel(" SPOT "))
	// reserved is priced as the on-demand baseline.
	assert.Equal(t, ModelOnDemand, normalizeModel("reserved"))
	// unknown is conservatively priced as on-demand.
	assert.Equal(t, ModelOnDemand, normalizeModel("weird"))
}

// Static must satisfy Pricer.
var _ Pricer = Static{}
