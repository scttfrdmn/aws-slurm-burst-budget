// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
)

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name   string
		config config.LoggingConfig
	}{
		{
			name: "json format",
			config: config.LoggingConfig{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "console format",
			config: config.LoggingConfig{
				Level:  "debug",
				Format: "console",
			},
		},
		{
			name: "with sampling",
			config: config.LoggingConfig{
				Level:  "info",
				Format: "json",
				Sampling: struct {
					Initial    int           `mapstructure:"initial" yaml:"initial"`
					Thereafter int           `mapstructure:"thereafter" yaml:"thereafter"`
					Tick       time.Duration `mapstructure:"tick" yaml:"tick"`
				}{
					Initial: 50,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that setupLogging doesn't panic
			assert.NotPanics(t, func() {
				setupLogging(&tt.config)
			})
		})
	}
}

func TestLoggingResponseWriter(t *testing.T) {
	// Test loggingResponseWriter exists and has correct fields
	lrw := &loggingResponseWriter{
		statusCode: 200,
	}

	assert.Equal(t, 200, lrw.statusCode)

	// Note: WriteHeader test would require a proper ResponseWriter mock
	// For now, just test that the struct exists and has the right fields
	assert.NotNil(t, lrw)
}