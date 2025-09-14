-- Copyright 2025 Scott Friedman. All rights reserved.
-- Grant management and burn rate analytics schema

-- Grant accounts for long-term funding tracking
CREATE TABLE grant_accounts (
    id BIGSERIAL PRIMARY KEY,
    grant_number VARCHAR(128) NOT NULL UNIQUE,
    funding_agency VARCHAR(255) NOT NULL,
    agency_program VARCHAR(255),
    principal_investigator VARCHAR(255) NOT NULL,
    co_investigators TEXT[], -- Array of co-PI names
    institution VARCHAR(255) NOT NULL,
    department VARCHAR(255),

    -- Grant period (can span multiple years)
    grant_start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    grant_end_date TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Financial details
    total_award_amount DECIMAL(15,2) NOT NULL CHECK (total_award_amount > 0),
    direct_costs DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    indirect_cost_rate DECIMAL(5,4) DEFAULT 0.30, -- e.g., 0.30 for 30% overhead
    indirect_costs DECIMAL(15,2) GENERATED ALWAYS AS (direct_costs * indirect_cost_rate) STORED,

    -- Budget periods (many grants have annual budget periods)
    budget_period_months INTEGER NOT NULL DEFAULT 12 CHECK (budget_period_months > 0),
    current_budget_period INTEGER NOT NULL DEFAULT 1,

    -- Status and compliance
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active', 'suspended', 'completed', 'cancelled')),
    compliance_requirements JSONB, -- Agency-specific requirements

    -- Metadata and tracking
    federal_award_id VARCHAR(128), -- CFDA number for federal grants
    internal_project_code VARCHAR(64),
    cost_center VARCHAR(64),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT grant_accounts_period_check CHECK (grant_end_date > grant_start_date)
);

-- Grant budget periods for multi-year grants
CREATE TABLE grant_budget_periods (
    id BIGSERIAL PRIMARY KEY,
    grant_id BIGINT NOT NULL REFERENCES grant_accounts(id) ON DELETE CASCADE,
    period_number INTEGER NOT NULL CHECK (period_number > 0),
    period_start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end_date TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Budget for this period
    period_budget_amount DECIMAL(15,2) NOT NULL CHECK (period_budget_amount >= 0),
    period_spent_amount DECIMAL(15,2) NOT NULL DEFAULT 0.00 CHECK (period_spent_amount >= 0),
    period_committed_amount DECIMAL(15,2) NOT NULL DEFAULT 0.00 CHECK (period_committed_amount >= 0),

    -- Burn rate tracking
    expected_burn_rate DECIMAL(12,2), -- Expected spending per day
    actual_burn_rate DECIMAL(12,2),   -- Calculated from actual spending
    burn_rate_variance DECIMAL(8,4),  -- Percentage variance from expected

    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('future', 'active', 'completed', 'extended')),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(grant_id, period_number),
    CONSTRAINT grant_budget_periods_date_check CHECK (period_end_date > period_start_date)
);

-- Link budget accounts to grants
ALTER TABLE budget_accounts
ADD COLUMN grant_id BIGINT REFERENCES grant_accounts(id),
ADD COLUMN grant_budget_period_id BIGINT REFERENCES grant_budget_periods(id),
ADD COLUMN is_grant_funded BOOLEAN NOT NULL DEFAULT FALSE;

