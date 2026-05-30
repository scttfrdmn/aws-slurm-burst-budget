// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package pricing

import (
	"context"
	"fmt"

	truffleaws "github.com/spore-host/truffle/pkg/aws"
)

// Truffle is the production Pricer. It delegates to truffle's AWS client, which
// resolves live spot prices (min across AZs) and on-demand prices (AWS Price
// List API, cached) — see github.com/spore-host/truffle pkg/aws/pricing.go.
type Truffle struct {
	client *truffleaws.Client
}

// NewTruffle builds a truffle-backed Pricer using ambient AWS configuration
// (shared config, environment, or instance profile). It is constructed once and
// shared; the underlying client is concurrency-safe.
func NewTruffle(ctx context.Context) (*Truffle, error) {
	c, err := truffleaws.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("pricing: init truffle client: %w", err)
	}
	return &Truffle{client: c}, nil
}

// HourlyRate implements Pricer by delegating to truffle. "reserved" is mapped to
// the on-demand baseline before the call (truffle rejects "reserved" outright).
func (t *Truffle) HourlyRate(ctx context.Context, instanceType, region, model string) (float64, error) {
	return t.client.HourlyRate(ctx, instanceType, region, normalizeModel(model))
}
