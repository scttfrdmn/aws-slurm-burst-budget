# API Reference

Complete REST API reference for AWS SLURM Bursting Budget system.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently supports:
- **API Key Authentication** (optional, configurable)
- **JWT Authentication** (optional, configurable)
- **No Authentication** (default for internal networks)

## Core Endpoints

### Budget Operations

#### `POST /budget/check`
Check budget availability for job submission (used by SLURM plugin).

**Request Body:**
```json
{
  "account": "research-proj-001",
  "partition": "gpu-aws",
  "nodes": 2,
  "cpus": 16,
  "gpus": 4,
  "memory": "64GB",
  "wall_time": "04:00:00",
  "user_id": "researcher1"
}
```

**Response:**
```json
{
  "available": true,
  "estimated_cost": 125.50,
  "hold_amount": 150.60,
  "transaction_id": "txn_1694123456789_001",
  "budget_remaining": 2349.40,
  "recommendation": "Job should run efficiently on AWS",
  "details": {
    "account_balance": 2500.00,
    "current_hold": 150.60,
    "hold_percentage": 1.2,
    "advisor_confidence": 0.89
  }
}
```

#### `POST /budget/reconcile`
Reconcile actual job costs after completion.

**Request Body:**
```json
{
  "job_id": "slurm_67890",
  "actual_cost": 118.75,
  "transaction_id": "txn_1694123456789_001",
  "job_metadata": "{\"performance\": \"high_cpu\"}"
}
```

**Response:**
```json
{
  "success": true,
  "original_hold": 150.60,
  "actual_charge": 118.75,
  "refund_amount": 31.85,
  "transaction_id": "txn_1694123456789_001",
  "message": "Job reconciliation completed successfully"
}
```

## Account Management

#### `GET /accounts`
List budget accounts with optional filtering.

**Query Parameters:**
- `limit` (int): Maximum number of accounts to return (1-100)
- `offset` (int): Number of accounts to skip
- `status` (string): Filter by status (active, inactive, suspended)

**Response:**
```json
[
  {
    "id": 1,
    "slurm_account": "research-proj-001",
    "name": "ML Research Project",
    "description": "Machine learning research for CS department",
    "budget_limit": 5000.00,
    "budget_used": 1250.75,
    "budget_held": 320.50,
    "has_incremental_budget": true,
    "next_allocation_date": "2025-10-01T00:00:00Z",
    "total_allocated": 2500.00,
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-12-31T23:59:59Z",
    "status": "active",
    "created_at": "2025-01-01T10:00:00Z",
    "updated_at": "2025-09-14T08:30:00Z"
  }
]
```

#### `POST /accounts`
Create a new budget account.

**Request Body:**
```json
{
  "slurm_account": "new-research-proj",
  "name": "New Research Project",
  "description": "Quantum computing research",
  "budget_limit": 10000.00,
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-12-31T23:59:59Z",
  "has_incremental_budget": true,
  "allocation_schedule": {
    "total_budget": 12000.00,
    "allocation_amount": 1000.00,
    "allocation_frequency": "monthly",
    "start_date": "2025-01-01T00:00:00Z",
    "auto_allocate": true
  }
}
```

#### `GET /accounts/{account}`
Get detailed account information.

#### `PUT /accounts/{account}`
Update account settings.

#### `DELETE /accounts/{account}`
Delete account (only if no active transactions).

## Grant Management

#### `GET /grants`
List research grants with filtering.

**Query Parameters:**
- `status`: Filter by grant status (pending, active, suspended, completed, cancelled)
- `funding_agency`: Filter by funding agency
- `active_only`: Show only currently active grants
- `limit`/`offset`: Pagination

