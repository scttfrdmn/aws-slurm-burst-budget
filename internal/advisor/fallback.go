// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package advisor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
)

// FallbackClient provides cost estimation with graceful degradation when advisor service is unavailable
type FallbackClient struct {
	primaryClient *Client
	config        *config.IntegrationConfig
	isHealthy     bool
	lastCheck     time.Time
}

// NewFallbackClient creates a new fallback-aware advisor client
func NewFallbackClient(cfg *config.AdvisorConfig, integrationCfg *config.IntegrationConfig) *FallbackClient {
	var primaryClient *Client
	if integrationCfg.AdvisorEnabled {
		primaryClient = NewClient(cfg)
	}

	return &FallbackClient{
		primaryClient: primaryClient,
		config:        integrationCfg,
		isHealthy:     true,
		lastCheck:     time.Now(),
	}
}

// EstimateCost estimates cost with fallback mechanisms
func (fc *FallbackClient) EstimateCost(ctx context.Context, req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
	// If advisor integration is disabled, use fallback immediately
	if !fc.config.AdvisorEnabled {
		log.Info().Msg("Advisor integration disabled, using fallback cost estimation")
		return fc.fallbackEstimate(req)
	}

	// Try primary advisor client if available and healthy
	if fc.primaryClient != nil && fc.isHealthy {
		resp, err := fc.primaryClient.EstimateCost(ctx, req)
		if err == nil {
			return resp, nil
		}

		// Mark as unhealthy and log error
		log.Warn().Err(err).Msg("Advisor service unavailable, switching to fallback mode")
		fc.isHealthy = false
		fc.lastCheck = time.Now()

		// Check if we should fail strictly or fall back gracefully
		if fc.config.FailureMode == "STRICT" {
			return nil, fmt.Errorf("advisor service required but unavailable: %w", err)
		}
	}

	// Use fallback estimation
	log.Info().Str("fallback_mode", fc.config.AdvisorFallback).Msg("Using fallback cost estimation")
	return fc.fallbackEstimate(req)
}

// fallbackEstimate provides cost estimation when advisor service is unavailable
func (fc *FallbackClient) fallbackEstimate(req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
	switch fc.config.AdvisorFallback {
	case "STATIC":
		return fc.staticEstimate(req)
	case "SIMPLE":
		return fc.simpleEstimate(req)
	case "NONE":
		return nil, fmt.Errorf("advisor service unavailable and fallback disabled")
	default:
		return fc.simpleEstimate(req) // Default to simple estimation
	}
}

// staticEstimate provides a fixed cost estimate
func (fc *FallbackClient) staticEstimate(req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
	// Parse wall time to get duration
	duration := fc.parseWallTime(req.WallTime)

	// Simple calculation: nodes * CPUs * fallback_rate * hours
	cost := float64(req.Nodes*req.CPUs) * fc.config.FallbackCostRate * duration

	return &budget.CostEstimateResponse{
		EstimatedCost:  cost,
		Confidence:     0.5, // Low confidence for static estimates
		Recommendation: "Static cost estimate - advisor service unavailable",
	}, nil
}

// simpleEstimate provides basic cost estimation based on resource requirements
func (fc *FallbackClient) simpleEstimate(req *budget.CostEstimateRequest) (*budget.CostEstimateResponse, error) {
	// Parse wall time to get duration
	duration := fc.parseWallTime(req.WallTime)

	// Simple heuristic-based estimation
	var baseCost float64

	// Base cost per CPU-hour
	cpuCost := float64(req.CPUs) * fc.config.FallbackCostRate * duration

	// GPU multiplier if GPUs requested
	gpuCost := 0.0
	if req.GPUs > 0 {
		gpuCost = float64(req.GPUs) * fc.config.FallbackCostRate * 10.0 * duration // 10x multiplier for GPUs
	}

	// Memory cost estimation (if specified)
	memoryCost := 0.0
	if req.Memory != "" {
		memoryGB := fc.parseMemory(req.Memory)
		memoryCost = memoryGB * 0.01 * duration // $0.01/GB-hour
	}

	baseCost = cpuCost + gpuCost + memoryCost

	// Partition-based adjustments
	partitionMultiplier := 1.0
	switch strings.ToLower(req.Partition) {
	case "gpu", "gpu-aws":
		partitionMultiplier = 2.0 // GPU partitions more expensive
	case "high-mem", "himem":
		partitionMultiplier = 1.5 // High memory premium
	case "debug", "test":
		partitionMultiplier = 0.5 // Test partitions cheaper
	}

	finalCost := baseCost * partitionMultiplier

	// Ensure minimum cost
	if finalCost < 0.01 {
		finalCost = 0.01
	}

	confidence := 0.7 // Moderate confidence for heuristic estimates
	recommendation := "Simple heuristic estimate - advisor service unavailable"

	if finalCost > 100.0 {
		recommendation += ". Consider optimization for high-cost job."
	}

	return &budget.CostEstimateResponse{
		EstimatedCost:  finalCost,
		Confidence:     confidence,
		Recommendation: recommendation,
	}, nil
}

