// Copyright 2025 Scott Friedman. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package api

import (
	"time"
)

// ASBA Integration Types for Academic Slurm Burst Allocation decision making

// BudgetStatusQuery represents a request for budget status information
type BudgetStatusQuery struct {
	Account     string `json:"account" validate:"required"`
	GrantNumber string `json:"grant_number,omitempty"`
	ProjectCode string `json:"project_code,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

// BudgetStatusResponse provides comprehensive budget status for decision making
type BudgetStatusResponse struct {
	Account     string `json:"account"`
	GrantNumber string `json:"grant_number,omitempty"`

	// Current budget status
	BudgetLimit       float64 `json:"budget_limit"`
	BudgetUsed        float64 `json:"budget_used"`
	BudgetHeld        float64 `json:"budget_held"`
	BudgetAvailable   float64 `json:"budget_available"`
	BudgetUtilization float64 `json:"budget_utilization"` // Percentage used

	// Grant timeline context
	GrantStartDate *time.Time `json:"grant_start_date,omitempty"`
	GrantEndDate   *time.Time `json:"grant_end_date,omitempty"`
	DaysRemaining  int        `json:"days_remaining"`

	// Burn rate and health
	DailyBurnRate     float64 `json:"daily_burn_rate"`
	ExpectedDailyRate float64 `json:"expected_daily_rate"`
	BurnRateVariance  float64 `json:"burn_rate_variance"`  // Percentage
	BudgetHealthScore float64 `json:"budget_health_score"` // 0-100
	HealthStatus      string  `json:"health_status"`       // HEALTHY, CONCERN, WARNING, CRITICAL

	// Projection and risk
	ProjectedDepletionDate *time.Time `json:"projected_depletion_date,omitempty"`
	RiskLevel              string     `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL

	// Decision making context
	CanAffordAWSBurst   bool     `json:"can_afford_aws_burst"`
	RecommendedDecision string   `json:"recommended_decision"` // PREFER_LOCAL, PREFER_AWS, EITHER, EMERGENCY_ONLY
	DecisionReasoning   []string `json:"decision_reasoning"`

	// Alerts and warnings
	ActiveAlerts []BudgetAlert `json:"active_alerts,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`

	LastUpdated time.Time `json:"last_updated"`
}

// AffordabilityCheckRequest represents a request to check if a job is affordable
type AffordabilityCheckRequest struct {
	Account            string            `json:"account" validate:"required"`
	EstimatedAWSCost   float64           `json:"estimated_aws_cost" validate:"required,min=0"`
	EstimatedLocalTime int64             `json:"estimated_local_time"` // Minutes
	JobPriority        string            `json:"job_priority,omitempty" validate:"omitempty,oneof=low normal high critical emergency"`
	JobDeadline        *time.Time        `json:"job_deadline,omitempty"`
	JobMetadata        map[string]string `json:"job_metadata,omitempty"`
}

// AffordabilityCheckResponse provides decision making guidance
type AffordabilityCheckResponse struct {
	Affordable          bool    `json:"affordable"`
	RecommendedDecision string  `json:"recommended_decision"` // LOCAL, AWS, EITHER
	ConfidenceLevel     float64 `json:"confidence_level"`     // 0.0-1.0

	// Financial analysis
	EstimatedAWSCost     float64 `json:"estimated_aws_cost"`
	BudgetImpact         float64 `json:"budget_impact"`          // Percentage of remaining budget
	CostOpportunityRatio float64 `json:"cost_opportunity_ratio"` // Cost vs time saved

	// Timeline analysis
	TimeToDeadline      int64      `json:"time_to_deadline"` // Minutes
	LocalCompletionTime *time.Time `json:"local_completion_time,omitempty"`
	AWSCompletionTime   *time.Time `json:"aws_completion_time,omitempty"`

	// Risk assessment
	BudgetRisk   string `json:"budget_risk"`   // LOW, MEDIUM, HIGH, CRITICAL
	DeadlineRisk string `json:"deadline_risk"` // LOW, MEDIUM, HIGH, CRITICAL
	OverallRisk  string `json:"overall_risk"`

	// Decision factors
	DecisionFactors map[string]interface{} `json:"decision_factors"`
	Reasoning       []string               `json:"reasoning"`
	Recommendations []string               `json:"recommendations"`

	// Alternative suggestions
	AlternativeOptions []ResourceOption `json:"alternative_options,omitempty"`

	Message string `json:"message"`
}

