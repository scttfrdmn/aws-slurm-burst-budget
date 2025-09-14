// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/api"
)

var allocationsCmd = &cobra.Command{
	Use:   "allocations",
	Short: "Manage incremental budget allocations",
	Long: `Manage incremental budget allocation schedules and process allocations.

Examples:
  # List allocation schedules
  asbb allocations list

  # Show specific allocation schedule
  asbb allocations show 123

  # Process pending allocations
  asbb allocations process

  # Pause an allocation schedule
  asbb allocations pause 123`,
}

var allocationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List allocation schedules",
	Long:  "List all incremental budget allocation schedules.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		schedules, err := client.ListAllocationSchedules(cmd.Context(), &api.AllocationScheduleRequest{})
		if err != nil {
			return fmt.Errorf("failed to list allocation schedules: %w", err)
		}

		if len(schedules) == 0 {
			fmt.Println("No allocation schedules found.")
			return nil
		}

		// Create tabwriter for aligned output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Print header
		fmt.Fprintln(w, "ID\tACCOUNT\tTOTAL\tALLOCATED\tREMAINING\tFREQUENCY\tNEXT\tSTATUS")

		for _, schedule := range schedules {
			nextDate := "N/A"
			if schedule.Status == "active" {
				nextDate = schedule.NextAllocationDate.Format("2006-01-02")
			}

			fmt.Fprintf(w, "%d\t%d\t$%.2f\t$%.2f\t$%.2f\t%s\t%s\t%s\n",
				schedule.ID,
				schedule.AccountID,
				schedule.TotalBudget,
				schedule.AllocatedToDate,
				schedule.RemainingBudget,
				schedule.AllocationFrequency,
				nextDate,
				schedule.Status,
			)
		}

		return nil
	},
}

var allocationsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show allocation schedule details",
	Long:  "Show detailed information about a specific allocation schedule.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement show allocation schedule
		fmt.Printf("Show allocation schedule %s - Not implemented yet\n", args[0])
		return nil
	},
}

var allocationsProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "Process pending allocations",
	Long:  "Manually trigger processing of pending allocation schedules.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getAPIClient()
		if err != nil {
			return fmt.Errorf("failed to create API client: %w", err)
		}

		result, err := client.ProcessAllocations(cmd.Context(), &api.ProcessAllocationsRequest{})
		if err != nil {
			return fmt.Errorf("failed to process allocations: %w", err)
		}

		fmt.Printf("✅ Allocation processing completed!\n")
		fmt.Printf("Processed: %d allocations\n", result.ProcessedCount)
		fmt.Printf("Total Allocated: $%.2f\n", result.TotalAllocated)

		if len(result.Allocations) > 0 {
			fmt.Printf("\nProcessed Allocations:\n")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer w.Flush()

			fmt.Fprintln(w, "SCHEDULE_ID\tACCOUNT_ID\tAMOUNT\tTRANSACTION_ID")
			for _, alloc := range result.Allocations {
				fmt.Fprintf(w, "%d\t%d\t$%.2f\t%s\n",
					alloc.ScheduleID,
					alloc.AccountID,
					alloc.AllocatedAmount,
					alloc.TransactionID,
				)
			}
		}

		return nil
	},
}

func init() {
	allocationsCmd.AddCommand(allocationsListCmd)
	allocationsCmd.AddCommand(allocationsShowCmd)
	allocationsCmd.AddCommand(allocationsProcessCmd)
}