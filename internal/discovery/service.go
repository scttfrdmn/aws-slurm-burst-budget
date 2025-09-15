// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// ServiceDiscovery handles auto-detection of ecosystem companion tools
type ServiceDiscovery struct {
	httpClient *http.Client
	services   map[string]*ServiceInfo
}

// ServiceInfo represents information about a discovered service
type ServiceInfo struct {
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Endpoint     string    `json:"endpoint"`
	Available    bool      `json:"available"`
	LastCheck    time.Time `json:"last_check"`
	Capabilities []string  `json:"capabilities"`
	HealthStatus string    `json:"health_status"`
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery() *ServiceDiscovery {
	return &ServiceDiscovery{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		services:   make(map[string]*ServiceInfo),
	}
}

// DiscoverEcosystem auto-detects available companion tools in the ecosystem
func (sd *ServiceDiscovery) DiscoverEcosystem(ctx context.Context) map[string]*ServiceInfo {
	log.Info().Msg("Starting ecosystem service discovery...")

	// Common service discovery endpoints and ports
	discoveryTargets := map[string][]string{
		"advisor": {
			"http://localhost:8081",
			"http://advisor:8081",
			"http://aws-slurm-burst-advisor:8081",
		},
		"asbx": {
			"http://localhost:8082",
			"http://asbx:8082",
			"http://aws-slurm-burst:8082",
		},
		"asba": {
			"http://localhost:8083",
			"http://asba:8083",
			"http://academic-slurm-burst:8083",
		},
	}

	for serviceName, endpoints := range discoveryTargets {
		sd.discoverService(ctx, serviceName, endpoints)
	}

	return sd.services
}

// discoverService attempts to discover a specific service
func (sd *ServiceDiscovery) discoverService(ctx context.Context, serviceName string, endpoints []string) {
	for _, endpoint := range endpoints {
		if sd.probeService(ctx, serviceName, endpoint) {
			log.Info().
				Str("service", serviceName).
				Str("endpoint", endpoint).
				Msg("Ecosystem service discovered")
			return // Found it, stop looking
		}
	}

	// Service not found
	sd.services[serviceName] = &ServiceInfo{
		Name:         serviceName,
		Endpoint:     endpoints[0], // Use first as default
		Available:    false,
		LastCheck:    time.Now(),
		HealthStatus: "not_found",
	}

	log.Debug().Str("service", serviceName).Msg("Ecosystem service not available")
}

// probeService checks if a service is available at the given endpoint
func (sd *ServiceDiscovery) probeService(ctx context.Context, serviceName, endpoint string) bool {
	// Try common health check endpoints
	healthEndpoints := []string{
		"/health",
		"/api/v1/health",
		"/status",
		"/version",
	}

	for _, healthPath := range healthEndpoints {
		url := endpoint + healthPath

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := sd.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// Try to parse service information
			var serviceInfo map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&serviceInfo); err == nil {
				sd.parseServiceInfo(serviceName, endpoint, serviceInfo)
			} else {
				// Basic service info if parsing fails
				sd.services[serviceName] = &ServiceInfo{
					Name:         serviceName,
					Endpoint:     endpoint,
					Available:    true,
					LastCheck:    time.Now(),
					HealthStatus: "healthy",
				}
			}

			if err := resp.Body.Close(); err != nil {
				// Log error but continue
				_ = err
			}
			return true
		}

		if err := resp.Body.Close(); err != nil {
			// Log error but continue
			_ = err
		}
	}

	return false
}

// parseServiceInfo extracts service information from health check response
func (sd *ServiceDiscovery) parseServiceInfo(serviceName, endpoint string, info map[string]interface{}) {
	serviceInfo := &ServiceInfo{
		Name:         serviceName,
		Endpoint:     endpoint,
		Available:    true,
		LastCheck:    time.Now(),
		HealthStatus: "healthy",
		Capabilities: []string{},
	}

	// Extract version if available
	if version, ok := info["version"].(string); ok {
		serviceInfo.Version = version
	}

	// Extract capabilities based on service type
	switch serviceName {
	case "advisor":
		serviceInfo.Capabilities = []string{"cost_estimation", "performance_analysis"}
		if features, ok := info["features"].([]interface{}); ok {
			for _, feature := range features {
				if featureStr, ok := feature.(string); ok {
					serviceInfo.Capabilities = append(serviceInfo.Capabilities, featureStr)
				}
			}
		}
	case "asbx":
		serviceInfo.Capabilities = []string{"cost_reconciliation", "performance_data", "job_tracking"}
	case "asba":
		serviceInfo.Capabilities = []string{"decision_making", "resource_allocation", "burst_optimization"}
	}

	sd.services[serviceName] = serviceInfo
}

