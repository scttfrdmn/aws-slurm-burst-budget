-- migrations/001_initial_schema.up.sql
-- AWS SLURM Bursting Budget Database Schema

-- Budget accounts table - maps to SLURM accounts
CREATE TABLE budget_accounts (
    id SERIAL PRIMARY KEY,
    slurm_account VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    budget_limit DECIMAL(12,2) NOT NULL CHECK (budget_limit >= 0),
    budget_used DECIMAL(12,2) DEFAULT 0 CHECK (budget_used >= 0),
    budget_held DECIMAL(12,2) DEFAULT 0 CHECK (budget_held >= 0),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'expired')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure start_date is before end_date
    CONSTRAINT valid_date_range CHECK (start_date < end_date),
    
    -- Ensure used + held doesn't exceed limit (with small tolerance for race conditions)
    CONSTRAINT budget_not_exceeded CHECK (budget_used + budget_held <= budget_limit * 1.01)
);

-- Index for fast lookups by SLURM account
CREATE INDEX idx_budget_accounts_slurm_account ON budget_accounts(slurm_account);
CREATE INDEX idx_budget_accounts_status ON budget_accounts(status);
CREATE INDEX idx_budget_accounts_dates ON budget_accounts(start_date, end_date);

-- Partition-specific budget limits (optional overrides)
CREATE TABLE budget_partition_limits (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    partition_name VARCHAR(255) NOT NULL,
    partition_limit DECIMAL(12,2) NOT NULL CHECK (partition_limit >= 0),
    partition_used DECIMAL(12,2) DEFAULT 0 CHECK (partition_used >= 0),
    partition_held DECIMAL(12,2) DEFAULT 0 CHECK (partition_held >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(account_id, partition_name),
    
    -- Ensure partition used + held doesn't exceed partition limit
    CONSTRAINT partition_budget_not_exceeded CHECK (partition_used + partition_held <= partition_limit * 1.01)
);

CREATE INDEX idx_budget_partition_limits_account ON budget_partition_limits(account_id);
CREATE INDEX idx_budget_partition_limits_partition ON budget_partition_limits(partition_name);

-- Transaction log for all budget operations
CREATE TABLE budget_transactions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    partition_limit_id INTEGER REFERENCES budget_partition_limits(id) ON DELETE SET NULL,
    slurm_job_id INTEGER,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('hold', 'charge', 'refund', 'reconcile', 'adjustment')),
    amount DECIMAL(12,2) NOT NULL,
    partition_name VARCHAR(255),
    estimated_cost DECIMAL(12,2),
    actual_cost DECIMAL(12,2),
    description TEXT,
    
    -- Job and cost estimation details
    job_details JSONB,  -- stores job requirements (nodes, cpus, wall_time, etc.)
    cost_breakdown JSONB,  -- stores detailed cost analysis from advisor
    
    -- AWS-specific information
    aws_instance_info JSONB,  -- instance types, spot prices, regions, etc.
    advisor_recommendation JSONB,  -- full advisor response for audit
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reconciled_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),  -- username or service that created transaction
    
    -- Parent transaction reference for reconciliation
    parent_transaction_id INTEGER REFERENCES budget_transactions(id)
);

-- Indexes for efficient queries
CREATE INDEX idx_budget_transactions_account ON budget_transactions(account_id);
CREATE INDEX idx_budget_transactions_job ON budget_transactions(slurm_job_id);
CREATE INDEX idx_budget_transactions_type ON budget_transactions(transaction_type);
CREATE INDEX idx_budget_transactions_created ON budget_transactions(created_at);
CREATE INDEX idx_budget_transactions_reconciled ON budget_transactions(reconciled_at);
CREATE INDEX idx_budget_transactions_parent ON budget_transactions(parent_transaction_id);

-- Composite index for finding unreconciled holds
CREATE INDEX idx_budget_transactions_unreconciled ON budget_transactions(account_id, transaction_type, reconciled_at)
WHERE transaction_type = 'hold' AND reconciled_at IS NULL;

