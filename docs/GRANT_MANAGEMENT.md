# Grant Management & Long-term Budget Tracking

This document provides comprehensive guidance for managing multi-year research grants and tracking burn rates over extended periods using the AWS SLURM Bursting Budget system.

## 🎓 Overview

The grant management system supports:
- **Multi-year grants** spanning months to years (e.g., 3-year NSF, 5-year NIH grants)
- **Budget period tracking** within grants (annual, quarterly, custom periods)
- **Burn rate analytics** with variance tracking and projections
- **Compliance reporting** for federal and private funding agencies
- **Automated alerts** for budget variances and grant deadlines

## 🏗️ Grant Account Structure

### Grant Types Supported
- **Federal Grants**: NSF, NIH, DOE, NASA, etc.
- **Private Foundation Grants**: Research foundations, corporate funding
- **Multi-year Contracts**: Long-term research agreements
- **Multi-PI Grants**: Collaborative research with multiple investigators

### Key Grant Attributes
- **Grant Number**: Unique identifier (e.g., NSF-2025-12345)
- **Funding Agency**: Source of funding with program details
- **Principal Investigator(s)**: Lead researcher(s) and collaborators
- **Award Amount**: Total funding with direct/indirect cost breakdown
- **Grant Period**: Start and end dates with budget period structure
- **Compliance Requirements**: Agency-specific reporting and oversight

## 💰 Grant Creation Examples

### Example 1: 3-Year NSF Computer Science Grant
```bash
asbb grant create \
  --number=NSF-2025-12345 \
  --agency="National Science Foundation" \
  --program="Computer and Information Science and Engineering" \
  --pi="Dr. Jane Smith" \
  --co-pi="Dr. Bob Johnson,Dr. Alice Chen" \
  --institution="Research University" \
  --department="Computer Science" \
  --amount=750000 \
  --start=2025-01-01 \
  --end=2027-12-31 \
  --periods=12 \
  --indirect=0.30 \
  --federal-id="47.070" \
  --project="ML-HPC-2025"
```

**Result**: 3-year grant with $250K/year, 30% indirect costs, tracked by annual budget periods

### Example 2: 5-Year NIH Medical Research Grant
```bash
asbb grant create \
  --number=NIH-R01-567890 \
  --agency="National Institutes of Health" \
  --program="National Cancer Institute" \
  --pi="Dr. Medical Researcher" \
  --institution="Medical University" \
  --department="Oncology Research" \
  --amount=1250000 \
  --start=2025-07-01 \
  --end=2030-06-30 \
  --periods=12 \
  --indirect=0.25 \
  --federal-id="93.393"
```

**Result**: 5-year NIH grant with $250K/year, quarterly reporting capability

### Example 3: 2-Year DOE Energy Research Grant
```bash
asbb grant create \
  --number=DOE-DE-SC-98765 \
  --agency="Department of Energy" \
  --program="Office of Science - Basic Energy Sciences" \
  --pi="Dr. Energy Expert" \
  --institution="National Lab Consortium" \
  --amount=500000 \
  --start=2025-10-01 \
  --end=2027-09-30 \
  --periods=6 \
  --indirect=0.35 \
  --cost-center="BES-Materials"
```

**Result**: 2-year DOE grant with semi-annual budget periods

## 📊 Burn Rate Analytics

### Understanding Burn Rate Metrics

#### **Daily Metrics**
- **Daily Spend Rate**: Actual spending per day
- **Daily Expected Rate**: Planned spending per day based on grant timeline
- **Variance Percentage**: `((Actual - Expected) / Expected) × 100`

#### **Health Scoring (0-100 scale)**
- **90-100**: Excellent budget management
- **80-89**: Good, minor variances
- **60-79**: Concerning, monitor closely
- **40-59**: Warning, intervention needed
- **0-39**: Critical, immediate action required

