// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/advisor"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/asbx"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/budget"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/config"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/database"
	"github.com/scttfrdmn/aws-slurm-burst-budget/internal/pricing"
	"github.com/scttfrdmn/aws-slurm-burst-budget/pkg/version"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Setup logging
	setupLogging(&cfg.Logging)

	log.Info().
		Str("version", version.Version).
		Str("git_commit", version.GitCommit).
		Str("build_time", version.BuildTime).
		Msg("Starting AWS SLURM Bursting Budget Service")

	// Connect to database
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection")
		}
	}()

	// Run migrations if enabled
	if cfg.Database.AutoMigrate {
		log.Info().Msg("Running database migrations")
		if err := db.Migrate(); err != nil {
			log.Fatal().Err(err).Msg("Failed to run database migrations")
		}
	}

	// Initialize advisor client
	advisorClient := advisor.NewClient(&cfg.Advisor)

	// Initialize AWS pricing for fleet/resume-shaped admission (issue #6). Falls
	// back to a coarse static pricer if AWS is unreachable so admission still
	// returns a (less precise) verdict rather than failing outright.
	var pricer pricing.Pricer
	if tp, perr := pricing.NewTruffle(context.Background()); perr != nil {
		log.Warn().Err(perr).Msg("AWS pricing unavailable, using static fallback rates for fleet admission")
		pricer = pricing.Static{OnDemand: 1.0, SpotDiscount: 0.7}
	} else {
		pricer = tp
	}

	// Initialize budget service
	budgetService := budget.NewService(db, advisorClient, pricer, &cfg.Budget)

	// ASBX integration service (issue #7/#9): closes resume-time holds against
	// actual cost. Mapped from the optional [integration] config block.
	asbxIntegration := asbx.NewIntegrationService(budgetService, &asbx.IntegrationConfig{
		Enabled:               cfg.Integration.ASBXEnabled,
		ASBXEndpoint:          cfg.Integration.ASBXEndpoint,
		AutoReconcile:         true,
		UpdateCostModel:       cfg.Integration.ASBXEnabled,
		ReconciliationTimeout: cfg.Budget.ReconciliationTimeout,
		MaxRetries:            cfg.Integration.RetryAttempts,
	})

	// Setup HTTP server
	router := mux.NewRouter()
	setupRoutes(router, budgetService, asbxIntegration, cfg)

	server := &http.Server{
		Addr:         cfg.Service.ListenAddr,
		Handler:      router,
		ReadTimeout:  cfg.Service.ReadTimeout,
		WriteTimeout: cfg.Service.WriteTimeout,
	}

	// Start server in background
	go func() {
		log.Info().Str("addr", cfg.Service.ListenAddr).Msg("Starting HTTP server")

		var err error
		if cfg.Service.TLSEnabled {
			err = server.ListenAndServeTLS(cfg.Service.TLSCertFile, cfg.Service.TLSKeyFile)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Start background recovery process
	if cfg.Budget.AutoRecoveryEnabled {
		go func() {
			ticker := time.NewTicker(cfg.Budget.RecoveryCheckInterval)
			defer ticker.Stop()

			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := budgetService.RecoverOrphanedTransactions(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to recover orphaned transactions")
				}
				cancel()
			}
		}()
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Service.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server shutdown failed")
	} else {
		log.Info().Msg("Server shutdown complete")
	}
}

func setupLogging(cfg *config.LoggingConfig) {
	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure output format
	if cfg.Format == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	// Configure sampling if enabled
	if cfg.Sampling.Initial > 0 {
		// Use configured sampling rate (already uint32, no conversion needed)
		log.Logger = log.Sample(&zerolog.BasicSampler{N: cfg.Sampling.Initial})
	}
}

func setupRoutes(router *mux.Router, service *budget.Service, asbxIntegration *asbx.IntegrationService, cfg *config.Config) {
	// Setup CORS if enabled
	if cfg.Service.CORSEnabled {
		router.Use(corsMiddleware(cfg.Service.CORSOrigins))
	}

	// Add request logging middleware
	router.Use(loggingMiddleware)

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Budget operations
	api.HandleFunc("/budget/check", handleBudgetCheck(service)).Methods("POST")
	api.HandleFunc("/budget/admit", handleFleetAdmission(service)).Methods("POST")
	api.HandleFunc("/budget/reconcile", handleJobReconcile(service)).Methods("POST")

	// Account management
	api.HandleFunc("/accounts", handleListAccounts(service)).Methods("GET")
	api.HandleFunc("/accounts", handleCreateAccount(service)).Methods("POST")
	api.HandleFunc("/accounts/{account}", handleGetAccount(service)).Methods("GET")
	api.HandleFunc("/accounts/{account}", handleUpdateAccount(service)).Methods("PUT")
	api.HandleFunc("/accounts/{account}", handleDeleteAccount(service)).Methods("DELETE")

	// Transaction management
	api.HandleFunc("/transactions", handleListTransactions(service)).Methods("GET")

	// ASBX Integration endpoints
	api.HandleFunc("/asbx/reconcile", handleASBXReconciliation(asbxIntegration)).Methods("POST")
	api.HandleFunc("/asbx/epilog", handleASBXEpilog(asbxIntegration)).Methods("POST")
	api.HandleFunc("/asbx/status", handleASBXStatus(asbxIntegration)).Methods("GET")

	// ASBA Integration endpoints (Issues #2 and #3)
	api.HandleFunc("/asba/budget-status", handleASBABudgetStatus(service)).Methods("POST")
	api.HandleFunc("/asba/affordability-check", handleASBAAffordabilityCheck(service)).Methods("POST")
	api.HandleFunc("/asba/grant-timeline", handleASBAGrantTimeline(service)).Methods("POST")
	api.HandleFunc("/asba/burst-decision", handleASBABurstDecision(service)).Methods("POST")

	// Health and metrics
	router.HandleFunc("/health", handleHealth(service)).Methods("GET")
	router.HandleFunc("/metrics", handleMetrics()).Methods("GET")

	// Version information
	router.HandleFunc("/version", handleVersion()).Methods("GET")
}

func corsMiddleware(origins []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Simple CORS - in production, implement proper origin checking
			if len(origins) == 1 && origins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowedOrigin := range origins {
					if origin == allowedOrigin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		log.Info().
			Str("method", r.Method).
			Str("uri", r.RequestURI).
			Int("status", lrw.statusCode).
			Dur("duration", time.Since(start)).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("HTTP request")
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