// GetService returns information about a specific service
func (sd *ServiceDiscovery) GetService(serviceName string) (*ServiceInfo, bool) {
	service, exists := sd.services[serviceName]
	return service, exists
}

// IsServiceAvailable checks if a service is available
func (sd *ServiceDiscovery) IsServiceAvailable(serviceName string) bool {
	if service, exists := sd.services[serviceName]; exists {
		return service.Available
	}
	return false
}

// GetAvailableServices returns a list of all available services
func (sd *ServiceDiscovery) GetAvailableServices() []string {
	var available []string
	for name, service := range sd.services {
		if service.Available {
			available = append(available, name)
		}
	}
	return available
}

// GenerateConfigRecommendations suggests configuration based on discovered services
func (sd *ServiceDiscovery) GenerateConfigRecommendations() map[string]interface{} {
	recommendations := map[string]interface{}{
		"operational_mode": "standalone", // Default
		"integrations":     map[string]bool{},
		"suggestions":      []string{},
	}

	integrations := recommendations["integrations"].(map[string]bool)
	suggestions := []string{}

	// Check each service and provide recommendations
	if advisor, exists := sd.services["advisor"]; exists && advisor.Available {
		integrations["advisor_enabled"] = true
		suggestions = append(suggestions,
			fmt.Sprintf("Advisor service detected at %s - enable for improved cost estimation", advisor.Endpoint))
		recommendations["operational_mode"] = "enhanced"
	}

	if asbx, exists := sd.services["asbx"]; exists && asbx.Available {
		integrations["asbx_enabled"] = true
		suggestions = append(suggestions,
			fmt.Sprintf("ASBX service detected at %s - enable for automatic cost reconciliation", asbx.Endpoint))
		recommendations["operational_mode"] = "integrated"
	}

	if asba, exists := sd.services["asba"]; exists && asba.Available {
		integrations["asba_enabled"] = true
		suggestions = append(suggestions,
			fmt.Sprintf("ASBA service detected at %s - enable for intelligent decision making", asba.Endpoint))
		recommendations["operational_mode"] = "ecosystem"
	}

	// Determine overall ecosystem status
	availableCount := len(sd.GetAvailableServices())
	switch availableCount {
	case 0:
		suggestions = append(suggestions, "Running in standalone mode - all core functionality available")
		suggestions = append(suggestions, "Consider deploying advisor service for enhanced cost estimation")
	case 1:
		suggestions = append(suggestions, "Partial ecosystem detected - consider deploying additional services")
	case 2:
		suggestions = append(suggestions, "Most ecosystem services available - consider full deployment")
	case 3:
		suggestions = append(suggestions, "Complete ecosystem detected - enable all integrations for full functionality")
		recommendations["operational_mode"] = "complete_ecosystem"
	}

	recommendations["suggestions"] = suggestions
	return recommendations
}

// RefreshService updates the status of a specific service
func (sd *ServiceDiscovery) RefreshService(ctx context.Context, serviceName string) bool {
	if service, exists := sd.services[serviceName]; exists {
		// Re-probe the service
		available := sd.probeService(ctx, serviceName, service.Endpoint)
		service.Available = available
		service.LastCheck = time.Now()

		if available {
			service.HealthStatus = "healthy"
		} else {
			service.HealthStatus = "unavailable"
		}

		return available
	}
	return false
}

// GetEcosystemStatus returns overall ecosystem health and recommendations
func (sd *ServiceDiscovery) GetEcosystemStatus() map[string]interface{} {
	availableServices := sd.GetAvailableServices()
	totalServices := len(sd.services)
	availableCount := len(availableServices)

	status := map[string]interface{}{
		"total_services":     totalServices,
		"available_services": availableCount,
		"available_list":     availableServices,
		"ecosystem_health":   "unknown",
		"operational_mode":   "standalone",
		"recommendations":    sd.GenerateConfigRecommendations(),
	}

	// Determine ecosystem health
	healthPercentage := float64(availableCount) / float64(totalServices) * 100
	switch {
	case healthPercentage >= 100:
		status["ecosystem_health"] = "complete"
		status["operational_mode"] = "complete_ecosystem"
	case healthPercentage >= 66:
		status["ecosystem_health"] = "good"
		status["operational_mode"] = "integrated"
	case healthPercentage >= 33:
		status["ecosystem_health"] = "partial"
		status["operational_mode"] = "enhanced"
	default:
		status["ecosystem_health"] = "standalone"
		status["operational_mode"] = "standalone"
	}

	return status
}
