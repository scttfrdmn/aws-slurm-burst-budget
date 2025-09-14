-- Copyright 2025 Scott Friedman. All rights reserved.
-- Rollback grant management and burn rate analytics

-- Drop functions
DROP FUNCTION IF EXISTS process_daily_burn_rates(DATE);
DROP FUNCTION IF EXISTS check_burn_rate_alerts(BIGINT);
DROP FUNCTION IF EXISTS get_burn_rate_analysis(BIGINT, DATE, DATE);
DROP FUNCTION IF EXISTS update_burn_rate_metrics(BIGINT, DATE);
DROP FUNCTION IF EXISTS calculate_daily_expected_burn_rate(BIGINT, BIGINT);

-- Drop triggers
DROP TRIGGER IF EXISTS grant_budget_periods_updated_at ON grant_budget_periods;
DROP TRIGGER IF EXISTS grant_accounts_updated_at ON grant_accounts;

-- Drop views
DROP VIEW IF EXISTS budget_burn_rate_analysis;

-- Remove columns from budget_accounts
ALTER TABLE budget_accounts
DROP COLUMN IF EXISTS grant_id,
DROP COLUMN IF EXISTS grant_budget_period_id,
DROP COLUMN IF EXISTS is_grant_funded;

-- Drop tables in reverse order
DROP TABLE IF EXISTS budget_alerts;
DROP TABLE IF EXISTS budget_burn_rates;
DROP TABLE IF EXISTS grant_budget_periods;
DROP TABLE IF EXISTS grant_accounts;