-- Burn rate tracking for all budget accounts
CREATE TABLE budget_burn_rates (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    measurement_date DATE NOT NULL,

    -- Daily metrics
    daily_spend_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    daily_expected_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    daily_variance_pct DECIMAL(8,4) GENERATED ALWAYS AS (
        CASE
            WHEN daily_expected_amount > 0 THEN
                ((daily_spend_amount - daily_expected_amount) / daily_expected_amount) * 100
            ELSE 0
        END
    ) STORED,

    -- Rolling averages (7-day, 30-day)
    rolling_7day_avg DECIMAL(12,2),
    rolling_30day_avg DECIMAL(12,2),

    -- Cumulative tracking
    cumulative_spend DECIMAL(15,2) NOT NULL,
    cumulative_expected DECIMAL(15,2) NOT NULL,
    cumulative_variance_pct DECIMAL(8,4) GENERATED ALWAYS AS (
        CASE
            WHEN cumulative_expected > 0 THEN
                ((cumulative_spend - cumulative_expected) / cumulative_expected) * 100
            ELSE 0
        END
    ) STORED,

    -- Forecasting
    projected_end_date TIMESTAMP WITH TIME ZONE,
    projected_depletion_date TIMESTAMP WITH TIME ZONE,
    budget_health_score DECIMAL(5,2) CHECK (budget_health_score >= 0 AND budget_health_score <= 100),

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(account_id, measurement_date)
);

-- Budget alerts and notifications
CREATE TABLE budget_alerts (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES budget_accounts(id) ON DELETE CASCADE,
    grant_id BIGINT REFERENCES grant_accounts(id) ON DELETE CASCADE,

    alert_type VARCHAR(64) NOT NULL CHECK (alert_type IN (
        'burn_rate_high', 'burn_rate_low', 'budget_threshold', 'grant_expiring',
        'period_ending', 'overspend_risk', 'underspend_risk', 'compliance_warning'
    )),

    severity VARCHAR(32) NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'critical')),
    threshold_value DECIMAL(12,2),
    actual_value DECIMAL(12,2),

    message TEXT NOT NULL,
    details JSONB,

    -- Alert management
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    acknowledged_by VARCHAR(255),
    resolved_at TIMESTAMP WITH TIME ZONE,

    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved', 'dismissed'))
);

-- Burn rate analytics view
CREATE VIEW budget_burn_rate_analysis AS
SELECT
    ba.id as account_id,
    ba.slurm_account,
    ba.name as account_name,
    ba.budget_limit,
    ba.budget_used,
    ba.budget_held,
    ba.start_date,
    ba.end_date,

    -- Grant information
    ga.grant_number,
    ga.funding_agency,
    ga.principal_investigator,
    ga.total_award_amount,

    -- Current burn rate metrics
    bbr.daily_spend_amount as current_daily_spend,
    bbr.daily_expected_amount as expected_daily_spend,
    bbr.daily_variance_pct,
    bbr.rolling_7day_avg,
    bbr.rolling_30day_avg,
    bbr.cumulative_variance_pct,
    bbr.budget_health_score,
    bbr.projected_depletion_date,

    -- Time-based calculations
    EXTRACT(EPOCH FROM (ba.end_date - NOW())) / 86400 as days_remaining,
    (ba.budget_limit - ba.budget_used - ba.budget_held) as budget_remaining,

    -- Burn rate status
    CASE
        WHEN bbr.daily_variance_pct > 20 THEN 'OVERSPENDING'
        WHEN bbr.daily_variance_pct < -20 THEN 'UNDERSPENDING'
        WHEN bbr.daily_variance_pct BETWEEN -20 AND 20 THEN 'ON_TRACK'
        ELSE 'UNKNOWN'
    END as burn_rate_status,

    -- Budget health assessment
    CASE
        WHEN bbr.budget_health_score >= 80 THEN 'HEALTHY'
        WHEN bbr.budget_health_score >= 60 THEN 'CONCERN'
        WHEN bbr.budget_health_score >= 40 THEN 'WARNING'
        ELSE 'CRITICAL'
    END as budget_health_status

FROM budget_accounts ba
LEFT JOIN grant_accounts ga ON ba.grant_id = ga.id
LEFT JOIN budget_burn_rates bbr ON ba.id = bbr.account_id
    AND bbr.measurement_date = CURRENT_DATE;

-- Indexes for performance
CREATE INDEX idx_grant_accounts_grant_number ON grant_accounts(grant_number);
CREATE INDEX idx_grant_accounts_agency ON grant_accounts(funding_agency);
CREATE INDEX idx_grant_accounts_pi ON grant_accounts(principal_investigator);
CREATE INDEX idx_grant_accounts_dates ON grant_accounts(grant_start_date, grant_end_date);
CREATE INDEX idx_grant_accounts_status ON grant_accounts(status);
CREATE INDEX idx_grant_accounts_federal_award ON grant_accounts(federal_award_id) WHERE federal_award_id IS NOT NULL;

