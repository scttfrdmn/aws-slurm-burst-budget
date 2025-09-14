-- Copyright 2025 Scott Friedman. All rights reserved.
-- Initial schema for AWS SLURM Bursting Budget

-- Budget accounts table
CREATE TABLE budget_accounts (
    id BIGSERIAL PRIMARY KEY,
    slurm_account VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    budget_limit DECIMAL(12,2) NOT NULL CHECK (budget_limit >= 0),
    budget_used DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (budget_used >= 0),
    budget_held DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (budget_held >= 0),
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended', 'expired')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT budget_accounts_date_check CHECK (end_date > start_date)
);

-- Partition-specific budget limits
CREATE TABLE budget_partition_limits (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    partition VARCHAR(64) NOT NULL,
    limit_amount DECIMAL(12,2) NOT NULL CHECK (limit_amount >= 0),
    used_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (used_amount >= 0),
    held_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 CHECK (held_amount >= 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(account_id, partition)
);

-- Budget transactions table
CREATE TABLE budget_transactions (
    id BIGSERIAL PRIMARY KEY,
    transaction_id VARCHAR(128) NOT NULL UNIQUE,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    job_id VARCHAR(128),
    type VARCHAR(32) NOT NULL CHECK (type IN ('hold', 'charge', 'refund', 'adjustment')),
    amount DECIMAL(12,2) NOT NULL,
    description TEXT NOT NULL,
    metadata JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'failed', 'cancelled')),
    parent_transaction_id VARCHAR(128) REFERENCES budget_transactions(transaction_id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Job submissions tracking (for reconciliation)
CREATE TABLE job_submissions (
    id BIGSERIAL PRIMARY KEY,
    job_id VARCHAR(128) NOT NULL UNIQUE,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    partition VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    estimated_cost DECIMAL(12,2) NOT NULL,
    hold_amount DECIMAL(12,2) NOT NULL,
    actual_cost DECIMAL(12,2),
    hold_transaction_id VARCHAR(128) NOT NULL REFERENCES budget_transactions(transaction_id),
    charge_transaction_id VARCHAR(128) REFERENCES budget_transactions(transaction_id),
    job_metadata JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted', 'running', 'completed', 'failed', 'cancelled')),
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    reconciled_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX idx_budget_accounts_slurm_account ON budget_accounts(slurm_account);
CREATE INDEX idx_budget_accounts_status ON budget_accounts(status);
CREATE INDEX idx_budget_accounts_active_period ON budget_accounts(start_date, end_date) WHERE status = 'active';

CREATE INDEX idx_budget_partition_limits_account_partition ON budget_partition_limits(account_id, partition);

CREATE INDEX idx_budget_transactions_transaction_id ON budget_transactions(transaction_id);
CREATE INDEX idx_budget_transactions_account_id ON budget_transactions(account_id);
CREATE INDEX idx_budget_transactions_job_id ON budget_transactions(job_id) WHERE job_id IS NOT NULL;
CREATE INDEX idx_budget_transactions_type ON budget_transactions(type);
CREATE INDEX idx_budget_transactions_status ON budget_transactions(status);
CREATE INDEX idx_budget_transactions_created_at ON budget_transactions(created_at);
CREATE INDEX idx_budget_transactions_parent ON budget_transactions(parent_transaction_id) WHERE parent_transaction_id IS NOT NULL;

CREATE INDEX idx_job_submissions_job_id ON job_submissions(job_id);
CREATE INDEX idx_job_submissions_account_id ON job_submissions(account_id);
CREATE INDEX idx_job_submissions_status ON job_submissions(status);
CREATE INDEX idx_job_submissions_partition ON job_submissions(partition);
CREATE INDEX idx_job_submissions_user_id ON job_submissions(user_id);
CREATE INDEX idx_job_submissions_submitted_at ON job_submissions(submitted_at);
CREATE INDEX idx_job_submissions_reconciliation ON job_submissions(status, reconciled_at) WHERE status = 'completed' AND reconciled_at IS NULL;

-- Trigger to update budget_accounts.updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER budget_accounts_updated_at
    BEFORE UPDATE ON budget_accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER budget_partition_limits_updated_at
    BEFORE UPDATE ON budget_partition_limits
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Function to get account budget summary
CREATE OR REPLACE FUNCTION get_account_budget_summary(p_account_id BIGINT)
RETURNS TABLE(
    account_id BIGINT,
    budget_limit DECIMAL(12,2),
    budget_used DECIMAL(12,2),
    budget_held DECIMAL(12,2),
    budget_available DECIMAL(12,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ba.id,
        ba.budget_limit,
        ba.budget_used,
        ba.budget_held,
        (ba.budget_limit - ba.budget_used - ba.budget_held) AS budget_available
    FROM budget_accounts ba
    WHERE ba.id = p_account_id;
END;
$$ LANGUAGE plpgsql;

-- Function to update account balances after transaction completion
CREATE OR REPLACE FUNCTION update_account_balance()
RETURNS TRIGGER AS $$
DECLARE
    account_rec budget_accounts%ROWTYPE;
BEGIN
    -- Only process completed transactions
    IF NEW.status = 'completed' AND (OLD.status IS NULL OR OLD.status != 'completed') THEN
        SELECT * INTO account_rec FROM budget_accounts WHERE id = NEW.account_id;

        IF NEW.type = 'hold' THEN
            -- Increase held amount
            UPDATE budget_accounts
            SET budget_held = budget_held + NEW.amount,
                updated_at = NOW()
            WHERE id = NEW.account_id;

        ELSIF NEW.type = 'charge' THEN
            -- Increase used amount, decrease held amount if this was from a hold
            IF NEW.parent_transaction_id IS NOT NULL THEN
                -- This is a charge from a previous hold
                UPDATE budget_accounts
                SET budget_used = budget_used + NEW.amount,
                    budget_held = GREATEST(0, budget_held - NEW.amount),
                    updated_at = NOW()
                WHERE id = NEW.account_id;
            ELSE
                -- Direct charge
                UPDATE budget_accounts
                SET budget_used = budget_used + NEW.amount,
                    updated_at = NOW()
                WHERE id = NEW.account_id;
            END IF;

        ELSIF NEW.type = 'refund' THEN
            -- Decrease used amount or held amount
            IF NEW.parent_transaction_id IS NOT NULL THEN
                -- Get the parent transaction to determine what to refund
                DECLARE
                    parent_type VARCHAR(32);
                BEGIN
                    SELECT type INTO parent_type
                    FROM budget_transactions
                    WHERE transaction_id = NEW.parent_transaction_id;

                    IF parent_type = 'charge' THEN
                        UPDATE budget_accounts
                        SET budget_used = GREATEST(0, budget_used - NEW.amount),
                            updated_at = NOW()
                        WHERE id = NEW.account_id;
                    ELSIF parent_type = 'hold' THEN
                        UPDATE budget_accounts
                        SET budget_held = GREATEST(0, budget_held - NEW.amount),
                            updated_at = NOW()
                        WHERE id = NEW.account_id;
                    END IF;
                END;
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER budget_transactions_balance_update
    AFTER INSERT OR UPDATE ON budget_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_account_balance();