**Response:**
```json
[
  {
    "id": 1,
    "grant_number": "NSF-2025-12345",
    "funding_agency": "National Science Foundation",
    "agency_program": "Computer and Information Science and Engineering",
    "principal_investigator": "Dr. Jane Smith",
    "co_investigators": ["Dr. Bob Johnson", "Dr. Alice Chen"],
    "institution": "Research University",
    "department": "Computer Science",
    "grant_start_date": "2025-01-01T00:00:00Z",
    "grant_end_date": "2027-12-31T23:59:59Z",
    "total_award_amount": 750000.00,
    "direct_costs": 576923.08,
    "indirect_cost_rate": 0.30,
    "indirect_costs": 173076.92,
    "budget_period_months": 12,
    "current_budget_period": 1,
    "status": "active",
    "federal_award_id": "47.070",
    "internal_project_code": "CS-ML-2025",
    "created_at": "2024-12-15T10:00:00Z"
  }
]
```

#### `POST /grants`
Create a new research grant.

#### `GET /grants/{grant_number}`
Get detailed grant information with budget periods and burn rate analysis.

## Burn Rate Analytics

#### `GET /burn-rate/{account}`
Get burn rate analysis for a budget account.

**Query Parameters:**
- `period`: Analysis period (7d, 30d, 90d, 6m, 1y)
- `include_projection`: Include spending projections (true/false)
- `include_alerts`: Include active alerts (true/false)

**Response:**
```json
{
  "account": "research-proj-001",
  "analysis_period": "30d",
  "time_range": {
    "start_date": "2025-08-15T00:00:00Z",
    "end_date": "2025-09-14T00:00:00Z",
    "days": 30
  },
  "current_metrics": {
    "daily_spend_rate": 145.67,
    "daily_expected_rate": 125.00,
    "variance_percentage": 16.5,
    "rolling_7day_average": 142.33,
    "rolling_30day_average": 138.92,
    "cumulative_spend": 4175.25,
    "cumulative_expected": 3750.00,
    "cumulative_variance_pct": 11.3,
    "budget_health_score": 78.5,
    "budget_remaining_amount": 825.75,
    "budget_remaining_percent": 16.5,
    "time_remaining_days": 108,
    "burn_rate_status": "ON_TRACK",
    "budget_health_status": "CONCERN"
  },
  "projection": {
    "projected_end_date": "2025-12-31T00:00:00Z",
    "projected_final_spend": 5234.67,
    "projected_overrun": 234.67,
    "confidence_level": 0.85,
    "projection_method": "linear_regression",
    "risk_level": "MEDIUM"
  },
  "alerts": [
    {
      "id": 123,
      "alert_type": "burn_rate_high",
      "severity": "warning",
      "message": "Spending 16.5% above expected rate",
      "triggered_at": "2025-09-14T08:00:00Z",
      "status": "active"
    }
  ],
  "recommendations": [
    "Monitor spending closely as variance is approaching concerning levels",
    "Consider optimizing resource requests for similar workloads"
  ]
}
```

#### `GET /burn-rate/grant/{grant_number}`
Get burn rate analysis for a specific grant.

## ASBX Integration

#### `POST /asbx/reconcile`
Process ASBX v0.2.0 cost data for automatic reconciliation.

**Request Body:**
```json
{
  "job_cost_data": {
    "job_id": "asbx_job_12345",
    "slurm_job_id": "67890",
    "account": "NSF-2025-12345",
    "partition": "gpu-aws",
    "submitted_at": "2025-09-14T08:00:00Z",
    "started_at": "2025-09-14T08:05:00Z",
    "completed_at": "2025-09-14T12:05:00Z",
    "job_state": "COMPLETED",
    "estimated_cost": 125.00,
    "actual_cost": 118.50,
    "aws_cost": 95.20,
    "local_cost": 23.30,
    "cpu_efficiency": 0.85,
    "memory_efficiency": 0.78,
    "burst_decision": "AWS",
    "budget_transaction_id": "txn_1694123456789_001"
  },
  "auto_reconcile": true,
  "update_cost_model": true,
  "generate_report": true
}
```