CREATE INDEX idx_grant_budget_periods_grant_id ON grant_budget_periods(grant_id);
CREATE INDEX idx_grant_budget_periods_dates ON grant_budget_periods(period_start_date, period_end_date);
CREATE INDEX idx_grant_budget_periods_status ON grant_budget_periods(status);

CREATE INDEX idx_budget_accounts_grant ON budget_accounts(grant_id) WHERE grant_id IS NOT NULL;
CREATE INDEX idx_budget_accounts_grant_funded ON budget_accounts(is_grant_funded) WHERE is_grant_funded = TRUE;

CREATE INDEX idx_budget_burn_rates_account_date ON budget_burn_rates(account_id, measurement_date);
CREATE INDEX idx_budget_burn_rates_date ON budget_burn_rates(measurement_date);
CREATE INDEX idx_budget_burn_rates_health ON budget_burn_rates(budget_health_score);
CREATE INDEX idx_budget_burn_rates_variance ON budget_burn_rates(daily_variance_pct);

CREATE INDEX idx_budget_alerts_account ON budget_alerts(account_id);
CREATE INDEX idx_budget_alerts_grant ON budget_alerts(grant_id) WHERE grant_id IS NOT NULL;
CREATE INDEX idx_budget_alerts_type ON budget_alerts(alert_type);
CREATE INDEX idx_budget_alerts_severity ON budget_alerts(severity);
CREATE INDEX idx_budget_alerts_status ON budget_alerts(status);
CREATE INDEX idx_budget_alerts_triggered ON budget_alerts(triggered_at);

-- Triggers for automatic updates
CREATE TRIGGER grant_accounts_updated_at
    BEFORE UPDATE ON grant_accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER grant_budget_periods_updated_at
    BEFORE UPDATE ON grant_budget_periods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Function to calculate daily expected burn rate for grants
CREATE OR REPLACE FUNCTION calculate_daily_expected_burn_rate(
    p_grant_id BIGINT,
    p_period_id BIGINT DEFAULT NULL
) RETURNS DECIMAL(12,2) AS $$
DECLARE
    grant_rec grant_accounts%ROWTYPE;
    period_rec grant_budget_periods%ROWTYPE;
    total_days INTEGER;
    daily_rate DECIMAL(12,2);
BEGIN
    -- Get grant information
    SELECT * INTO grant_rec FROM grant_accounts WHERE id = p_grant_id;

    IF NOT FOUND THEN
        RETURN 0.00;
    END IF;

    -- If specific period provided, use period budget
    IF p_period_id IS NOT NULL THEN
        SELECT * INTO period_rec FROM grant_budget_periods WHERE id = p_period_id;
        IF FOUND THEN
            total_days := EXTRACT(EPOCH FROM (period_rec.period_end_date - period_rec.period_start_date)) / 86400;
            daily_rate := period_rec.period_budget_amount / total_days;
            RETURN daily_rate;
        END IF;
    END IF;

    -- Otherwise, use total grant amount over grant period
    total_days := EXTRACT(EPOCH FROM (grant_rec.grant_end_date - grant_rec.grant_start_date)) / 86400;
    daily_rate := grant_rec.total_award_amount / total_days;

    RETURN daily_rate;
END;
$$ LANGUAGE plpgsql;

-- Function to update burn rate metrics
CREATE OR REPLACE FUNCTION update_burn_rate_metrics(p_account_id BIGINT, p_date DATE DEFAULT CURRENT_DATE)
RETURNS VOID AS $$
DECLARE
    account_rec budget_accounts%ROWTYPE;
    daily_spend DECIMAL(12,2);
    expected_daily DECIMAL(12,2);
    cumulative_spend DECIMAL(15,2);
    days_elapsed INTEGER;
    total_days INTEGER;
    health_score DECIMAL(5,2);
