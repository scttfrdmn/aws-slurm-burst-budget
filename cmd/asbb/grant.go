// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

var grantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Manage research grants and long-term funding",
	Long: `Manage research grants with multi-year funding periods, track burn rates,
and generate compliance reports for funding agencies.

Examples:
  # Create a 3-year NSF grant
  asbb grant create --number=NSF-2025-12345 --agency="National Science Foundation" \
    --pi="Dr. Jane Smith" --institution="University Research" \
    --amount=750000 --start=2025-01-01 --end=2027-12-31 --periods=36

  # List active grants
  asbb grant list --active

  # Show grant details with burn rate analysis
  asbb grant show NSF-2025-12345

  # Generate annual report for funding agency
  asbb grant report NSF-2025-12345 --type=annual --format=pdf --period=1`,
}

var (
	createGrantNumber     string
	createGrantAgency     string
	createGrantProgram    string
	createGrantPI         string
	createGrantCoPI       []string
	createGrantInst       string
	createGrantDept       string
	createGrantAmount     float64
	createGrantStart      string
	createGrantEnd        string
	createGrantPeriods    int
	createGrantIndirect   float64
	createGrantFedID      string
	createGrantProject    string
	createGrantCostCenter string
)

var grantCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new research grant account",
	Long: `Create a new research grant account with multi-year funding support.

This creates a grant account that can span multiple years with automatic
budget period management and burn rate tracking.

Example:
  # 3-year NSF grant with $250K/year
  asbb grant create \
    --number=NSF-2025-12345 \
    --agency="National Science Foundation" \
    --program="Computer and Information Science and Engineering" \
    --pi="Dr. Jane Smith" \
    --co-pi="Dr. Bob Johnson,Dr. Alice Chen" \
    --institution="Research University" \
    --department="Computer Science" \
    --amount=750000 \
    --start=2025-01-01 \
    --end=2027-12-31 \
    --periods=36 \
    --indirect=0.30 \
    --federal-id="47.070" \
    --project="ML-Research-2025"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Parse dates
		startDate, err := time.Parse("2006-01-02", createGrantStart)
		if err != nil {
			return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
		}

		endDate, err := time.Parse("2006-01-02", createGrantEnd)
		if err != nil {
			return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
		}

		// Parse co-PIs if provided
		var coPIs []string
		if len(createGrantCoPI) > 0 {
			for _, coPI := range createGrantCoPI {
				if strings.Contains(coPI, ",") {
					coPIs = append(coPIs, strings.Split(coPI, ",")...)
				} else {
					coPIs = append(coPIs, coPI)
				}
			}
		}

		req := &api.CreateGrantRequest{
			GrantNumber:           createGrantNumber,
			FundingAgency:         createGrantAgency,
			AgencyProgram:         createGrantProgram,
			PrincipalInvestigator: createGrantPI,
			CoInvestigators:       coPIs,
			Institution:           createGrantInst,
			Department:            createGrantDept,
			GrantStartDate:        startDate,
			GrantEndDate:          endDate,
			TotalAwardAmount:      createGrantAmount,
			IndirectCostRate:      createGrantIndirect,
			BudgetPeriodMonths:    createGrantPeriods,
			FederalAwardID:        createGrantFedID,
			InternalProjectCode:   createGrantProject,
			CostCenter:            createGrantCostCenter,
		}

		grant, err := client.CreateGrant(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to create grant: %w", err)
		}

		fmt.Printf("✅ Research grant created successfully!\n")
		fmt.Printf("Grant Number: %s\n", grant.GrantNumber)
		fmt.Printf("Funding Agency: %s\n", grant.FundingAgency)
		fmt.Printf("Principal Investigator: %s\n", grant.PrincipalInvestigator)
		fmt.Printf("Total Award: $%.2f\n", grant.TotalAwardAmount)
		fmt.Printf("Grant Period: %s to %s\n",
			grant.GrantStartDate.Format("2006-01-02"),
			grant.GrantEndDate.Format("2006-01-02"))

		// Calculate grant duration
		duration := grant.GrantEndDate.Sub(grant.GrantStartDate)
		years := duration.Hours() / (24 * 365)
		fmt.Printf("Duration: %.1f years (%d months)\n", years, int(duration.Hours()/(24*30)))

		if grant.BudgetPeriodMonths > 0 {
			periodsPerYear := 12 / grant.BudgetPeriodMonths
			totalPeriods := int(years) * periodsPerYear
			fmt.Printf("Budget Periods: %d periods of %d months each\n", totalPeriods, grant.BudgetPeriodMonths)
		}

		return nil
	},
}

var grantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List research grants",
	Long:  "List all research grants with their status and key metrics.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		grants, err := client.ListGrants(cmd.Context(), &api.GrantListRequest{})
		if err != nil {
			return fmt.Errorf("failed to list grants: %w", err)
		}

		if len(grants) == 0 {
			fmt.Println("No research grants found.")
			return nil
		}

		// Create tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Print header
		fmt.Fprintln(w, "GRANT NUMBER\tAGENCY\tPI\tAWARD AMOUNT\tPERIOD\tSTATUS")

		for _, grant := range grants {
			fmt.Fprintf(w, "%s\t%s\t%s\t$%.0f\t%s - %s\t%s\n",
				grant.GrantNumber,
				grant.FundingAgency,
				grant.PrincipalInvestigator,
				grant.TotalAwardAmount,
				grant.GrantStartDate.Format("2006-01-02"),
				grant.GrantEndDate.Format("2006-01-02"),
				grant.Status,
			)
		}

		return nil
	},
}

var grantShowCmd = &cobra.Command{
	Use:   "show <grant-number>",
	Short: "Show detailed grant information with burn rate analysis",
	Long:  "Show comprehensive grant details including budget periods, burn rates, and projections.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		grantNumber := args[0]

		// Get grant details
		grant, err := client.GetGrant(cmd.Context(), grantNumber)
		if err != nil {
			return fmt.Errorf("failed to get grant: %w", err)
		}

		// Get burn rate analysis
		burnRateReq := &api.BurnRateAnalysisRequest{
			GrantNumber:       grantNumber,
			AnalysisPeriod:    "30d",
			IncludeProjection: true,
			IncludeAlerts:     true,
		}

		burnAnalysis, err := client.GetBurnRateAnalysis(cmd.Context(), burnRateReq)
		if err != nil {
			fmt.Printf("Warning: Could not get burn rate analysis: %v\n", err)
		}

		// Display grant information
		fmt.Printf("Grant Details:\n")
		fmt.Printf("==============\n")
		fmt.Printf("Grant Number: %s\n", grant.GrantNumber)
		fmt.Printf("Funding Agency: %s\n", grant.FundingAgency)
		if grant.AgencyProgram != "" {
			fmt.Printf("Program: %s\n", grant.AgencyProgram)
		}
		fmt.Printf("Principal Investigator: %s\n", grant.PrincipalInvestigator)
		if len(grant.CoInvestigators) > 0 {
			fmt.Printf("Co-Investigators: %s\n", strings.Join(grant.CoInvestigators, ", "))
		}
		fmt.Printf("Institution: %s\n", grant.Institution)
		if grant.Department != "" {
			fmt.Printf("Department: %s\n", grant.Department)
		}

		fmt.Printf("\nFinancial Summary:\n")
		fmt.Printf("Total Award: $%.2f\n", grant.TotalAwardAmount)
		fmt.Printf("Direct Costs: $%.2f\n", grant.DirectCosts)
		if grant.IndirectCostRate > 0 {
			fmt.Printf("Indirect Rate: %.1f%% ($%.2f)\n",
				grant.IndirectCostRate*100, grant.IndirectCosts)
		}

		fmt.Printf("\nGrant Period:\n")
		fmt.Printf("Start: %s\n", grant.GrantStartDate.Format("2006-01-02"))
		fmt.Printf("End: %s\n", grant.GrantEndDate.Format("2006-01-02"))

		duration := grant.GrantEndDate.Sub(grant.GrantStartDate)
		fmt.Printf("Duration: %.1f years\n", duration.Hours()/(24*365))

		if grant.BudgetPeriodMonths > 0 {
			fmt.Printf("Budget Periods: %d months per period (Period %d of %d)\n",
				grant.BudgetPeriodMonths,
				grant.CurrentBudgetPeriod,
				int(duration.Hours()/(24*30))/grant.BudgetPeriodMonths)
		}

		fmt.Printf("Status: %s\n", strings.ToUpper(grant.Status))

		// Display burn rate analysis if available
		if burnAnalysis != nil {
			fmt.Printf("\nBurn Rate Analysis (Last 30 Days):\n")
			fmt.Printf("===================================\n")

			metrics := burnAnalysis.CurrentMetrics
			fmt.Printf("Current Daily Rate: $%.2f (Expected: $%.2f)\n",
				metrics.DailySpendRate, metrics.DailyExpectedRate)

			if metrics.VariancePercentage != 0 {
				symbol := "+"
				if metrics.VariancePercentage < 0 {
					symbol = ""
				}
				fmt.Printf("Variance: %s%.1f%% (%s)\n",
					symbol, metrics.VariancePercentage, metrics.BurnRateStatus)
			}

			fmt.Printf("7-Day Average: $%.2f\n", metrics.Rolling7DayAverage)
			fmt.Printf("30-Day Average: $%.2f\n", metrics.Rolling30DayAverage)
			fmt.Printf("Budget Health Score: %.1f%% (%s)\n",
				metrics.BudgetHealthScore, metrics.BudgetHealthStatus)

			fmt.Printf("\nBudget Status:\n")
			fmt.Printf("Remaining: $%.2f (%.1f%%)\n",
				metrics.BudgetRemainingAmount, metrics.BudgetRemainingPercent)
			fmt.Printf("Days Remaining: %d\n", metrics.TimeRemainingDays)

			// Show projection if available
			if burnAnalysis.Projection != nil {
				proj := burnAnalysis.Projection
				fmt.Printf("\nProjection:\n")
				fmt.Printf("Projected End Spend: $%.2f\n", proj.ProjectedFinalSpend)

				if proj.ProjectedOverrun > 0 {
					fmt.Printf("⚠️  Projected Overrun: $%.2f\n", proj.ProjectedOverrun)
				} else if proj.ProjectedUnderrun > 0 {
					fmt.Printf("💰 Projected Underrun: $%.2f\n", proj.ProjectedUnderrun)
				}

				fmt.Printf("Risk Level: %s\n", proj.RiskLevel)
				fmt.Printf("Confidence: %.1f%%\n", proj.ConfidenceLevel*100)
			}

			// Show alerts if any
			if len(burnAnalysis.Alerts) > 0 {
				fmt.Printf("\n🚨 Active Alerts:\n")
				for _, alert := range burnAnalysis.Alerts {
					severityIcon := "ℹ️"
					if alert.Severity == "warning" {
						severityIcon = "⚠️"
					} else if alert.Severity == "critical" {
						severityIcon = "🚨"
					}

					fmt.Printf("%s %s: %s\n", severityIcon, strings.ToUpper(alert.Severity), alert.Message)
				}
			}

			// Show recommendations
			if len(burnAnalysis.Recommendations) > 0 {
				fmt.Printf("\n💡 Recommendations:\n")
				for i, rec := range burnAnalysis.Recommendations {
					fmt.Printf("%d. %s\n", i+1, rec)
				}
			}
		}

		return nil
	},
}

var burnRateCmd = &cobra.Command{
	Use:   "burn-rate <account|grant-number>",
	Short: "Analyze burn rate and spending patterns",
	Long: `Analyze spending burn rate over time with projections and alerts.

