// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package pricing is ASBB's source of real AWS instance prices. It exists so the
// budget service can size a hold from an actual per-hour rate rather than the
// $0.10/CPU-hour heuristic the advisor fallback uses.
//
// It is the first place ASBB touches AWS pricing. The dependency is isolated
// behind the Pricer interface: the production implementation delegates to
// truffle (github.com/spore-host/truffle/pkg/aws), which queries live spot and
// on-demand prices; a static implementation backs tests and the degraded path
// when AWS/truffle is unreachable. Callers depend only on Pricer.
package pricing

import (
	"context"
	"fmt"
	"strings"
)

// Capacity models accepted by HourlyRate. They mirror the strings q0 sends in a
// fleet admission request (its cohort.CapacityModel rendered to a string).
const (
	ModelSpot     = "spot"
	ModelOnDemand = "on-demand"
	ModelReserved = "reserved"
)

// Pricer resolves the current $/hr for one instance type in one region under a
// purchase model. Implementations must be safe for concurrent use.
//
// This is intentionally identical in shape to truffle's
// (*aws.Client).HourlyRate, so the production implementation is a thin delegate.
type Pricer interface {
	// HourlyRate returns the current on-demand or spot $/hr for instanceType in
	// region. model is case-insensitive; "" is treated as on-demand. A
	// (0, error) result means no price could be determined.
	HourlyRate(ctx context.Context, instanceType, region, model string) (float64, error)
}

// normalizeModel folds the accepted aliases to a canonical model string and maps
// "reserved" onto the on-demand baseline. A reserved rate is not a point-in-time
// price (it depends on term, payment option, and offering class), so ASBB prices
// it as on-demand and leaves any reserved discount to budget policy — matching
// the semantics decision in issue #10 and truffle's own rejection of "reserved".
func normalizeModel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "", ModelOnDemand, "ondemand", "on_demand":
		return ModelOnDemand
	case ModelSpot:
		return ModelSpot
	case ModelReserved:
		return ModelOnDemand
	default:
		return ModelOnDemand
	}
}

// Static is a Pricer that returns fixed per-hour rates. It backs tests and the
// degraded path when the truffle-backed pricer is unavailable, so a budget hold
// can still be sized (coarsely) rather than failing the admission outright.
type Static struct {
	// OnDemand is the $/hr charged for the on-demand (and reserved) model.
	OnDemand float64
	// SpotDiscount is the fraction (0..1) subtracted from OnDemand for the spot
	// model; e.g. 0.7 yields a spot rate of 30% of on-demand.
	SpotDiscount float64
}

// HourlyRate implements Pricer with the configured static rates.
func (s Static) HourlyRate(_ context.Context, instanceType, _, model string) (float64, error) {
	if s.OnDemand <= 0 {
		return 0, fmt.Errorf("pricing: no static on-demand rate configured for %s", instanceType)
	}
	if normalizeModel(model) == ModelSpot {
		rate := s.OnDemand * (1 - s.SpotDiscount)
		if rate <= 0 {
			return 0, fmt.Errorf("pricing: static spot rate for %s is non-positive", instanceType)
		}
		return rate, nil
	}
	return s.OnDemand, nil
}
