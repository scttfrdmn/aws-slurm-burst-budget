# Operational Independence Design

This document outlines the operational independence architecture ensuring each component in the ASBA + ASBX + ASBB ecosystem can run standalone while providing enhanced functionality when others are available.

## 🎯 Design Principle

**Each service must be fully operational on its own, with others providing enhancement rather than dependency.**

## 🏗️ Independence Architecture

### Core Components

#### **ASBB (This Service) - Budget Management**
**Standalone Capabilities:**
- ✅ **Budget account management** (accounts, limits, transactions)
- ✅ **Real-time budget enforcement** (SLURM plugin budget checks)
- ✅ **Grant management** (multi-year grants, budget periods)
- ✅ **Burn rate analytics** (spending patterns, variance tracking)
- ✅ **Incremental allocations** (scheduled budget releases)
- ✅ **Cost reconciliation** (manual or automated)

**Enhanced with Integrations:**
- 🔗 **ASBX Available**: Automatic cost reconciliation from performance data
- 🔗 **ASBA Available**: Intelligent decision making for resource allocation
- 🔗 **Advisor Available**: Improved cost estimation accuracy

#### **Operational Modes**

**1. Standalone Mode** (Database only)
```yaml
integration:
  advisor_enabled: false
  asbx_enabled: false
  asba_enabled: false
  failure_mode: "GRACEFUL"
```

**Features Available:**
- Budget enforcement with fallback cost estimation
- Grant management and burn rate analytics
- Manual cost reconciliation
- All CLI commands and REST APIs

**2. Partially Integrated Mode** (Some services available)
```yaml
integration:
  advisor_enabled: true    # Enhanced cost estimation
  asbx_enabled: false     # Manual reconciliation
  asba_enabled: false     # Basic decision making
  failure_mode: "GRACEFUL"
```

**3. Fully Integrated Mode** (Complete ecosystem)
```yaml
integration:
  advisor_enabled: true   # Accurate cost estimation
  asbx_enabled: true     # Automatic reconciliation
  asba_enabled: true     # Intelligent decisions
  failure_mode: "GRACEFUL"
```

## 🔄 Graceful Degradation Patterns

### Cost Estimation Fallback

**Primary**: aws-slurm-burst-advisor service
**Fallback**: Simple heuristic-based estimation

```go
// If advisor unavailable, use built-in estimation
costResp, err := advisorClient.EstimateCost(ctx, req)
if err != nil {
    log.Warn().Msg("Advisor unavailable, using fallback estimation")
    costResp = service.fallbackCostEstimate(req)
}
```

**Fallback Logic:**
- CPU cost: `nodes × CPUs × $0.10/hour × duration`
- GPU premium: `GPUs × $2.00/hour × duration`
- Partition multipliers: GPU (2x), AWS (1.5x), Debug (0.5x)
- Confidence: 0.6 (vs 0.9+ from advisor)

### Integration API Graceful Responses

**ASBX Integration Endpoints**:
- ✅ **Always available** - return appropriate status
- ⚠️ **Service unavailable**: Return "integration not configured" with guidance
- 🔄 **Partial functionality**: Basic responses without advanced features

**ASBA Integration Endpoints**:
- ✅ **Always available** - provide basic decision making
- 🔗 **With advisor**: Enhanced cost analysis
- 📊 **With grants**: Timeline and deadline awareness

### Health Check Transparency

```json
GET /health

{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "advisor": "unavailable",  // ← Transparent about service status
    "asbx": "not_configured",
    "asba": "not_configured"
  },
  "operational_mode": "standalone"  // ← Clear operational state
}
```

## 🔧 Configuration Examples

### Minimal Standalone Configuration
```yaml
# Minimal config - only database required
service:
  listen_addr: ":8080"

database:
  driver: "postgres"
  dsn: "postgresql://user:pass@localhost/asbb"

# All integrations disabled by default
integration:
  advisor_enabled: false
  asbx_enabled: false
  asba_enabled: false
  failure_mode: "GRACEFUL"
```

### Graceful Integration Configuration
```yaml
# Enhanced functionality with graceful fallbacks
integration:
  # Advisor integration with fallback
  advisor_enabled: true
  advisor_fallback: "SIMPLE"  # Use simple estimation if unavailable
  fallback_cost_rate: 0.15    # $0.15/CPU-hour fallback rate

  # ASBX integration (optional)
  asbx_enabled: true
  asbx_endpoint: "http://asbx-service:8082"

  # Graceful degradation settings
  failure_mode: "GRACEFUL"    # Continue operating without failed services
  retry_attempts: 3
  health_check_interval: "60s"
```