-- System configuration table
CREATE TABLE budget_config (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default configuration values
INSERT INTO budget_config (key, value, description) VALUES
('default_hold_percentage', '1.2', 'Default percentage buffer for cost holds (1.2 = 20% buffer)'),
('reconciliation_timeout_hours', '24', 'Hours to wait before marking unreconciled transactions as stale'),
('max_hold_duration_hours', '168', 'Maximum hours a hold can remain active (168 = 1 week)'),
('cost_estimation_timeout_seconds', '30', 'Timeout for advisor cost estimation calls'),
('enable_partition_limits', 'false', 'Whether to enforce partition-specific budget limits');

-- Audit log for budget account changes
CREATE TABLE budget_audit_log (
    id SERIAL PRIMARY KEY,
    account_id INTEGER REFERENCES budget_accounts(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    changed_by VARCHAR(255) NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reason TEXT
);

CREATE INDEX idx_budget_audit_log_account ON budget_audit_log(account_id);
CREATE INDEX idx_budget_audit_log_changed_at ON budget_audit_log(changed_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_budget_accounts_updated_at BEFORE UPDATE ON budget_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_budget_partition_limits_updated_at BEFORE UPDATE ON budget_partition_limits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Views for common queries

-- Active accounts with available budget
CREATE VIEW budget_accounts_active AS
SELECT 
    ba.*,
    (ba.budget_limit - ba.budget_used - ba.budget_held) AS available_budget,
    ROUND((ba.budget_used / ba.budget_limit * 100), 2) AS usage_percentage
FROM budget_accounts ba
WHERE ba.status = 'active' 
AND ba.start_date <= CURRENT_DATE 
AND ba.end_date >= CURRENT_DATE;

-- Account usage summary with transaction counts
CREATE VIEW budget_usage_summary AS
SELECT 
    ba.slurm_account,
    ba.name,
    ba.budget_limit,
    ba.budget_used,
    ba.budget_held,
    (ba.budget_limit - ba.budget_used - ba.budget_held) AS available_budget,
    COUNT(bt.id) AS total_transactions,
    COUNT(CASE WHEN bt.transaction_type = 'hold' AND bt.reconciled_at IS NULL THEN 1 END) AS active_holds,
    MAX(bt.created_at) AS last_transaction_at
FROM budget_accounts ba
LEFT JOIN budget_transactions bt ON ba.id = bt.account_id
GROUP BY ba.id, ba.slurm_account, ba.name, ba.budget_limit, ba.budget_used, ba.budget_held;

-- Orphaned holds (holds without corresponding charge/refund after timeout)
CREATE VIEW budget_orphaned_holds AS
SELECT 
    bt.*,
    ba.slurm_account,
    EXTRACT(EPOCH FROM (NOW() - bt.created_at))/3600 AS hours_since_hold
FROM budget_transactions bt
JOIN budget_accounts ba ON bt.account_id = ba.id
WHERE bt.transaction_type = 'hold' 
AND bt.reconciled_at IS NULL
AND bt.created_at < NOW() - INTERVAL '24 hours';

-- Daily usage aggregation for reporting
CREATE VIEW budget_daily_usage AS
SELECT 
    ba.slurm_account,
    DATE(bt.created_at) as usage_date,
    SUM(CASE WHEN bt.transaction_type = 'charge' THEN bt.amount ELSE 0 END) AS daily_charges,
    COUNT(CASE WHEN bt.transaction_type = 'hold' THEN 1 END) AS daily_jobs,
    COUNT(DISTINCT bt.partition_name) AS partitions_used
FROM budget_accounts ba
JOIN budget_transactions bt ON ba.id = bt.account_id
WHERE bt.created_at >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY ba.slurm_account, DATE(bt.created_at)
ORDER BY ba.slurm_account, usage_date DESC;