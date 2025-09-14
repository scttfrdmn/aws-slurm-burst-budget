-- Copyright 2025 Scott Friedman. All rights reserved.
-- Rollback incremental budget allocation feature

-- Drop functions
DROP FUNCTION IF EXISTS get_allocation_schedule_summary(BIGINT);
DROP FUNCTION IF EXISTS process_pending_allocations();
DROP FUNCTION IF EXISTS calculate_next_allocation_date(TIMESTAMP WITH TIME ZONE, VARCHAR(32));

-- Drop triggers
DROP TRIGGER IF EXISTS budget_allocation_schedules_updated_at ON budget_allocation_schedules;

-- Remove columns from budget_accounts
ALTER TABLE budget_accounts
DROP COLUMN IF EXISTS has_incremental_budget,
DROP COLUMN IF EXISTS next_allocation_date,
DROP COLUMN IF EXISTS total_allocated;

-- Drop tables in reverse order
DROP TABLE IF EXISTS budget_allocations;
DROP TABLE IF EXISTS budget_allocation_schedules;