### Strict Mode Configuration
```yaml
# Fail fast if required services unavailable
integration:
  advisor_enabled: true
  failure_mode: "STRICT"     # Fail if advisor unavailable

  # Optional services still graceful
  asbx_enabled: true
  asba_enabled: false
```

## 📊 API Behavior by Mode

### Budget Check API (`POST /budget/check`)

**Standalone Mode:**
```json
{
  "available": true,
  "estimated_cost": 12.50,      // ← Fallback estimation
  "hold_amount": 15.00,
  "confidence": 0.6,            // ← Lower confidence
  "recommendation": "Fallback cost estimate - advisor unavailable"
}
```

**Integrated Mode:**
```json
{
  "available": true,
  "estimated_cost": 11.75,      // ← Advisor estimation
  "hold_amount": 14.10,
  "confidence": 0.92,           // ← High confidence
  "recommendation": "Optimal resource allocation for this workload"
}
```

### ASBX Integration APIs

**When ASBX Unavailable:**
```json
POST /asbx/reconcile

{
  "success": false,
  "message": "ASBX integration not available",
  "alternatives": [
    "Use manual reconciliation: asbb reconcile <job_id>",
    "Enable ASBX integration in configuration",
    "Check ASBX service availability"
  ]
}
```

**When ASBX Available:**
```json
{
  "success": true,
  "reconciliation_id": "asbx_recon_12345",
  "cost_variance": -5.25,
  "model_update_applied": true
}
```

## 🚀 Deployment Strategies

### 1. Progressive Rollout
```bash
# Phase 1: Deploy ASBB standalone
helm install asbb-budget ./charts/asbb --set integrations.enabled=false

# Phase 2: Add advisor integration
helm upgrade asbb-budget ./charts/asbb --set advisor.enabled=true

# Phase 3: Enable full ecosystem
helm upgrade asbb-budget ./charts/asbb --set integrations.all=true
```

### 2. Multi-Environment Deployment
```bash
# Development: Standalone mode
export ASBB_INTEGRATION_ADVISOR_ENABLED=false

# Staging: Partial integration
export ASBB_INTEGRATION_ADVISOR_ENABLED=true
export ASBB_INTEGRATION_ASBX_ENABLED=false

# Production: Full integration
export ASBB_INTEGRATION_ADVISOR_ENABLED=true
export ASBB_INTEGRATION_ASBX_ENABLED=true
export ASBB_INTEGRATION_ASBA_ENABLED=true
```

### 3. Disaster Recovery
```bash
# If advisor service fails, continue with fallback
ASBB_INTEGRATION_FAILURE_MODE=GRACEFUL

# Budget service continues operating with:
# - Fallback cost estimation
# - Manual reconciliation
# - Basic decision making
# - All grant management features
```

## ✅ Benefits of Operational Independence

### **Reliability**
- **No single point of failure** - budget enforcement always works
- **Graceful degradation** - reduced functionality vs complete failure
- **Independent scaling** - scale each service based on load

### **Maintainability**
- **Independent development** - update services without coordinating releases
- **Simplified testing** - test each service in isolation
- **Clear boundaries** - well-defined interfaces between services

### **Flexibility**
- **Gradual adoption** - organizations can adopt services incrementally
- **Mix and match** - use only needed components
- **Vendor independence** - no lock-in to specific service combinations

### **Operations**
- **Independent monitoring** - health checks per service
- **Isolated incidents** - advisor failure doesn't break budget enforcement
- **Flexible deployment** - different services in different environments

## 🎯 Academic Research Benefits

### **PhD Students**
- Budget enforcement works regardless of other services
- Can use grant management without complex setup
- Fallback cost estimation for quick decisions

### **System Administrators**
- Deploy budget service immediately for SLURM integration
- Add enhanced features (advisor, ASBX) when ready
- No "big bang" deployment required

### **Research Institutions**
- Start with basic budget management
- Enhance with intelligent decision making over time
- Maintain operations during service maintenance

This operational independence design ensures robust, flexible, and maintainable academic research computing budget management.