**Response:**
```json
{
  "success": true,
  "reconciliation_id": "asbx_recon_1694123456789",
  "original_transaction": "txn_1694123456789_001",
  "estimated_cost": 125.00,
  "actual_cost": 118.50,
  "cost_variance": -6.50,
  "cost_variance_pct": -5.2,
  "refund_amount": 6.50,
  "estimation_accuracy": 0.948,
  "model_update_applied": true,
  "compliance_report_generated": true,
  "report_path": "/reports/compliance_asbx_job_12345.pdf",
  "message": "ASBX cost reconciliation completed successfully",
  "recommendations": [
    "Cost estimation accuracy is excellent for this workload type",
    "Consider similar resource allocation for future ML jobs"
  ]
}
```

#### `POST /asbx/epilog`
Process SLURM epilog data for ASBX integration.

**Request Body:**
```json
{
  "job_id": "67890",
  "account": "NSF-2025-12345",
  "partition": "gpu-aws",
  "user_id": "researcher1",
  "job_state": "COMPLETED",
  "exit_code": 0,
  "submit_time": 1694123400,
  "start_time": 1694123700,
  "end_time": 1694138100,
  "allocated_nodes": 2,
  "allocated_cpus": 16,
  "allocated_gpus": 4,
  "asbx_data_path": "/var/spool/asbx/job_67890_cost.json"
}
```

#### `GET /asbx/status`
Get ASBX integration health status.

**Response:**
```json
{
  "asbx_version": "0.2.0",
  "integration_enabled": true,
  "last_data_import": "2025-09-14T11:30:00Z",
  "total_jobs_reconciled": 1247,
  "successful_reconciliations": 1238,
  "failed_reconciliations": 9,
  "average_reconciliation_time": "2.3s",
  "cost_model_accuracy": 0.89,
  "last_health_check": "2025-09-14T11:35:00Z",
  "health_status": "healthy"
}
```

## ASBA Integration (Academic Slurm Burst Allocation)

### Decision Making APIs for Intelligent Resource Allocation

#### `POST /asba/budget-status`
Get comprehensive budget status for ASBA decision making.

**Request Body:**
```json
{
  "account": "research-proj-001",
  "grant_number": "NSF-2025-12345",
  "user_id": "researcher1"
}
```

**Response:**
```json
{
  "account": "research-proj-001",
  "grant_number": "NSF-2025-12345",
  "budget_limit": 5000.00,
  "budget_used": 1250.75,
  "budget_held": 320.50,
  "budget_available": 3428.75,
  "budget_utilization": 25.015,
  "daily_burn_rate": 125.50,
  "expected_daily_rate": 100.00,
  "burn_rate_variance": 25.5,
  "budget_health_score": 78.5,
  "health_status": "CONCERN",
  "days_remaining": 90,
  "risk_level": "MEDIUM",
  "can_afford_aws_burst": true,
  "recommended_decision": "PREFER_LOCAL",
  "decision_reasoning": [
    "Budget health concerning with 25.5% overspend rate",
    "Sufficient budget for moderate AWS usage",
    "Recommend local execution for cost efficiency"
  ],
  "last_updated": "2025-09-14T12:00:00Z"
}
```

#### `POST /asba/affordability-check`
Check if a specific job is affordable and get execution recommendations.

**Request Body:**
```json
{
  "account": "research-proj-001",
  "estimated_aws_cost": 125.50,
  "estimated_local_time": 480,
  "job_priority": "high",
  "job_deadline": "2025-09-16T12:00:00Z"
}
```

**Response:**
```json
{
  "affordable": true,
  "recommended_decision": "AWS",
  "confidence_level": 0.85,
  "estimated_aws_cost": 125.50,
  "budget_impact": 3.66,
  "cost_opportunity_ratio": 0.75,
  "time_to_deadline": 2880,
  "aws_completion_time": "2025-09-14T16:00:00Z",
  "local_completion_time": "2025-09-15T00:00:00Z",
  "budget_risk": "LOW",
  "deadline_risk": "MEDIUM",
  "overall_risk": "LOW",
  "decision_factors": {
    "budget_health": "good",
    "cost_efficiency": 0.8,
    "deadline_pressure": 0.6
  },
  "reasoning": [
    "Job cost $125.50 is within budget limits",
    "AWS execution saves 8 hours vs local",
    "Deadline in 2 days makes time savings valuable"
  ],
  "message": "AWS burst recommended for deadline optimization"
}
```

