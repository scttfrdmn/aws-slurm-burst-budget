// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/discovery"
)

var ecosystemCmd = &cobra.Command{
	Use:   "ecosystem",
	Short: "Manage ecosystem service discovery and integration",
	Long: `Auto-detect and manage integration with companion tools in the
ASBA + ASBX + ASBB ecosystem. Provides operational independence with
enhanced functionality when other services are available.

Examples:
  # Discover available ecosystem services
  asbb ecosystem discover

  # Check ecosystem status
  asbb ecosystem status

  # Generate integration configuration
  asbb ecosystem config

  # Test ecosystem connectivity
  asbb ecosystem health`,
}

var ecosystemDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Auto-discover available ecosystem services",
	Long: `Automatically discover available companion tools (ASBA, ASBX, Advisor)
and provide recommendations for integration configuration.

This enables operational independence - ASBB works standalone but
enhances functionality when other services are detected.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sd := discovery.NewServiceDiscovery()

		fmt.Printf("🔍 Discovering ecosystem services...\n\n")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		services := sd.DiscoverEcosystem(ctx)

		// Display discovered services
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer func() {
			if err := w.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to flush output: %v\n", err)
			}
		}()

		if _, err := fmt.Fprintln(w, "SERVICE\tENDPOINT\tSTATUS\tVERSION\tCAPABILITIES"); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}

		for name, service := range services {
			status := "❌ UNAVAILABLE"
			if service.Available {
				status = "✅ AVAILABLE"
			}

			capabilities := "N/A"
			if len(service.Capabilities) > 0 {
				capabilities = fmt.Sprintf("%d features", len(service.Capabilities))
			}

			version := service.Version
			if version == "" {
				version = "unknown"
			}

			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name, service.Endpoint, status, version, capabilities); err != nil {
				return fmt.Errorf("failed to write service data: %w", err)
			}
		}

		fmt.Printf("\n")

		// Generate recommendations
		recommendations := sd.GenerateConfigRecommendations()
		fmt.Printf("🎯 Operational Mode: %s\n", recommendations["operational_mode"])

		if suggestions, ok := recommendations["suggestions"].([]string); ok && len(suggestions) > 0 {
			fmt.Printf("\n💡 Recommendations:\n")
			for i, suggestion := range suggestions {
				fmt.Printf("%d. %s\n", i+1, suggestion)
			}
		}

		// Show integration settings
		if integrations, ok := recommendations["integrations"].(map[string]bool); ok && len(integrations) > 0 {
			fmt.Printf("\n⚙️  Suggested Integration Settings:\n")
			for service, enabled := range integrations {
				fmt.Printf("  %s: %t\n", service, enabled)
			}
		}

		return nil
	},
}

var ecosystemStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show ecosystem integration status",
	Long:  "Display current ecosystem health and integration status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sd := discovery.NewServiceDiscovery()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Quick discovery to get current status
		sd.DiscoverEcosystem(ctx)
		status := sd.GetEcosystemStatus()

		fmt.Printf("Ecosystem Status Report\n")
		fmt.Printf("=======================\n")

		fmt.Printf("Total Services: %v\n", status["total_services"])
		fmt.Printf("Available Services: %v\n", status["available_services"])
		fmt.Printf("Ecosystem Health: %s\n", status["ecosystem_health"])
		fmt.Printf("Operational Mode: %s\n", status["operational_mode"])

		if availableList, ok := status["available_list"].([]string); ok && len(availableList) > 0 {
			fmt.Printf("\nAvailable Services:\n")
			for _, service := range availableList {
				fmt.Printf("  ✅ %s\n", service)
			}
		}

		return nil
	},
}

var ecosystemConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate integration configuration based on discovered services",
	Long:  "Generate integration configuration YAML based on auto-discovered services.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sd := discovery.NewServiceDiscovery()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		services := sd.DiscoverEcosystem(ctx)

		fmt.Printf("# Auto-generated integration configuration\n")
		fmt.Printf("# Based on ecosystem service discovery\n")
		fmt.Printf("# Generated: %s\n\n", time.Now().Format(time.RFC3339))

		fmt.Printf("integration:\n")

		// Generate advisor config
		if advisor, exists := services["advisor"]; exists {
			fmt.Printf("  # Advisor service integration\n")
			fmt.Printf("  advisor_enabled: %t\n", advisor.Available)
			if advisor.Available {
				fmt.Printf("  advisor_endpoint: \"%s\"\n", advisor.Endpoint)
				fmt.Printf("  advisor_fallback: \"SIMPLE\"  # Graceful degradation\n")
			} else {
				fmt.Printf("  advisor_fallback: \"SIMPLE\"  # Standalone operation\n")
			}
		}

		// Generate ASBX config
		if asbx, exists := services["asbx"]; exists {
			fmt.Printf("\n  # ASBX (aws-slurm-burst) integration\n")
			fmt.Printf("  asbx_enabled: %t\n", asbx.Available)
			if asbx.Available {
				fmt.Printf("  asbx_endpoint: \"%s\"\n", asbx.Endpoint)
			}
		}

		// Generate ASBA config
		if asba, exists := services["asba"]; exists {
			fmt.Printf("\n  # ASBA (Academic Slurm Burst Allocation) integration\n")
			fmt.Printf("  asba_enabled: %t\n", asba.Available)
			if asba.Available {
				fmt.Printf("  asba_endpoint: \"%s\"\n", asba.Endpoint)
			}
		}

		fmt.Printf("\n  # Graceful degradation settings\n")
		fmt.Printf("  failure_mode: \"GRACEFUL\"  # Continue operating if services fail\n")
		fmt.Printf("  health_check_interval: \"60s\"\n")
		fmt.Printf("  retry_attempts: 3\n")

		return nil
	},
}

var ecosystemHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Test ecosystem service connectivity",
	Long:  "Test connectivity to all ecosystem services and report health status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sd := discovery.NewServiceDiscovery()

		fmt.Printf("🏥 Testing ecosystem service health...\n\n")

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		services := sd.DiscoverEcosystem(ctx)

		allHealthy := true
		for name, service := range services {
			icon := "❌"
			status := "UNAVAILABLE"

			if service.Available {
				icon = "✅"
				status = "HEALTHY"
			} else {
				allHealthy = false
			}

			fmt.Printf("%s %s: %s", icon, name, status)
			if service.Available && service.Version != "" {
				fmt.Printf(" (v%s)", service.Version)
			}
			fmt.Printf("\n")

			if len(service.Capabilities) > 0 {
				fmt.Printf("   Capabilities: %v\n", service.Capabilities)
			}
		}

		fmt.Printf("\n")

		if allHealthy {
			fmt.Printf("🎉 Complete ecosystem available - full functionality enabled\n")
		} else {
			fmt.Printf("ℹ️  Partial ecosystem - operating with graceful degradation\n")
			fmt.Printf("   All core functionality remains available\n")
		}

		ecosystemStatus := sd.GetEcosystemStatus()
		fmt.Printf("\n📊 Ecosystem Health: %s\n", ecosystemStatus["ecosystem_health"])
		fmt.Printf("🔧 Operational Mode: %s\n", ecosystemStatus["operational_mode"])

		return nil
	},
}

func init() {
	ecosystemCmd.AddCommand(ecosystemDiscoverCmd)
	ecosystemCmd.AddCommand(ecosystemStatusCmd)
	ecosystemCmd.AddCommand(ecosystemConfigCmd)
	ecosystemCmd.AddCommand(ecosystemHealthCmd)
}