#### **Burn Rate Status**
- **ON_TRACK**: Variance within ±20%
- **OVERSPENDING**: Spending >20% above expected
- **UNDERSPENDING**: Spending >20% below expected

### Burn Rate Analysis Commands

#### **Basic Analysis**
```bash
# Current burn rate status
asbb burn-rate NSF-2025-12345

# Output:
# Daily Spend Rate: $145.67 (Expected: $125.00)
# Variance: +16.5% (ON_TRACK)
# Health Score: 78% (CONCERN)
# Days Remaining: 847
```

#### **Historical Analysis**
```bash
# 6-month historical analysis with projections
asbb burn-rate NSF-2025-12345 --period=6m --projection

# Output:
# Analysis Period: 6m (2025-03-14 to 2025-09-14)
# Rolling 30-Day Average: $142.33/day
# Cumulative Variance: +12.8%
# Projected Final Spend: $773,245 (+$23,245 overrun)
# Risk Level: MEDIUM (85% confidence)
```

#### **Alert Monitoring**
```bash
# Check for active budget alerts
asbb burn-rate NSF-2025-12345 --alerts-only

# Output:
# 🚨 CRITICAL: Account NSF-2025-12345 is spending 45% above expected rate
# ⚠️  WARNING: Projected to deplete on 2026-11-15 (before grant end 2027-12-31)
```

### Long-term Tracking Scenarios

#### **Scenario 1: 3-Year Grant Monitoring**
```bash
# Year 1: Establish baseline
asbb burn-rate NSF-2025-12345 --period=30d
# Health Score: 95% (HEALTHY)

# Year 2: Monitor for changes
asbb burn-rate NSF-2025-12345 --period=90d --projection
# Health Score: 82% (GOOD)
# Projected Depletion: On schedule

# Year 3: Final year tracking
asbb burn-rate NSF-2025-12345 --period=1y --projection
# Health Score: 67% (CONCERN)
# Projected Overrun: $15,000
```

#### **Scenario 2: Multi-Grant Portfolio**
```bash
# Monitor all active grants
asbb grant list --active

# Bulk burn rate analysis
for grant in $(asbb grant list --active --format=ids); do
  echo "=== $grant ==="
  asbb burn-rate $grant --period=30d --alerts-only
done
```

## 📈 Advanced Analytics Features

### Predictive Modeling

The system uses historical spending patterns to project:
- **Depletion Dates**: When budget will be exhausted
- **Final Spend Amount**: Projected total spending at grant end
- **Overrun/Underrun**: Projected budget variance
- **Risk Assessment**: Statistical confidence in projections

### Performance Learning

Integration with ASBX provides:
- **Cost Estimation Improvement**: Learn from actual vs estimated costs
- **Resource Optimization**: Identify underutilized resources
- **Efficiency Metrics**: Track CPU, memory, GPU utilization
- **Workload Profiling**: Categorize jobs for better cost prediction

## 🚨 Alert Management

### Alert Types
- **`burn_rate_high`**: Spending significantly above expected rate
- **`burn_rate_low`**: Spending significantly below expected rate
- **`budget_threshold`**: Budget health score below acceptable level
- **`grant_expiring`**: Grant approaching end date
- **`period_ending`**: Budget period nearing completion
- **`overspend_risk`**: Projected to exceed budget
- **`underspend_risk`**: Likely to underspend significantly
- **`compliance_warning`**: Agency deadline or requirement alert

### Alert Thresholds
- **Critical**: >50% variance from expected spending
- **Warning**: 20-50% variance from expected spending
- **Info**: <20% variance, informational alerts

### Managing Alerts
```bash
# View all active alerts
asbb alerts list

# Acknowledge critical alerts
asbb alerts acknowledge 123 --user="Dr. Smith" --notes="Reviewing with finance team"

# Filter alerts by grant
asbb alerts list --grant=NSF-2025-12345 --severity=critical
```

## 📋 Compliance Reporting