// ResourceOption represents an alternative resource allocation option
type ResourceOption struct {
	Option              string  `json:"option"` // LOCAL, AWS_SPOT, AWS_ONDEMAND, HYBRID
	EstimatedCost       float64 `json:"estimated_cost"`
	EstimatedTime       int64   `json:"estimated_time"` // Minutes
	Risk                string  `json:"risk"`
	Reasoning           string  `json:"reasoning"`
	RecommendationScore float64 `json:"recommendation_score"` // 0.0-1.0
}

// GrantTimelineQuery represents a request for grant timeline information
type GrantTimelineQuery struct {
	GrantNumber   string `json:"grant_number,omitempty"`
	Account       string `json:"account,omitempty"`
	LookAheadDays int    `json:"look_ahead_days"` // How far to project
	IncludeAlerts bool   `json:"include_alerts"`
}

// GrantTimelineResponse provides grant timeline and deadline information
type GrantTimelineResponse struct {
	GrantNumber string `json:"grant_number,omitempty"`
	Account     string `json:"account"`

	// Grant timeline
	GrantStartDate     time.Time `json:"grant_start_date"`
	GrantEndDate       time.Time `json:"grant_end_date"`
	CurrentPeriod      int       `json:"current_period"`
	TotalPeriods       int       `json:"total_periods"`
	PeriodEndDate      time.Time `json:"period_end_date"`
	DaysUntilPeriodEnd int       `json:"days_until_period_end"`
	DaysUntilGrantEnd  int       `json:"days_until_grant_end"`

	// Budget allocation timeline
	AllocationSchedule []AllocationEvent `json:"allocation_schedule"`
	NextAllocation     *AllocationEvent  `json:"next_allocation,omitempty"`

	// Critical deadlines
	UpcomingDeadlines []CriticalDeadline `json:"upcoming_deadlines"`

	// Budget consumption timeline
	BudgetTimeline []BudgetTimelinePoint `json:"budget_timeline"`

	// Decision making guidance
	CurrentUrgency         string `json:"current_urgency"`         // LOW, MEDIUM, HIGH, CRITICAL
	BurstingRecommendation string `json:"bursting_recommendation"` // CONSERVATIVE, NORMAL, AGGRESSIVE, EMERGENCY

	// Optimization suggestions
	OptimizationAdvice []string          `json:"optimization_advice"`
	EmergencyOptions   []EmergencyOption `json:"emergency_options,omitempty"`

	LastUpdated time.Time `json:"last_updated"`
}

