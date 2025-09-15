# ASBA Integration Guide

Complete integration guide for Academic Slurm Burst Allocation (ASBA) decision making APIs.

## 🎯 Overview

The ASBA integration provides intelligent decision making capabilities for academic research computing, enabling:
- **Budget-aware resource allocation** based on grant health and spending patterns
- **Deadline-driven optimization** for conference submissions and research milestones
- **Multi-factor decision analysis** considering budget, timeline, and research phase
- **Emergency bursting guidance** for critical research deadlines

## 🧠 Decision Making Intelligence

### Core Decision Factors

#### **Budget Health Analysis**
- **Budget utilization**: Percentage of grant funding used
- **Burn rate variance**: Over/under spending vs expected rates
- **Health scoring**: 0-100 scale budget health assessment
- **Risk levels**: LOW, MEDIUM, HIGH, CRITICAL risk classification

#### **Timeline Pressure Assessment**
- **Days to deadline**: Conference, grant report, period end deadlines
- **Research phase urgency**: Exploration vs validation vs publication
- **Collaboration impact**: Effects on other researchers and projects
- **Grant lifecycle stage**: Early, mid, or late in grant period

#### **Cost Efficiency Optimization**
- **AWS vs local cost analysis**: Direct cost comparison
- **Time value calculation**: Cost per hour saved
- **Resource utilization**: CPU, memory, GPU efficiency projections
- **Alternative options**: Spot instances, hybrid execution, deferred execution

## 🔧 API Endpoints

### 1. Budget Status Query

**Purpose**: Get comprehensive budget status for intelligent decision making

**Endpoint**: `POST /api/v1/asba/budget-status`

**Use Cases**:
- ASBA checking budget health before recommending AWS vs local
- Grant managers monitoring spending patterns
- Researchers planning expensive computational experiments

**Example**:
```bash
curl -X POST /api/v1/asba/budget-status \
  -H "Content-Type: application/json" \
  -d '{
    "account": "nsf-ml-research",
    "grant_number": "NSF-2025-12345",
    "user_id": "phd_student_1"
  }'
```

**Key Response Fields**:
- `can_afford_aws_burst`: Boolean decision flag
- `recommended_decision`: PREFER_LOCAL, PREFER_AWS, EITHER, EMERGENCY_ONLY
- `budget_health_score`: 0-100 health assessment
- `decision_reasoning`: Human-readable rationale

### 2. Affordability Check

**Purpose**: Determine if specific job costs are within budget constraints

**Endpoint**: `POST /api/v1/asba/affordability-check`

**Use Cases**:
- Pre-submission cost validation
- Alternative execution strategy comparison
- Risk assessment for expensive jobs

**Example**:
```bash
curl -X POST /api/v1/asba/affordability-check \
  -H "Content-Type: application/json" \
  -d '{
    "account": "nsf-ml-research",
    "estimated_aws_cost": 250.00,
    "estimated_local_time": 600,
    "job_priority": "high",
    "job_deadline": "2025-09-20T23:59:59Z"
  }'
```

**Decision Logic**:
- **Affordable + Low Risk**: Recommend AWS for time savings
- **Affordable + High Cost**: Suggest local with longer timeline
- **Unaffordable**: Provide alternatives (spot instances, optimization, deferral)

### 3. Grant Timeline Analysis

**Purpose**: Provide grant lifecycle and deadline context for resource planning

**Endpoint**: `POST /api/v1/asba/grant-timeline`

**Use Cases**:
- Conference deadline planning
- Grant period budget allocation
- Emergency resource planning
- Long-term research scheduling

**Example**:
```bash
curl -X POST /api/v1/asba/grant-timeline \
  -H "Content-Type: application/json" \
  -d '{
    "grant_number": "NSF-2025-12345",
    "look_ahead_days": 120,
    "include_alerts": true
  }'
```

**Timeline Intelligence**:
- **Upcoming deadlines**: Conference submissions, grant reports, renewals
- **Budget allocation schedule**: When new funding becomes available
- **Critical periods**: End-of-grant urgency, period transitions
- **Emergency options**: Reallocation, cost sharing, deadline extensions

### 4. Comprehensive Burst Decision

**Purpose**: Multi-factor analysis for optimal resource allocation decisions

**Endpoint**: `POST /api/v1/asba/burst-decision`

**Use Cases**:
- Complex research scenarios requiring multiple considerations
- PhD students with conference deadlines
- Multi-project PIs balancing resources
- Collaborative research with dependencies

**Example**:
```bash
curl -X POST /api/v1/asba/burst-decision \
  -H "Content-Type: application/json" \
  -d '{
    "account": "nsf-ml-research",
    "estimated_aws_cost": 180.00,
    "estimated_local_time": 480,
    "job_priority": "critical",
    "conference_deadline": "2025-09-25T23:59:59Z",
    "research_phase": "validation",
    "collaboration_impact": true
  }'
```

## 🎓 Academic Research Use Cases

