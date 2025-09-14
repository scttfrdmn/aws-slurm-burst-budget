// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage budget accounts",
	Long: `Manage budget accounts including creation, updates, and incremental allocation schedules.

Examples:
  # List all accounts
  asbb account list

  # Create a simple account
  asbb account create --name="Research" --account=proj001 --budget=1000 --start=2025-01-01 --end=2025-12-31

  # Create account with incremental allocation
  asbb account create --name="Research" --account=proj001 --incremental --total-budget=600 --allocation-amount=100 --allocation-frequency=monthly --start=2025-01-01

  # Show account details
  asbb account show proj001`,
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List budget accounts",
	Long:  "List all budget accounts with their current status and usage information.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		accounts, err := client.ListAccounts(cmd.Context(), &api.ListAccountsRequest{})
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}

		if len(accounts) == 0 {
			fmt.Println("No budget accounts found.")
			return nil
		}

		// Create tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Print header
		fmt.Fprintln(w, "ACCOUNT\tNAME\tLIMIT\tUSED\tHELD\tAVAILABLE\tSTATUS\tINCREMENTAL")

		for _, account := range accounts {
			fmt.Fprintf(w, "%s\t%s\t$%.2f\t$%.2f\t$%.2f\t$%.2f\t%s\t%t\n",
				account.SlurmAccount,
				account.Name,
				account.BudgetLimit,
				account.BudgetUsed,
				account.BudgetHeld,
				account.BudgetAvailable(),
				account.Status,
				account.HasIncrementalBudget,
			)
		}

		return nil
	},
}

var (
	createAccountName        string
	createAccountAccount     string
	createAccountDescription string
	createAccountBudget      float64
	createAccountStart       string
	createAccountEnd         string
	createIncremental        bool
	createTotalBudget        float64
	createAllocationAmount   float64
	createAllocationFreq     string
)

var accountCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new budget account",
	Long: `Create a new budget account with optional incremental allocation schedule.

Examples:
  # Create simple account
  asbb account create --name="Research" --account=proj001 --budget=1000 --start=2025-01-01 --end=2025-12-31

  # Create account with monthly incremental allocation
  asbb account create --name="Research" --account=proj001 --incremental --total-budget=1200 --allocation-amount=100 --allocation-frequency=monthly --start=2025-01-01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		// Parse dates
		startDate, err := time.Parse("2006-01-02", createAccountStart)
		if err != nil {
			return fmt.Errorf("invalid start date format (use YYYY-MM-DD): %w", err)
		}

		endDate, err := time.Parse("2006-01-02", createAccountEnd)
		if err != nil {
			return fmt.Errorf("invalid end date format (use YYYY-MM-DD): %w", err)
		}

		req := &api.CreateAccountRequest{
			SlurmAccount:         createAccountAccount,
			Name:                 createAccountName,
			Description:          createAccountDescription,
			BudgetLimit:          createAccountBudget,
			StartDate:            startDate,
			EndDate:              endDate,
			HasIncrementalBudget: createIncremental,
		}

		// Add allocation schedule if incremental
		if createIncremental {
			if createTotalBudget <= 0 || createAllocationAmount <= 0 || createAllocationFreq == "" {
				return fmt.Errorf("incremental budget requires --total-budget, --allocation-amount, and --allocation-frequency")
			}

			req.AllocationSchedule = &api.CreateAllocationScheduleRequest{
				TotalBudget:         createTotalBudget,
				AllocationAmount:    createAllocationAmount,
				AllocationFrequency: createAllocationFreq,
				StartDate:           startDate,
				AutoAllocate:        true,
			}

			// For incremental budgets, start with 0 initial limit
			req.BudgetLimit = 0
		}

		account, err := client.CreateAccount(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}

		fmt.Printf("✅ Budget account created successfully!\n")
		fmt.Printf("Account: %s\n", account.SlurmAccount)
		fmt.Printf("Name: %s\n", account.Name)
		fmt.Printf("Budget Limit: $%.2f\n", account.BudgetLimit)
		if account.HasIncrementalBudget {
			fmt.Printf("Incremental Budget: $%.2f total, $%.2f per %s\n",
				createTotalBudget, createAllocationAmount, createAllocationFreq)
		}
		fmt.Printf("Period: %s to %s\n", account.StartDate.Format("2006-01-02"), account.EndDate.Format("2006-01-02"))
		fmt.Printf("Status: %s\n", account.Status)

		return nil
	},
}

var accountShowCmd = &cobra.Command{
	Use:   "show <account>",
	Short: "Show detailed account information",
	Long:  "Show detailed information about a budget account including allocation schedule if applicable.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		account, err := client.GetAccount(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}

		fmt.Printf("Account Details:\n")
		fmt.Printf("================\n")
		fmt.Printf("SLURM Account: %s\n", account.SlurmAccount)
		fmt.Printf("Name: %s\n", account.Name)
		if account.Description != "" {
			fmt.Printf("Description: %s\n", account.Description)
		}
		fmt.Printf("\nBudget Information:\n")
		fmt.Printf("Limit: $%.2f\n", account.BudgetLimit)
		fmt.Printf("Used: $%.2f\n", account.BudgetUsed)
		fmt.Printf("Held: $%.2f\n", account.BudgetHeld)
		fmt.Printf("Available: $%.2f\n", account.BudgetAvailable())
		fmt.Printf("\nAccount Status: %s\n", account.Status)
		fmt.Printf("Period: %s to %s\n", account.StartDate.Format("2006-01-02"), account.EndDate.Format("2006-01-02"))

		if account.HasIncrementalBudget {
			fmt.Printf("\nIncremental Budget:\n")
			fmt.Printf("Total Allocated: $%.2f\n", account.TotalAllocated)
			if account.NextAllocationDate != nil {
				fmt.Printf("Next Allocation: %s\n", account.NextAllocationDate.Format("2006-01-02 15:04:05"))
			}
		}

		fmt.Printf("\nCreated: %s\n", account.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated: %s\n", account.UpdatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

func init() {
	// Account list command
	accountCmd.AddCommand(accountListCmd)

	// Account create command
	accountCreateCmd.Flags().StringVar(&createAccountName, "name", "", "Account name (required)")
	accountCreateCmd.Flags().StringVar(&createAccountAccount, "account", "", "SLURM account name (required)")
	accountCreateCmd.Flags().StringVar(&createAccountDescription, "description", "", "Account description")
	accountCreateCmd.Flags().Float64Var(&createAccountBudget, "budget", 0, "Initial budget limit (required for non-incremental)")
	accountCreateCmd.Flags().StringVar(&createAccountStart, "start", "", "Start date (YYYY-MM-DD, required)")
	accountCreateCmd.Flags().StringVar(&createAccountEnd, "end", "", "End date (YYYY-MM-DD, required)")
	accountCreateCmd.Flags().BoolVar(&createIncremental, "incremental", false, "Enable incremental budget allocation")
	accountCreateCmd.Flags().Float64Var(&createTotalBudget, "total-budget", 0, "Total budget for incremental allocation")
	accountCreateCmd.Flags().Float64Var(&createAllocationAmount, "allocation-amount", 0, "Amount per allocation")
	accountCreateCmd.Flags().StringVar(&createAllocationFreq, "allocation-frequency", "", "Allocation frequency (daily, weekly, monthly, quarterly, yearly)")

	if err := accountCreateCmd.MarkFlagRequired("name"); err != nil {
		panic(err) // This should never happen during initialization
	}
	if err := accountCreateCmd.MarkFlagRequired("account"); err != nil {
		panic(err) // This should never happen during initialization
	}
	if err := accountCreateCmd.MarkFlagRequired("start"); err != nil {
		panic(err) // This should never happen during initialization
	}
	if err := accountCreateCmd.MarkFlagRequired("end"); err != nil {
		panic(err) // This should never happen during initialization
	}

	accountCmd.AddCommand(accountCreateCmd)
	accountCmd.AddCommand(accountShowCmd)
}

// getAPIClient creates an API client - placeholder implementation
func getAPIClient() (*api.Client, error) {
	// TODO: Implement API client creation based on configuration
	return nil, fmt.Errorf("API client not implemented yet")
}
