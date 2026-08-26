package application

import "manuscript-conservation-gate/internal/domain"

type WriteContext struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	ActorID          string `json:"actor_id"`
	Role             string `json:"role"`
}
type CreateCaseCommand struct {
	WriteContext
	ManuscriptCode   string                     `json:"manuscript_code"`
	Title            string                     `json:"title"`
	CustodianID      string                     `json:"custodian_id"`
	SignificanceNote string                     `json:"significance_note"`
	TreatmentGoal    string                     `json:"treatment_goal"`
	InitialRisk      string                     `json:"initial_risk"`
	RequiredRegions  []domain.RegionRequirement `json:"required_regions"`
}
type AddConditionCommand struct {
	WriteContext
	LeafRef     string `json:"leaf_ref"`
	RegionRef   string `json:"region_ref"`
	Medium      string `json:"medium"`
	DamageType  string `json:"damage_type"`
	Severity    int    `json:"severity"`
	Measurement string `json:"measurement"`
	EvidenceRef string `json:"evidence_ref"`
}
type SavePlanCommand struct {
	WriteContext
	Steps                 []domain.PlanStep `json:"steps"`
	ReversibilityNote     string            `json:"reversibility_note"`
	TracePreservationNote string            `json:"trace_preservation_note"`
	RiskControls          string            `json:"risk_controls"`
}
type TrialCommand struct {
	WriteContext
	PlanVersion  int                  `json:"plan_version"`
	MaterialCode string               `json:"material_code"`
	Protocol     string               `json:"protocol"`
	Thresholds   []domain.MetricRule  `json:"thresholds"`
	Measurements []domain.MetricValue `json:"measurements"`
	EvidenceRef  string               `json:"evidence_ref"`
}
type EthicsCommand struct {
	WriteContext
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}
type CheckpointCommand struct {
	WriteContext
	StepIndex        int                `json:"step_index"`
	ActualParameters map[string]float64 `json:"actual_parameters"`
	EvidenceRef      string             `json:"evidence_ref"`
}
type ResolveDeviationCommand struct {
	WriteContext
	Impact      string `json:"impact"`
	Remediation string `json:"remediation"`
	VerifiedBy  string `json:"verified_by"`
}
type StabilityCommand struct {
	WriteContext
	DurationHours int                  `json:"duration_hours"`
	Thresholds    []domain.MetricRule  `json:"thresholds"`
	Measurements  []domain.MetricValue `json:"measurements"`
	EvidenceRef   string               `json:"evidence_ref"`
}
type ReleaseCommand struct {
	WriteContext
	Statement string `json:"statement"`
}
