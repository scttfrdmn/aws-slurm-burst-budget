-- Copyright 2025 Scott Friedman. All rights reserved.
-- Add incremental budget allocation feature

-- Budget allocation schedules table
CREATE TABLE budget_allocation_schedules (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    total_budget DECIMAL(12,2) NOT NULL CHECK (total_budget > 0),
    allocation_amount DECIMAL(12,2) NOT NULL CHECK (allocation_amount > 0),
    allocation_frequency VARCHAR(32) NOT NULL CHECK (allocation_frequency IN ('daily', 'weekly', 'monthly', 'quarterly', 'yearly')),
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE,
    next_allocation_date TIMESTAMP WITH TIME ZONE NOT NULL,
    allocated_to_date DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (allocated_to_date >= 0),
    remaining_budget DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (remaining_budget >= 0),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'cancelled')),
    auto_allocate BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT budget_allocation_schedules_budget_check CHECK (allocated_to_date <= total_budget)
);

-- Budget allocation history table
CREATE TABLE budget_allocations (
    id BIGSERIAL PRIMARY KEY,
    schedule_id BIGINT NOT NULL REFERENCES budget_allocation_schedules(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    allocation_amount DECIMAL(12,2) NOT NULL CHECK (allocation_amount > 0),
    allocated_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    transaction_id VARCHAR(128) REFERENCES budget_transactions(transaction_id),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Add incremental budget fields to budget_accounts
ALTER TABLE budget_accounts
ADD COLUMN has_incremental_budget BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN next_allocation_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN total_allocated DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (total_allocated >= 0);

-- Indexes for performance
CREATE INDEX idx_budget_allocation_schedules_account_id ON budget_allocation_schedules(account_id);
CREATE INDEX idx_budget_allocation_schedules_status ON budget_allocation_schedules(status);
CREATE INDEX idx_budget_allocation_schedules_next_allocation ON budget_allocation_schedules(next_allocation_date) WHERE status = 'active';
CREATE INDEX idx_budget_allocation_schedules_frequency ON budget_allocation_schedules(allocation_frequency);

CREATE INDEX idx_budget_allocations_schedule_id ON budget_allocations(schedule_id);
CREATE INDEX idx_budget_allocations_account_id ON budget_allocations(account_id);
CREATE INDEX idx_budget_allocations_allocated_date ON budget_allocations(allocated_date);

-- Trigger to update updated_at timestamp
CREATE TRIGGER budget_allocation_schedules_updated_at
    BEFORE UPDATE ON budget_allocation_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Function to calculate next allocation date
CREATE OR REPLACE FUNCTION calculate_next_allocation_date(
    p_current_date TIMESTAMP WITH TIME ZONE,
    p_frequency VARCHAR(32)
) RETURNS TIMESTAMP WITH TIME ZONE AS $$
BEGIN
    CASE p_frequency
        WHEN 'daily' THEN
            RETURN p_current_date + INTERVAL '1 day';
        WHEN 'weekly' THEN
            RETURN p_current_date + INTERVAL '1 week';
        WHEN 'monthly' THEN
            RETURN p_current_date + INTERVAL '1 month';
        WHEN 'quarterly' THEN
            RETURN p_current_date + INTERVAL '3 months';
        WHEN 'yearly' THEN
            RETURN p_current_date + INTERVAL '1 year';
        ELSE
            RAISE EXCEPTION 'Invalid allocation frequency: %', p_frequency;
    END CASE;
END;
$$ LANGUAGE plpgsql;

-- Function to process pending allocations
CREATE OR REPLACE FUNCTION process_pending_allocations()
RETURNS TABLE(
    schedule_id BIGINT,
    account_id BIGINT,
    allocated_amount DECIMAL(12,2),
    transaction_id VARCHAR(128)
) AS $$
DECLARE
    schedule_rec RECORD;
    txn_id VARCHAR(128);
    allocation_amount DECIMAL(12,2);
BEGIN
    -- Find active schedules that are due for allocation
    FOR schedule_rec IN
        SELECT bas.id, bas.account_id, bas.allocation_amount, bas.allocated_to_date,
               bas.total_budget, bas.allocation_frequency, bas.next_allocation_date
        FROM budget_allocation_schedules bas
        WHERE bas.status = 'active'
          AND bas.auto_allocate = TRUE
          AND bas.next_allocation_date <= NOW()
          AND bas.allocated_to_date < bas.total_budget
    LOOP
        -- Calculate allocation amount (don't exceed total budget)
        allocation_amount := LEAST(schedule_rec.allocation_amount,
                                  schedule_rec.total_budget - schedule_rec.allocated_to_date);

        -- Generate transaction ID
        txn_id := 'alloc_' || schedule_rec.id || '_' || extract(epoch from now())::bigint;

        -- Create budget transaction
        INSERT INTO budget_transactions (
            transaction_id, account_id, type, amount, description, status
        ) VALUES (
            txn_id, schedule_rec.account_id, 'allocation', allocation_amount,
            'Automated budget allocation', 'completed'
        );

        -- Record the allocation
        INSERT INTO budget_allocations (
            schedule_id, account_id, allocation_amount, transaction_id, notes
        ) VALUES (
            schedule_rec.id, schedule_rec.account_id, allocation_amount, txn_id,
            'Automated allocation'
        );

        -- Update allocation schedule
        UPDATE budget_allocation_schedules
        SET allocated_to_date = allocated_to_date + allocation_amount,
            remaining_budget = total_budget - (allocated_to_date + allocation_amount),
            next_allocation_date = CASE
                WHEN (allocated_to_date + allocation_amount) >= total_budget THEN NULL
                ELSE calculate_next_allocation_date(next_allocation_date, allocation_frequency)
            END,
            status = CASE
                WHEN (allocated_to_date + allocation_amount) >= total_budget THEN 'completed'
                ELSE status
            END,
            updated_at = NOW()
        WHERE id = schedule_rec.id;

        -- Update account's budget limit and next allocation date
        UPDATE budget_accounts
        SET budget_limit = budget_limit + allocation_amount,
            total_allocated = total_allocated + allocation_amount,
            next_allocation_date = (
                SELECT next_allocation_date
                FROM budget_allocation_schedules
                WHERE account_id = schedule_rec.account_id
                  AND status = 'active'
                ORDER BY next_allocation_date ASC
                LIMIT 1
            ),
            updated_at = NOW()
        WHERE id = schedule_rec.account_id;

        -- Return result
        RETURN QUERY SELECT schedule_rec.id, schedule_rec.account_id, allocation_amount, txn_id;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Function to get allocation schedule summary
CREATE OR REPLACE FUNCTION get_allocation_schedule_summary(p_account_id BIGINT)
RETURNS TABLE(
    total_budget DECIMAL(12,2),
    allocated_to_date DECIMAL(12,2),
    remaining_budget DECIMAL(12,2),
    next_allocation_date TIMESTAMP WITH TIME ZONE,
    next_allocation_amount DECIMAL(12,2),
    allocation_frequency VARCHAR(32)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        bas.total_budget,
        bas.allocated_to_date,
        bas.remaining_budget,
        bas.next_allocation_date,
        CASE
            WHEN bas.status = 'active' AND bas.remaining_budget > 0 THEN
                LEAST(bas.allocation_amount, bas.remaining_budget)
            ELSE 0.00
        END as next_allocation_amount,
        bas.allocation_frequency
    FROM budget_allocation_schedules bas
    WHERE bas.account_id = p_account_id
      AND bas.status = 'active'
    ORDER BY bas.next_allocation_date ASC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;