#### `POST /asba/grant-timeline`
Get grant timeline and deadline information for resource planning.

**Request Body:**
```json
{
  "account": "nsf-ml-research",
  "grant_number": "NSF-2025-12345",
  "look_ahead_days": 90,
  "include_alerts": true
}
```

**Response:**
```json
{
  "account": "nsf-ml-research",
  "grant_number": "NSF-2025-12345",
  "grant_start_date": "2025-01-01T00:00:00Z",
  "grant_end_date": "2027-12-31T23:59:59Z",
  "current_period": 2,
  "total_periods": 3,
  "period_end_date": "2025-12-31T23:59:59Z",
  "days_until_period_end": 108,
  "days_until_grant_end": 838,
  "next_allocation": {
    "date": "2026-01-01T00:00:00Z",
    "amount": 250000.00,
    "description": "Annual budget allocation - Year 2",
    "type": "AUTOMATIC",
    "days_from_now": 108
  },
  "upcoming_deadlines": [
    {
      "type": "CONFERENCE",
      "description": "ICML 2025 Paper Submission",
      "date": "2025-12-08T23:59:59Z",
      "days_from_now": 85,
      "severity": "HIGH",
      "budget_impact": "May require intensive compute for final experiments",
      "recommendations": [
        "Reserve budget for final experiments",
        "Consider AWS burst for large-scale validation"
      ]
    },
    {
      "type": "GRANT_REPORT",
      "description": "NSF Annual Report Due",
      "date": "2025-12-31T23:59:59Z",
      "days_from_now": 108,
      "severity": "CRITICAL",
      "budget_impact": "Budget reconciliation required",
      "recommendations": [
        "Complete spending documentation",
        "Prepare financial summary"
      ]
    }
  ],
  "current_urgency": "MEDIUM",
  "bursting_recommendation": "NORMAL",
  "optimization_advice": [
    "Budget health is good, moderate AWS usage acceptable",
    "Plan for ICML deadline compute requirements",
    "Monitor burn rate as grant approaches mid-point"
  ]
}
```

#### `POST /asba/burst-decision`
Get comprehensive burst decision recommendations with multi-factor analysis.

**Request Body:**
```json
{
  "account": "research-proj-001",
  "estimated_aws_cost": 200.00,
  "estimated_local_time": 720,
  "job_priority": "critical",
  "job_deadline": "2025-09-16T09:00:00Z",
  "conference_deadline": "2025-09-20T23:59:59Z",
  "research_phase": "validation",
  "collaboration_impact": true
}
```

**Response:**
```json
{
  "recommended_action": "AWS",
  "confidence": 0.92,
  "urgency_level": "HIGH",
  "budget_impact": 5.8,
  "affordability_score": 0.89,
  "cost_efficiency": 0.75,
  "timeline_pressure": 0.85,
  "deadline_risk": "HIGH",
  "grant_health_impact": "MINIMAL",
  "decision_factors": [
    {
      "factor": "Deadline Pressure",
      "weight": 0.4,
      "value": 0.85,
      "impact": "POSITIVE",
      "description": "Conference deadline in 6 days requires fast completion"
    },
    {
      "factor": "Budget Health",
      "weight": 0.3,
      "value": 0.78,
      "impact": "POSITIVE",
      "description": "Account has adequate budget for AWS burst"
    },
    {
      "factor": "Cost Efficiency",
      "weight": 0.3,
      "value": 0.72,
      "impact": "NEUTRAL",
      "description": "AWS cost reasonable for time savings"
    }
  ],
  "immediate_actions": [
    "Submit job to AWS immediately for deadline compliance",
    "Monitor budget impact and adjust future jobs if needed",
    "Notify collaborators of accelerated timeline"
  ],
  "longterm_suggestions": [
    "Plan earlier for conference deadlines",
    "Consider optimizing job for better cost efficiency",
    "Review grant budget allocation for validation phase"
  ],
  "message": "AWS burst strongly recommended due to critical conference deadline"
}
```

## System Endpoints