### Conference Submission Scenarios

#### **Case 1: ICML Deadline Approaching**
```json
{
  "estimated_aws_cost": 300.00,
  "job_priority": "critical",
  "conference_deadline": "2025-12-08T23:59:59Z",
  "research_phase": "validation"
}
```

**Expected Response**: "AWS burst strongly recommended" with high confidence due to critical deadline.

#### **Case 2: Exploratory Research**
```json
{
  "estimated_aws_cost": 150.00,
  "job_priority": "normal",
  "research_phase": "exploration"
}
```

**Expected Response**: "Local execution preferred" for cost efficiency during exploration phase.

### Grant Management Scenarios

#### **Case 3: End of Grant Period**
```json
{
  "account": "nsf-research",
  "grant_number": "NSF-2025-12345"
}
```

**Timeline Response**: Shows upcoming period end, budget preservation needs, spending recommendations.

#### **Case 4: New Grant Period**
```json
{
  "account": "nih-medical",
  "look_ahead_days": 365
}
```

**Timeline Response**: Shows full year planning with quarterly allocations and major deadlines.

## 🚨 Emergency and Critical Scenarios

### Emergency Bursting Triggers
- **Conference deadline < 1 week**: Automatic HIGH urgency
- **Grant period ending < 30 days**: Budget preservation mode
- **Collaboration dependencies**: Priority boost for team projects
- **Validation phase near deadline**: Quality over cost optimization

### Risk Mitigation Strategies
- **Budget overrun risk**: Suggest local execution, job optimization
- **Deadline risk**: Recommend AWS burst, parallel execution
- **Grant risk**: Emergency fund options, cost sharing, reallocation

## 🔗 Integration with ASBA Workflow

### 1. Pre-submission Analysis
```bash
# ASBA calls budget status before making recommendations
curl /api/v1/asba/budget-status -d '{"account": "research-proj"}'

# ASBA gets timeline context for deadline awareness
curl /api/v1/asba/grant-timeline -d '{"account": "research-proj"}'
```

### 2. Job Submission Decision
```bash
# ASBA calls comprehensive decision analysis
curl /api/v1/asba/burst-decision -d '{
  "account": "research-proj",
  "estimated_aws_cost": 200.00,
  "job_deadline": "2025-09-20T23:59:59Z"
}'

# Based on response, ASBA recommends LOCAL, AWS, or alternatives
```

### 3. Continuous Monitoring
```bash
# ASBA periodically checks budget health for ongoing recommendations
curl /api/v1/asba/budget-status -d '{"account": "research-proj"}'

# Adjusts recommendations based on changing budget health
```

## 📊 Decision Making Examples

### PhD Student Conference Submission
```bash
# Student has NIPS deadline in 5 days, needs large-scale validation
curl -X POST /api/v1/asba/burst-decision -d '{
  "account": "phd-student-research",
  "estimated_aws_cost": 400.00,
  "job_priority": "critical",
  "conference_deadline": "2025-09-19T23:59:59Z",
  "research_phase": "validation",
  "collaboration_impact": false
}'

# Response: "AWS burst immediately - critical deadline"
# Confidence: 0.95
# Urgency: CRITICAL
```

### PI Managing Multiple Projects
```bash
# PI checking overall grant health for resource allocation decisions
curl -X POST /api/v1/asba/grant-timeline -d '{
  "grant_number": "NSF-2025-12345",
  "look_ahead_days": 180,
  "include_alerts": true
}'

# Response shows upcoming deadlines across all projects
# Helps prioritize AWS usage for most critical needs
```

### Postdoc Balancing Cost and Timeline
```bash
# Postdoc with moderate deadline pressure
curl -X POST /api/v1/asba/affordability-check -d '{
  "account": "postdoc-research",
  "estimated_aws_cost": 75.00,
  "estimated_local_time": 240,
  "job_priority": "normal"
}'

# Response: Cost analysis with local vs AWS trade-offs
# Recommendations based on budget health and timeline
```

## 🔄 Integration Workflow

### Complete Academic Research Computing Workflow

1. **Research Planning**: Grant timeline API provides deadline context
2. **Job Preparation**: Budget status API checks account health
3. **Submission Decision**: Affordability check validates cost constraints
4. **Execution Choice**: Burst decision API provides final recommendation
5. **Post-execution**: ASBX integration handles cost reconciliation
6. **Continuous Monitoring**: Ongoing budget health tracking

### ASBA Decision Framework

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   ASBA Query    │───→│  ASBB Analysis  │───→│   Recommendation│
│                 │    │                 │    │                 │
│ • Job details   │    │ • Budget health │    │ • LOCAL/AWS     │
│ • Deadline info │    │ • Timeline data │    │ • Confidence    │
│ • Priority      │    │ • Grant context │    │ • Reasoning     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

This comprehensive ASBA integration enables intelligent, budget-aware, deadline-driven resource allocation decisions for academic research computing environments.