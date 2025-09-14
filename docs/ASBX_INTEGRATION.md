# ASBX Integration Guide

Complete integration guide for aws-slurm-burst (ASBX) v0.2.0 cost data and performance feedback.

## 🔗 Overview

The ASBX integration provides seamless cost reconciliation between aws-slurm-burst execution data and budget management, enabling:
- **Automatic cost reconciliation** from ASBX v0.2.0 cost exports
- **Performance-based cost model learning** for improved accuracy
- **SLURM epilog integration** for real-time post-job processing
- **Compliance reporting** with detailed cost breakdowns

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SLURM Jobs    │    │  ASBX v0.2.0    │    │  ASBB Budget    │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Submit Check │ │────│ │Cost Tracking│ │────│ │Reconciliation│ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │Epilog Script│ │────│ │Export Data  │ │────│ │Update Budget│ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                      ┌─────────────────┐
                      │  Performance    │
                      │  Learning       │
                      │ ┌─────────────┐ │
                      │ │Cost Models  │ │
                      │ └─────────────┘ │
                      └─────────────────┘
```

## 🚀 Setup and Configuration

### Prerequisites
- ASBX v0.2.0 or later with cost export enabled
- SLURM epilog script integration
- Network connectivity between ASBX and ASBB services

### Configuration
```yaml
# /etc/asbb/config.yaml
asbx_integration:
  enabled: true
  asbx_endpoint: "http://asbx-service:8082"
  auto_reconcile: true
  update_cost_model: true
  data_retention_days: 365
  reconciliation_timeout: "5m"
  max_retries: 3
  notification_enabled: true
  compliance_reporting: true

  # Performance learning settings
  cost_model_learning:
    enabled: true
    accuracy_threshold: 0.8
    variance_threshold: 20.0
    feedback_frequency: "daily"
```

### SLURM Integration

#### Epilog Script (`/etc/slurm/epilog.d/asbx_budget_epilog.sh`)
```bash
#!/bin/bash
# ASBX-ASBB Integration Epilog Script

# Environment variables from SLURM
JOB_ID="${SLURM_JOB_ID}"
ACCOUNT="${SLURM_JOB_ACCOUNT}"
PARTITION="${SLURM_JOB_PARTITION}"
USER_ID="${SLURM_JOB_USER}"
JOB_STATE="${SLURM_JOB_EXIT_CODE}"

# ASBX data path (configured in ASBX)
ASBX_DATA_PATH="/var/spool/asbx/job_${JOB_ID}_cost.json"

# Send epilog data to ASBB
curl -s -X POST http://localhost:8080/api/v1/asbx/epilog \
  -H "Content-Type: application/json" \
  -d "{
    \"job_id\": \"${JOB_ID}\",
    \"account\": \"${ACCOUNT}\",
    \"partition\": \"${PARTITION}\",
    \"user_id\": \"${USER_ID}\",
    \"job_state\": \"${JOB_STATE}\",
    \"submit_time\": ${SLURM_JOB_SUBMIT_TIME:-0},
    \"start_time\": ${SLURM_JOB_START_TIME:-0},
    \"end_time\": ${SLURM_JOB_END_TIME:-0},
    \"allocated_nodes\": ${SLURM_JOB_NUM_NODES:-0},
    \"allocated_cpus\": ${SLURM_JOB_CPUS_PER_NODE:-0},
    \"asbx_data_path\": \"${ASBX_DATA_PATH}\"
  }" \
  >> /var/log/slurm/asbx_budget_epilog.log 2>&1
```

## 📊 Cost Reconciliation Workflow

### 1. Job Submission
```bash
# SLURM job submission with budget check
sbatch --account=NSF-2025-12345 --partition=gpu-aws my_research_job.sbatch

# ASBB creates budget hold:
# Transaction ID: txn_1694123456789_001
# Hold Amount: $150.00 (estimate: $125.00 + 20% buffer)
```

### 2. Job Execution (ASBX)
```json
// ASBX tracks actual costs during execution
{
  "job_id": "slurm_67890",
  "estimated_cost": 125.00,
  "actual_cost": 118.50,
  "aws_cost": 95.20,
  "local_cost": 23.30,
  "cpu_efficiency": 0.85,
  "memory_efficiency": 0.78,
  "burst_decision": "AWS"
}
```

### 3. Automatic Reconciliation
```bash
# SLURM epilog triggers reconciliation
# POST /api/v1/asbx/reconcile