BEGIN
    -- Get account information
    SELECT * INTO account_rec FROM budget_accounts WHERE id = p_account_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Calculate daily spend for the date
    SELECT COALESCE(SUM(amount), 0.00) INTO daily_spend
    FROM budget_transactions
    WHERE account_id = p_account_id
      AND type IN ('charge', 'allocation')
      AND DATE(created_at) = p_date
      AND status = 'completed';

    -- Calculate expected daily spend
    total_days := EXTRACT(EPOCH FROM (account_rec.end_date - account_rec.start_date)) / 86400;
    days_elapsed := EXTRACT(EPOCH FROM (p_date - DATE(account_rec.start_date))) / 86400;

    expected_daily := account_rec.budget_limit / total_days;
    cumulative_spend := account_rec.budget_used;

    -- Calculate health score (0-100)
    -- Based on spending trajectory vs budget timeline
    IF total_days > 0 AND days_elapsed > 0 THEN
        DECLARE
            expected_cumulative DECIMAL(15,2);
            variance_ratio DECIMAL(8,4);
        BEGIN
            expected_cumulative := (days_elapsed / total_days) * account_rec.budget_limit;

            IF expected_cumulative > 0 THEN
                variance_ratio := cumulative_spend / expected_cumulative;

                -- Health score calculation:
                -- 100 = perfect on track
                -- 80+ = within 20% of expected
                -- 60+ = within 40% of expected
                -- 40+ = significant variance
                -- <40 = critical variance
                health_score := GREATEST(0, 100 - (ABS(variance_ratio - 1.0) * 100));
            ELSE
                health_score := 100; -- Early in grant period
            END IF;
        END;
    ELSE
        health_score := 100; -- Default for edge cases
    END IF;

    -- Insert or update burn rate record
    INSERT INTO budget_burn_rates (
        account_id, measurement_date, daily_spend_amount, daily_expected_amount,
        cumulative_spend, cumulative_expected, budget_health_score
    ) VALUES (
        p_account_id, p_date, daily_spend, expected_daily,
        cumulative_spend, expected_daily * days_elapsed, health_score
    ) ON CONFLICT (account_id, measurement_date)
    DO UPDATE SET
        daily_spend_amount = EXCLUDED.daily_spend_amount,
        daily_expected_amount = EXCLUDED.daily_expected_amount,
        cumulative_spend = EXCLUDED.cumulative_spend,
        cumulative_expected = EXCLUDED.cumulative_expected,
        budget_health_score = EXCLUDED.budget_health_score;

    -- Update rolling averages
    UPDATE budget_burn_rates
    SET
        rolling_7day_avg = (
            SELECT AVG(daily_spend_amount)
            FROM budget_burn_rates
            WHERE account_id = p_account_id
              AND measurement_date BETWEEN p_date - INTERVAL '6 days' AND p_date
        ),
        rolling_30day_avg = (
            SELECT AVG(daily_spend_amount)
            FROM budget_burn_rates
            WHERE account_id = p_account_id
              AND measurement_date BETWEEN p_date - INTERVAL '29 days' AND p_date
        )
    WHERE account_id = p_account_id AND measurement_date = p_date;

END;
$$ LANGUAGE plpgsql;

