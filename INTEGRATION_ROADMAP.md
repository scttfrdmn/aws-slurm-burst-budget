# Integration Roadmap: Sister Project Requirements

Based on analysis of issues from [aws-slurm-burst-advisor](https://github.com/scttfrdmn/aws-slurm-burst-advisor) and [aws-slurm-burst](https://github.com/scttfrdmn/aws-slurm-burst), this document outlines key integration improvements needed for the AWS SLURM Bursting Budget system.

## 🔍 Sister Project Issues Analysis

### aws-slurm-burst-advisor Issues
- **Ecosystem Integration**: ASBX and ASBB coordination requirements
- **Budget Bank Integration**: Grant management features (Phase 3 priority)
- **Performance Learning**: Learning from execution data for better cost models
- **Environment Variables**: Shell-friendly workflow integration

### aws-slurm-burst Issues
- **Enhanced Metadata**: Better SLURM accounting metadata for integration
- **MPI Performance**: Performance profiling affects cost calculations
- **Domain-Specific Optimization**: Research-specific optimization profiles

## 🎯 Required Enhancements for ASBB

### 1. Enhanced Integration APIs

**Priority**: High
**Issue Reference**: "Ecosystem Integration: ASBX and ASBB Coordination"

**Implementation Needed**:
```go
// Add to pkg/api/types.go
type EcosystemIntegrationRequest struct {
    JobID           string                 `json:"job_id"`
    PerformanceData map[string]interface{} `json:"performance_data"`
    CostData        map[string]interface{} `json:"cost_data"`
    Metadata        map[string]interface{} `json:"metadata"`
}

// Add to REST API
POST /api/v1/integration/performance-feedback
POST /api/v1/integration/cost-update
GET  /api/v1/integration/budget-context/{job_id}
```

### 2. Grant Management Features

**Priority**: Critical
**Issue Reference**: "Phase 3 Priority: ASBB (Budget Bank) Integration for Grant Management"

**Implementation Needed**:
```sql
-- Add to migrations/003_grant_management.up.sql
CREATE TABLE grant_accounts (
    id BIGSERIAL PRIMARY KEY,
    grant_number VARCHAR(128) NOT NULL UNIQUE,
    funding_agency VARCHAR(255) NOT NULL,
    principal_investigator VARCHAR(255) NOT NULL,
    grant_start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    grant_end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    total_award_amount DECIMAL(12,2) NOT NULL,
    indirect_cost_rate DECIMAL(5,4), -- e.g., 0.30 for 30%
    budget_account_id BIGINT REFERENCES budget_accounts(id)
);
```

**CLI Commands Needed**:
```bash
asbb grant create --number=NSF-12345 --agency="NSF" --pi="Dr. Smith" --amount=250000
asbb grant list --active
asbb grant report --number=NSF-12345 --format=agency-required
```

### 3. Enhanced Job Metadata Support

**Priority**: High
**Issue Reference**: "Enhanced SLURM Accounting Metadata for Integration"

**Implementation Needed**:
```go
// Expand BudgetCheckRequest
type BudgetCheckRequest struct {
    // ... existing fields ...

    // Enhanced metadata for integration
    JobMetadata struct {
        ResearchDomain     string            `json:"research_domain,omitempty"`
        MPIProfile         string            `json:"mpi_profile,omitempty"`
        PerformanceHints   map[string]string `json:"performance_hints,omitempty"`
        CostConstraints    map[string]float64 `json:"cost_constraints,omitempty"`
        GrantNumber        string            `json:"grant_number,omitempty"`
        ProjectCode        string            `json:"project_code,omitempty"`
    } `json:"job_metadata,omitempty"`
}
```

### 4. Performance-Based Cost Learning

**Priority**: Medium
**Issue Reference**: "Performance Learning from aws-slurm-burst Execution Data"

**Implementation Needed**:
```go
// Add to internal/budget/service.go
type PerformanceFeedback struct {
    JobID              string  `json:"job_id"`
    EstimatedCost      float64 `json:"estimated_cost"`
    ActualCost         float64 `json:"actual_cost"`
    EstimatedDuration  int64   `json:"estimated_duration"`
    ActualDuration     int64   `json:"actual_duration"`
    PerformanceRating  float64 `json:"performance_rating"`
    OptimizationHints  string  `json:"optimization_hints"`
}

func (s *Service) ProcessPerformanceFeedback(ctx context.Context, feedback *PerformanceFeedback) error {
    // Update cost estimation accuracy
    // Feed data back to advisor service
    // Improve future estimates
}
```

### 5. Environment Variable & Shell Integration

**Priority**: Medium
**Issue Reference**: "Environment Variable Integration for Shell-friendly Workflows"

**Implementation Needed**:
```bash
# Add environment variable support for common operations
export ASBB_ACCOUNT=research-proj-001
export ASBB_DEFAULT_PARTITION=gpu-aws

# Shell-friendly commands
asbb quick-check --nodes=4 --cpus=16 --time=2h  # Uses env vars
asbb status                                       # Shows current account status
asbb remaining                                    # Shows remaining budget
```

## 🚀 Implementation Priority

### Phase 1 (v0.2.0) - Core Integration
1. **Enhanced Metadata Support**: Expand job metadata fields
2. **Grant Management**: Basic grant account support
3. **Integration APIs**: Cross-system coordination endpoints

### Phase 2 (v0.3.0) - Intelligence
1. **Performance Learning**: Cost model improvements
2. **Advanced Reporting**: Grant-compliant reporting
3. **Shell Integration**: Environment variable support

### Phase 3 (v0.4.0) - Advanced Features
1. **Domain-Specific Profiles**: Research area optimizations
2. **Predictive Budgeting**: ML-based budget forecasting
3. **Advanced Analytics**: Cross-project cost analysis

## 🔗 Integration Points Needed

### With aws-slurm-burst-advisor
- **Bidirectional Feedback**: Send actual costs back to improve estimates
- **Metadata Sharing**: Pass budget constraints to advisor
- **Performance Context**: Share budget utilization for better decisions

### With aws-slurm-burst
- **Job Lifecycle Tracking**: Better reconciliation data
- **Performance Metrics**: Use actual performance for cost model learning
- **Metadata Enhancement**: Rich job context for cost attribution

## 📋 Immediate Action Items

1. **Add grant management database schema**
2. **Implement enhanced metadata API endpoints**
3. **Create integration endpoints for cross-system communication**
4. **Add environment variable support for CLI**
5. **Implement performance feedback learning system**

This roadmap ensures our budget system evolves to meet the ecosystem needs identified in the sister projects.