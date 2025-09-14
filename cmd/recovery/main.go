// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/advisor"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/database"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

func main() {
	fmt.Printf("AWS SLURM Bursting Budget Recovery Tool %s\n", version.Version)
	fmt.Println("=========================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to database
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize advisor client
	advisorClient := advisor.NewClient(&cfg.Advisor)

	// Initialize budget service
	budgetService := budget.NewService(db, advisorClient, &cfg.Budget)

	// Run recovery operation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Starting recovery operation...")

	if err := budgetService.RecoverOrphanedTransactions(ctx); err != nil {
		log.Fatal().Err(err).Msg("Recovery operation failed")
	}

	fmt.Println("✅ Recovery operation completed successfully!")
	fmt.Println("Check the service logs for details on recovered transactions.")
}