### Supported Report Types
- **Financial Reports**: Spending summaries by budget period
- **Technical Reports**: Research progress and deliverables
- **Compliance Reports**: Agency-specific requirement tracking
- **Annual Reports**: Comprehensive yearly summaries

### Generating Reports
```bash
# Annual financial report for NSF
asbb grant report NSF-2025-12345 --type=financial --format=pdf --period=1

# Compliance report for current budget period
asbb grant report NIH-R01-567890 --type=compliance --format=csv

# Custom date range report
asbb grant report DOE-DE-SC-98765 --type=financial \
  --start=2025-01-01 --end=2025-12-31 --format=json
```

## 🔗 ASBX Integration Workflow

### Automatic Cost Reconciliation

1. **Job Submission**: SLURM plugin checks budget availability
2. **Job Execution**: ASBX tracks actual costs and performance
3. **Job Completion**: SLURM epilog triggers cost reconciliation
4. **Data Import**: ASBX v0.2.0 cost data imported automatically
5. **Reconciliation**: Actual costs reconciled with budget holds
6. **Learning**: Performance data improves future estimates
7. **Reporting**: Compliance reports generated if required

### Integration Endpoints

#### **Cost Reconciliation**
```bash
curl -X POST /api/v1/asbx/reconcile \
  -H "Content-Type: application/json" \
  -d '{
    "job_cost_data": {
      "job_id": "job_12345",
      "account": "NSF-2025-12345",
      "estimated_cost": 125.00,
      "actual_cost": 118.50,
      "budget_transaction_id": "txn_abc123"
    },
    "auto_reconcile": true,
    "update_cost_model": true,
    "generate_report": true
  }'
```

#### **SLURM Epilog Processing**
```bash
curl -X POST /api/v1/asbx/epilog \
  -H "Content-Type: application/json" \
  -d '{
    "job_id": "12345",
    "account": "NSF-2025-12345",
    "job_state": "COMPLETED",
    "asbx_data_path": "/var/spool/asbx/job_12345_cost.json"
  }'
```

### Integration Status Monitoring
```bash
# Check ASBX integration health
curl /api/v1/asbx/status

# Response:
{
  "asbx_version": "0.2.0",
  "integration_enabled": true,
  "total_jobs_reconciled": 1247,
  "cost_model_accuracy": 0.89,
  "health_status": "healthy"
}
```

## 🛠️ Configuration

### Grant Management Configuration
```yaml
# config.yaml
grant_management:
  enabled: true
  default_budget_period_months: 12
  auto_create_periods: true
  compliance_reporting: true
  burn_rate_monitoring: true
  alert_thresholds:
    critical_variance: 50.0
    warning_variance: 20.0
    health_score_warning: 60.0
    health_score_critical: 40.0

asbx_integration:
  enabled: true
  asbx_endpoint: "http://localhost:8082"
  auto_reconcile: true
  update_cost_model: true
  data_retention_days: 365
  reconciliation_timeout: "5m"
  max_retries: 3
  notification_enabled: true
  compliance_reporting: true
```

## 📚 Best Practices

### Grant Setup
1. **Create grant account first** with all required metadata
2. **Link budget accounts** to grant for cost attribution
3. **Set up budget periods** appropriate for agency requirements
4. **Configure alerts** for variance thresholds
5. **Enable ASBX integration** for automatic reconciliation

### Burn Rate Monitoring
1. **Review daily** for critical grants approaching deadlines
2. **Analyze monthly** for trend identification
3. **Project quarterly** for budget planning
4. **Report annually** for compliance and renewal preparation

### Performance Optimization
1. **Monitor efficiency metrics** from ASBX integration
2. **Adjust resource requests** based on utilization data
3. **Update cost models** with actual performance data
4. **Optimize workloads** using efficiency recommendations

This comprehensive grant management system provides everything needed for long-term research funding tracking and compliance in academic computing environments.