// parseWallTime converts wall time string to hours
func (fc *FallbackClient) parseWallTime(wallTime string) float64 {
	// Parse common formats: HH:MM:SS, HH:MM, or just minutes
	parts := strings.Split(wallTime, ":")

	var hours, minutes, seconds float64

	switch len(parts) {
	case 3: // HH:MM:SS
		if h, err := strconv.ParseFloat(parts[0], 64); err == nil {
			hours = h
		}
		if m, err := strconv.ParseFloat(parts[1], 64); err == nil {
			minutes = m
		}
		if s, err := strconv.ParseFloat(parts[2], 64); err == nil {
			seconds = s
		}
	case 2: // HH:MM
		if h, err := strconv.ParseFloat(parts[0], 64); err == nil {
			hours = h
		}
		if m, err := strconv.ParseFloat(parts[1], 64); err == nil {
			minutes = m
		}
	case 1: // Assume minutes
		if m, err := strconv.ParseFloat(parts[0], 64); err == nil {
			minutes = m
		}
	}

	totalHours := hours + (minutes / 60.0) + (seconds / 3600.0)

	// Minimum of 1 minute
	if totalHours < (1.0 / 60.0) {
		totalHours = 1.0 / 60.0
	}

	return totalHours
}

// parseMemory converts memory string to GB
func (fc *FallbackClient) parseMemory(memory string) float64 {
	memory = strings.ToUpper(memory)
	memory = strings.TrimSpace(memory)

	var value float64
	var unit string

	// Parse number and unit
	if strings.HasSuffix(memory, "GB") {
		unit = "GB"
		if v, err := strconv.ParseFloat(strings.TrimSuffix(memory, "GB"), 64); err == nil {
			value = v
		}
	} else if strings.HasSuffix(memory, "MB") {
		unit = "MB"
		if v, err := strconv.ParseFloat(strings.TrimSuffix(memory, "MB"), 64); err == nil {
			value = v
		}
	} else {
		// Assume MB if no unit
		if v, err := strconv.ParseFloat(memory, 64); err == nil {
			value = v
			unit = "MB"
		}
	}

	// Convert to GB
	switch unit {
	case "GB":
		return value
	case "MB":
		return value / 1024.0
	default:
		return 1.0 // Default 1GB
	}
}

// HealthCheck checks if the advisor service is available
func (fc *FallbackClient) HealthCheck(ctx context.Context) error {
	if !fc.config.AdvisorEnabled {
		return nil // Always healthy if disabled
	}

	if fc.primaryClient == nil {
		return fmt.Errorf("advisor client not configured")
	}

	// Only check health periodically to avoid overhead
	if time.Since(fc.lastCheck) < fc.config.HealthCheckInterval {
		if fc.isHealthy {
			return nil
		}
		return fmt.Errorf("advisor service marked unhealthy")
	}

	// Perform health check
	err := fc.primaryClient.HealthCheck(ctx)
	fc.lastCheck = time.Now()

	if err == nil {
		if !fc.isHealthy {
			log.Info().Msg("Advisor service restored, switching back from fallback mode")
		}
		fc.isHealthy = true
		return nil
	}

	fc.isHealthy = false
	return err
}

// GetStatus returns the current status of the advisor integration
func (fc *FallbackClient) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"advisor_enabled":   fc.config.AdvisorEnabled,
		"fallback_mode":     fc.config.AdvisorFallback,
		"failure_mode":      fc.config.FailureMode,
		"is_healthy":        fc.isHealthy,
		"last_health_check": fc.lastCheck,
		"operational_mode":  "standalone", // Default
	}

	if fc.config.AdvisorEnabled {
		if fc.isHealthy {
			status["operational_mode"] = "integrated"
		} else {
			status["operational_mode"] = "fallback"
		}
	}

	return status
}