// AllocationEvent represents a scheduled budget allocation
type AllocationEvent struct {
	Date        time.Time `json:"date"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // AUTOMATIC, MANUAL, EMERGENCY
	DaysFromNow int       `json:"days_from_now"`
}

// CriticalDeadline represents an important research deadline
type CriticalDeadline struct {
	Type            string    `json:"type"` // CONFERENCE, GRANT_REPORT, PERIOD_END, RENEWAL
	Description     string    `json:"description"`
	Date            time.Time `json:"date"`
	DaysFromNow     int       `json:"days_from_now"`
	Severity        string    `json:"severity"`      // LOW, MEDIUM, HIGH, CRITICAL
	BudgetImpact    string    `json:"budget_impact"` // Description of how this affects budget
	Recommendations []string  `json:"recommendations"`
}

// BudgetTimelinePoint represents budget usage at a specific point in time
type BudgetTimelinePoint struct {
	Date               time.Time `json:"date"`
	CumulativeSpend    float64   `json:"cumulative_spend"`
	CumulativeExpected float64   `json:"cumulative_expected"`
	RemainingBudget    float64   `json:"remaining_budget"`
	BurnRateStatus     string    `json:"burn_rate_status"`
	HealthScore        float64   `json:"health_score"`
}

// EmergencyOption represents emergency budget options for critical deadlines
type EmergencyOption struct {
	Option        string   `json:"option"` // REALLOCATE, EMERGENCY_FUND, COST_SHARING, DEADLINE_EXTENSION
	Description   string   `json:"description"`
	EstimatedCost float64  `json:"estimated_cost,omitempty"`
	Timeline      string   `json:"timeline"`
	Requirements  []string `json:"requirements"`
	RiskLevel     string   `json:"risk_level"`
}

// BurstDecisionRequest represents a request for burst decision making
type BurstDecisionRequest struct {
	Account             string            `json:"account" validate:"required"`
	EstimatedAWSCost    float64           `json:"estimated_aws_cost" validate:"required,min=0"`
	EstimatedLocalTime  int64             `json:"estimated_local_time"` // Minutes
	JobPriority         string            `json:"job_priority,omitempty"`
	JobDeadline         *time.Time        `json:"job_deadline,omitempty"`
	ConferenceDeadline  *time.Time        `json:"conference_deadline,omitempty"`
	ResearchPhase       string            `json:"research_phase,omitempty"` // EXPLORATION, DEVELOPMENT, VALIDATION, PUBLICATION
	CollaborationImpact bool              `json:"collaboration_impact"`     // Affects other researchers
	JobMetadata         map[string]string `json:"job_metadata,omitempty"`
}

// BurstDecisionResponse provides intelligent bursting recommendations
type BurstDecisionResponse struct {
	RecommendedAction string  `json:"recommended_action"` // LOCAL, AWS, DEFER, OPTIMIZE
	Confidence        float64 `json:"confidence"`         // 0.0-1.0
	UrgencyLevel      string  `json:"urgency_level"`      // LOW, MEDIUM, HIGH, CRITICAL

	// Financial analysis
	BudgetImpact       float64 `json:"budget_impact"`
	AffordabilityScore float64 `json:"affordability_score"` // 0.0-1.0
	CostEfficiency     float64 `json:"cost_efficiency"`     // Cost per hour saved

	// Timeline analysis
	TimelinePressure float64 `json:"timeline_pressure"` // 0.0-1.0
	DeadlineRisk     string  `json:"deadline_risk"`

	// Grant context
	GrantHealthImpact  string  `json:"grant_health_impact"`
	BudgetPreservation float64 `json:"budget_preservation"` // How much budget to preserve

	// Decision reasoning
	DecisionFactors []DecisionFactor `json:"decision_factors"`
	RiskAssessment  RiskAssessment   `json:"risk_assessment"`
	Alternatives    []ResourceOption `json:"alternatives"`

	// Actionable guidance
	ImmediateActions    []string `json:"immediate_actions"`
	LongtermSuggestions []string `json:"longterm_suggestions"`

	Message string `json:"message"`
}

// DecisionFactor represents a factor in the bursting decision
type DecisionFactor struct {
	Factor      string  `json:"factor"`
	Weight      float64 `json:"weight"` // 0.0-1.0
	Value       float64 `json:"value"`  // Normalized 0.0-1.0
	Impact      string  `json:"impact"` // POSITIVE, NEGATIVE, NEUTRAL
	Description string  `json:"description"`
}

// RiskAssessment provides comprehensive risk analysis
type RiskAssessment struct {
	OverallRisk          string   `json:"overall_risk"`
	BudgetRisk           string   `json:"budget_risk"`
	DeadlineRisk         string   `json:"deadline_risk"`
	GrantRisk            string   `json:"grant_risk"`
	RiskFactors          []string `json:"risk_factors"`
	MitigationStrategies []string `json:"mitigation_strategies"`
	ConfidenceLevel      float64  `json:"confidence_level"`
}