# Response:
{
  "success": true,
  "reconciliation_id": "asbx_recon_1694123456789",
  "cost_variance": -6.50,
  "cost_variance_pct": -5.2,
  "refund_amount": 6.50,
  "estimation_accuracy": 0.948,
  "model_update_applied": true,
  "message": "Cost reconciliation completed successfully"
}
```

### 4. Budget Update
- **Original Hold**: $150.00
- **Actual Charge**: $118.50
- **Refund Issued**: $31.50 ($150.00 - $118.50)
- **Account Balance**: Updated automatically
- **Grant Tracking**: Burn rate metrics updated

## 🧠 Performance Learning

### Cost Model Improvement

The system learns from ASBX performance data to improve future estimates:

#### **Efficiency Metrics Tracking**
```json
{
  "job_id": "research_job_001",
  "cpu_efficiency": 0.85,
  "memory_efficiency": 0.78,
  "gpu_efficiency": 0.92,
  "actual_vs_estimated_ratio": 0.948,
  "performance_profile": "compute_intensive",
  "optimization_opportunities": [
    "Consider reducing memory allocation by 15%",
    "CPU efficiency could be improved with better parallelization"
  ]
}
```

#### **Cost Estimation Accuracy**
- **Target Accuracy**: >90% within ±10% of actual cost
- **Learning Rate**: Improves over time with more data points
- **Profile-based**: Different accuracy targets for different workload types

### Feedback Loop
1. **Actual Performance Data** → Cost model training
2. **Improved Estimates** → Better budget planning
3. **Resource Optimization** → Lower costs
4. **Better Utilization** → Improved efficiency

## 📊 Monitoring and Troubleshooting

### Integration Health Check
```bash
# Check ASBX integration status
curl /api/v1/asbx/status

{
  "asbx_version": "0.2.0",
  "integration_enabled": true,
  "last_data_import": "2025-09-14T10:30:00Z",
  "total_jobs_reconciled": 1247,
  "successful_reconciliations": 1238,
  "failed_reconciliations": 9,
  "average_reconciliation_time": "2.3s",
  "cost_model_accuracy": 0.89,
  "health_status": "healthy"
}
```

### Troubleshooting Common Issues

#### **Data Import Failures**
```bash
# Check ASBX data path accessibility
ls -la /var/spool/asbx/job_*_cost.json

# Verify ASBX export format
jq . /var/spool/asbx/job_12345_cost.json

# Manual reconciliation if needed
asbb reconcile job_12345 --transaction=txn_abc123 --actual-cost=118.50
```

#### **Reconciliation Timeouts**
```bash
# Check reconciliation queue
asbb transactions list --type=hold --status=pending

# Process failed reconciliations
asbb recover --reconcile-orphaned
```

#### **Cost Model Accuracy Issues**
```bash
# Review estimation accuracy trends
asbb analytics accuracy --period=30d

# Check specific workload patterns
asbb analytics performance --profile=compute_intensive
```

## 🔐 Security and Compliance

### Data Security
- **Encrypted Communication**: TLS for all ASBX-ASBB communication
- **Authentication**: API key or JWT-based authentication
- **Audit Trail**: Complete logging of all reconciliation operations
- **Data Retention**: Configurable retention policies for compliance

### Federal Compliance
- **FISMA Requirements**: Security controls for federal grant data
- **Agency Reporting**: Automated reports in required formats
- **Audit Support**: Complete transaction history and documentation
- **Data Sovereignty**: Control over where sensitive grant data is stored

## 🎯 Performance Optimization

### Best Practices
1. **Monitor Integration Health**: Regular status checks
2. **Tune Reconciliation Timing**: Optimize for your workload patterns
3. **Review Cost Model Accuracy**: Ensure estimates improve over time
4. **Set Appropriate Alerts**: Balance notification frequency with utility
5. **Regular Compliance Reports**: Stay ahead of agency requirements

### Performance Metrics
- **Reconciliation Time**: <5 seconds per job typical
- **Cost Model Accuracy**: >90% target within ±10%
- **Data Import Success Rate**: >99% target
- **Alert Response Time**: <1 minute for critical alerts

This comprehensive ASBX integration provides the complete academic research computing budget management ecosystem with seamless cost tracking and compliance reporting.