#### `GET /health`
Service health check.

**Response:**
```json
{
  "status": "healthy",
  "version": "0.1.2",
  "timestamp": "2025-09-14T12:00:00Z",
  "services": {
    "database": "healthy",
    "advisor": "healthy"
  },
  "uptime": "2h 15m 30s"
}
```

#### `GET /metrics`
Prometheus metrics endpoint.

**Response:**
```
# HELP asbb_budget_accounts_total Total number of budget accounts
# TYPE asbb_budget_accounts_total gauge
asbb_budget_accounts_total 15

# HELP asbb_budget_used_total Total budget used across all accounts
# TYPE asbb_budget_used_total gauge
asbb_budget_used_total 45250.75

# HELP asbb_transactions_total Total number of transactions
# TYPE asbb_transactions_total counter
asbb_transactions_total 2847
```

#### `GET /version`
Application version information.

**Response:**
```json
{
  "version": "0.1.2",
  "git_commit": "c2e01d2",
  "build_time": "2025-09-14T10:30:00Z",
  "go_version": "go1.21.0",
  "os": "linux",
  "arch": "amd64"
}
```

## Error Handling

All endpoints return consistent error responses:

```json
{
  "error": {
    "code": "INSUFFICIENT_BUDGET",
    "message": "Insufficient budget for account 'research-proj-001'",
    "details": "Required: $150.60, Available: $75.25",
    "field": "budget_limit"
  },
  "request_id": "req_1694123456789",
  "timestamp": "2025-09-14T12:00:00Z"
}
```

### Error Codes

- `VALIDATION_ERROR`: Invalid request parameters
- `NOT_FOUND`: Resource not found
- `INSUFFICIENT_BUDGET`: Budget limit exceeded
- `ACCOUNT_INACTIVE`: Account not active
- `TRANSACTION_FAILED`: Transaction processing failed
- `SERVICE_UNAVAILABLE`: External service unavailable
- `DATABASE_ERROR`: Database operation failed

## Rate Limiting

- **Default**: 100 requests per minute per IP
- **Authenticated**: 1000 requests per minute per API key
- **Burst**: 200 requests in 30-second window

## Pagination

List endpoints support pagination:

```
GET /accounts?limit=50&offset=100
```

## Response Codes

- `200 OK`: Successful operation
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required
- `402 Payment Required`: Budget insufficient
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: External service unavailable

## Examples

### Complete Workflow Example

```bash
# 1. Create grant account
curl -X POST /api/v1/grants \
  -H "Content-Type: application/json" \
  -d '{
    "grant_number": "NSF-2025-12345",
    "funding_agency": "National Science Foundation",
    "principal_investigator": "Dr. Research Leader",
    "institution": "University Research",
    "grant_start_date": "2025-01-01T00:00:00Z",
    "grant_end_date": "2027-12-31T23:59:59Z",
    "total_award_amount": 750000.00,
    "indirect_cost_rate": 0.30
  }'

# 2. Create budget account linked to grant
curl -X POST /api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "slurm_account": "nsf-ml-proj",
    "name": "NSF ML Research",
    "budget_limit": 250000.00,
    "start_date": "2025-01-01T00:00:00Z",
    "end_date": "2025-12-31T23:59:59Z"
  }'

# 3. Check budget before job submission
curl -X POST /api/v1/budget/check \
  -H "Content-Type: application/json" \
  -d '{
    "account": "nsf-ml-proj",
    "partition": "gpu-aws",
    "nodes": 4,
    "cpus": 32,
    "gpus": 8,
    "wall_time": "08:00:00"
  }'

# 4. Monitor burn rate
curl "/api/v1/burn-rate/nsf-ml-proj?period=30d&include_projection=true"

# 5. Get grant compliance report
curl -X POST "/api/v1/grants/NSF-2025-12345/reports" \
  -H "Content-Type: application/json" \
  -d '{
    "report_type": "financial",
    "budget_period": 1,
    "format": "pdf"
  }'
```

This API provides complete programmatic access to all budget management, grant tracking, and burn rate analytics functionality.