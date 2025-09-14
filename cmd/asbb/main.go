// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

var (
	configPath string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "asbb",
	Short: "AWS SLURM Bursting Budget Management",
	Long: `AWS SLURM Bursting Budget (asbb) provides budget management and enforcement
for HPC clusters that burst workloads to AWS.

This tool manages budget accounts, monitors usage, and enforces budget limits
at job submission time through SLURM integration.`,
	Version: version.String(),
}

func main() {
	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add command groups
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(allocationsCmd)
	rootCmd.AddCommand(usageCmd)
	rootCmd.AddCommand(transactionCmd)
	rootCmd.AddCommand(reconcileCmd)
	rootCmd.AddCommand(recoverCmd)
	rootCmd.AddCommand(forecastCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(databaseCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		buildInfo := version.GetBuildInfo()
		fmt.Printf("Version: %s\n", buildInfo.Version)
		fmt.Printf("Git Commit: %s\n", buildInfo.GitCommit)
		fmt.Printf("Build Time: %s\n", buildInfo.BuildTime)
		fmt.Printf("Go Version: %s\n", buildInfo.GoVersion)
		fmt.Printf("OS/Arch: %s/%s\n", buildInfo.OS, buildInfo.Arch)
	},
}