Examples:
  # Analyze account burn rate over last 30 days
  asbb burn-rate proj001 --period=30d

  # Analyze grant burn rate with projections
  asbb burn-rate NSF-2025-12345 --period=90d --projection

  # Get burn rate alerts
  asbb burn-rate proj001 --alerts-only`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		target := args[0]
		period, _ := cmd.Flags().GetString("period")
		showProjection, _ := cmd.Flags().GetBool("projection")
		alertsOnly, _ := cmd.Flags().GetBool("alerts-only")

		req := &api.BurnRateAnalysisRequest{
			AnalysisPeriod:    period,
			IncludeProjection: showProjection,
			IncludeAlerts:     true,
		}

		// Determine if target is grant number or account
		if strings.Contains(target, "-") && (strings.HasPrefix(target, "NSF") ||
			strings.HasPrefix(target, "NIH") || strings.HasPrefix(target, "DOE")) {
			req.GrantNumber = target
		} else {
			req.Account = target
		}

		analysis, err := client.GetBurnRateAnalysis(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to get burn rate analysis: %w", err)
		}

		if alertsOnly {
			// Show only alerts
			if len(analysis.Alerts) == 0 {
				fmt.Printf("✅ No active alerts for %s\n", target)
				return nil
			}

			fmt.Printf("🚨 Active Alerts for %s:\n", target)
			for _, alert := range analysis.Alerts {
				severityIcon := "ℹ️"
				if alert.Severity == "warning" {
					severityIcon = "⚠️"
				} else if alert.Severity == "critical" {
					severityIcon = "🚨"
				}

				fmt.Printf("%s [%s] %s: %s\n",
					severityIcon,
					strings.ToUpper(alert.Severity),
					strings.ToUpper(alert.AlertType),
					alert.Message)
			}
			return nil
		}

		// Full burn rate analysis display
		fmt.Printf("Burn Rate Analysis: %s\n", target)
		fmt.Printf("Analysis Period: %s (%s to %s)\n",
			analysis.AnalysisPeriod,
			analysis.TimeRange.StartDate.Format("2006-01-02"),
			analysis.TimeRange.EndDate.Format("2006-01-02"))
		fmt.Printf("========================%s\n", strings.Repeat("=", len(target)))

		metrics := analysis.CurrentMetrics

		fmt.Printf("\nCurrent Metrics:\n")
		fmt.Printf("Daily Spend Rate: $%.2f (Expected: $%.2f)\n",
			metrics.DailySpendRate, metrics.DailyExpectedRate)

		if metrics.VariancePercentage != 0 {
			symbol := "+"
			statusIcon := "📊"
			if metrics.VariancePercentage < 0 {
				symbol = ""
				statusIcon = "📉"
			} else if metrics.VariancePercentage > 20 {
				statusIcon = "📈"
			}

			fmt.Printf("Variance: %s%s%.1f%% %s (%s)\n",
				statusIcon, symbol, metrics.VariancePercentage,
				strings.ToLower(metrics.BurnRateStatus), statusIcon)
		}

		fmt.Printf("Budget Health: %.1f%% (%s)\n",
			metrics.BudgetHealthScore, metrics.BudgetHealthStatus)

		fmt.Printf("\nRolling Averages:\n")
		fmt.Printf("7-Day Average: $%.2f/day\n", metrics.Rolling7DayAverage)
		fmt.Printf("30-Day Average: $%.2f/day\n", metrics.Rolling30DayAverage)

		fmt.Printf("\nBudget Status:\n")
		fmt.Printf("Remaining: $%.2f (%.1f%% of total)\n",
			metrics.BudgetRemainingAmount, metrics.BudgetRemainingPercent)
		fmt.Printf("Time Remaining: %d days\n", metrics.TimeRemainingDays)

		// Show projection if requested and available
		if showProjection && analysis.Projection != nil {
			proj := analysis.Projection
			fmt.Printf("\nProjection Analysis:\n")
			fmt.Printf("Projected Final Spend: $%.2f\n", proj.ProjectedFinalSpend)

			if proj.ProjectedOverrun > 0 {
				fmt.Printf("⚠️  Projected Overrun: $%.2f\n", proj.ProjectedOverrun)
			} else if proj.ProjectedUnderrun > 0 {
				fmt.Printf("💰 Projected Underrun: $%.2f\n", proj.ProjectedUnderrun)
			}

			if proj.ProjectedDepletionDate != nil {
				fmt.Printf("Projected Depletion: %s\n", proj.ProjectedDepletionDate.Format("2006-01-02"))
			}

			fmt.Printf("Risk Level: %s (%.1f%% confidence)\n",
				proj.RiskLevel, proj.ConfidenceLevel*100)
		}

		return nil
	},
}

func init() {
	// Grant create command flags
	grantCreateCmd.Flags().StringVar(&createGrantNumber, "number", "", "Grant number (required)")
	grantCreateCmd.Flags().StringVar(&createGrantAgency, "agency", "", "Funding agency (required)")
	grantCreateCmd.Flags().StringVar(&createGrantProgram, "program", "", "Agency program")
	grantCreateCmd.Flags().StringVar(&createGrantPI, "pi", "", "Principal investigator (required)")
	grantCreateCmd.Flags().StringSliceVar(&createGrantCoPI, "co-pi", []string{}, "Co-investigators (comma-separated)")
	grantCreateCmd.Flags().StringVar(&createGrantInst, "institution", "", "Institution (required)")
	grantCreateCmd.Flags().StringVar(&createGrantDept, "department", "", "Department")
	grantCreateCmd.Flags().Float64Var(&createGrantAmount, "amount", 0, "Total award amount (required)")
	grantCreateCmd.Flags().StringVar(&createGrantStart, "start", "", "Grant start date YYYY-MM-DD (required)")
	grantCreateCmd.Flags().StringVar(&createGrantEnd, "end", "", "Grant end date YYYY-MM-DD (required)")
	grantCreateCmd.Flags().IntVar(&createGrantPeriods, "periods", 12, "Budget period length in months")
	grantCreateCmd.Flags().Float64Var(&createGrantIndirect, "indirect", 0.30, "Indirect cost rate (0.0-1.0)")
	grantCreateCmd.Flags().StringVar(&createGrantFedID, "federal-id", "", "Federal award ID (CFDA)")
	grantCreateCmd.Flags().StringVar(&createGrantProject, "project", "", "Internal project code")
	grantCreateCmd.Flags().StringVar(&createGrantCostCenter, "cost-center", "", "Cost center")

	if err := grantCreateCmd.MarkFlagRequired("number"); err != nil {
		panic(err)
	}
	if err := grantCreateCmd.MarkFlagRequired("agency"); err != nil {
		panic(err)
	}
	if err := grantCreateCmd.MarkFlagRequired("pi"); err != nil {
		panic(err)
	}
	if err := grantCreateCmd.MarkFlagRequired("institution"); err != nil {
		panic(err)
	}
	if err := grantCreateCmd.MarkFlagRequired("amount"); err != nil {
		panic(err)
	}
	if err := grantCreateCmd.MarkFlagRequired("start"); err != nil {
		panic(err)
	}
	if err := grantCreateCmd.MarkFlagRequired("end"); err != nil {
		panic(err)
	}

	// Burn rate command flags
	burnRateCmd.Flags().String("period", "30d", "Analysis period (7d, 30d, 90d, 6m, 1y)")
	burnRateCmd.Flags().Bool("projection", false, "Include spending projections")
	burnRateCmd.Flags().Bool("alerts-only", false, "Show only active alerts")

	// Add commands to parent
	grantCmd.AddCommand(grantCreateCmd)
	grantCmd.AddCommand(grantListCmd)
	grantCmd.AddCommand(grantShowCmd)
}