-- Copyright 2025 Scott Friedman. All rights reserved.
-- Rollback initial schema for AWS SLURM Bursting Budget

-- Drop triggers first
DROP TRIGGER IF EXISTS budget_transactions_balance_update ON budget_transactions;
DROP TRIGGER IF EXISTS budget_partition_limits_updated_at ON budget_partition_limits;
DROP TRIGGER IF EXISTS budget_accounts_updated_at ON budget_accounts;

-- Drop functions
DROP FUNCTION IF EXISTS update_account_balance();
DROP FUNCTION IF EXISTS get_account_budget_summary(BIGINT);
DROP FUNCTION IF EXISTS update_updated_at();

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS job_submissions;
DROP TABLE IF EXISTS budget_transactions;
DROP TABLE IF EXISTS budget_partition_limits;
DROP TABLE IF EXISTS budget_accounts;