-- Function to get burn rate analysis for an account
CREATE OR REPLACE FUNCTION get_burn_rate_analysis(
    p_account_id BIGINT,
    p_start_date DATE DEFAULT CURRENT_DATE - INTERVAL '30 days',
    p_end_date DATE DEFAULT CURRENT_DATE
) RETURNS TABLE (
    measurement_date DATE,
    daily_spend DECIMAL(12,2),
    daily_expected DECIMAL(12,2),
    daily_variance_pct DECIMAL(8,4),
    rolling_7day_avg DECIMAL(12,2),
    rolling_30day_avg DECIMAL(12,2),
    cumulative_spend DECIMAL(15,2),
    cumulative_expected DECIMAL(15,2),
    cumulative_variance_pct DECIMAL(8,4),
    budget_health_score DECIMAL(5,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        bbr.measurement_date,
        bbr.daily_spend_amount,
        bbr.daily_expected_amount,
        bbr.daily_variance_pct,
        bbr.rolling_7day_avg,
        bbr.rolling_30day_avg,
        bbr.cumulative_spend,
        bbr.cumulative_expected,
        bbr.cumulative_variance_pct,
        bbr.budget_health_score
    FROM budget_burn_rates bbr
    WHERE bbr.account_id = p_account_id
      AND bbr.measurement_date BETWEEN p_start_date AND p_end_date
    ORDER BY bbr.measurement_date DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to generate burn rate alerts
CREATE OR REPLACE FUNCTION check_burn_rate_alerts(p_account_id BIGINT DEFAULT NULL)
RETURNS TABLE (
    account_id BIGINT,
    alert_type VARCHAR(64),
    severity VARCHAR(32),
    message TEXT
) AS $$
DECLARE
    account_rec RECORD;
BEGIN
    -- Check all accounts or specific account
    FOR account_rec IN
        SELECT ba.*, bbr.daily_variance_pct, bbr.budget_health_score, bbr.projected_depletion_date
        FROM budget_accounts ba
        LEFT JOIN budget_burn_rates bbr ON ba.id = bbr.account_id
            AND bbr.measurement_date = CURRENT_DATE
        WHERE (p_account_id IS NULL OR ba.id = p_account_id)
          AND ba.status = 'active'
    LOOP
        -- High burn rate alert
        IF account_rec.daily_variance_pct > 50 THEN
            RETURN QUERY SELECT
                account_rec.id,
                'burn_rate_high'::VARCHAR(64),
                'critical'::VARCHAR(32),
                format('Account %s is spending %s%% above expected rate',
                       account_rec.slurm_account,
                       ROUND(account_rec.daily_variance_pct, 1))::TEXT;
        END IF;

        -- Low burn rate alert
        IF account_rec.daily_variance_pct < -30 THEN
            RETURN QUERY SELECT
                account_rec.id,
                'burn_rate_low'::VARCHAR(64),
                'warning'::VARCHAR(32),
                format('Account %s is underspending by %s%%',
                       account_rec.slurm_account,
                       ROUND(ABS(account_rec.daily_variance_pct), 1))::TEXT;
        END IF;

        -- Budget health alerts
        IF account_rec.budget_health_score < 40 THEN
            RETURN QUERY SELECT
                account_rec.id,
                'budget_threshold'::VARCHAR(64),
                'critical'::VARCHAR(32),
                format('Account %s budget health score is critical: %s%%',
                       account_rec.slurm_account,
                       ROUND(account_rec.budget_health_score, 1))::TEXT;
        END IF;

        -- Projected depletion alert
        IF account_rec.projected_depletion_date IS NOT NULL
           AND account_rec.projected_depletion_date < account_rec.end_date THEN
            RETURN QUERY SELECT
                account_rec.id,
                'overspend_risk'::VARCHAR(64),
                'warning'::VARCHAR(32),
                format('Account %s projected to deplete on %s (before grant end %s)',
                       account_rec.slurm_account,
                       account_rec.projected_depletion_date::DATE,
                       account_rec.end_date::DATE)::TEXT;
        END IF;

    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Function to process daily burn rate updates for all active accounts
CREATE OR REPLACE FUNCTION process_daily_burn_rates(p_date DATE DEFAULT CURRENT_DATE)
RETURNS INTEGER AS $$
DECLARE
    account_id_rec RECORD;
    processed_count INTEGER := 0;
BEGIN
    -- Update burn rates for all active accounts
    FOR account_id_rec IN
        SELECT id FROM budget_accounts
        WHERE status = 'active'
          AND start_date <= p_date
          AND end_date >= p_date
    LOOP
        PERFORM update_burn_rate_metrics(account_id_rec.id, p_date);
        processed_count := processed_count + 1;
    END LOOP;

    RETURN processed_count;
END;
$$ LANGUAGE plpgsql;