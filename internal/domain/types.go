package domain

import "time"

type CaseState string

const (
	StateDraft          CaseState = "draft"
	StateBaselineLocked CaseState = "baseline_locked"
	StatePlanDraft      CaseState = "plan_draft"
	StateTrialPassed    CaseState = "trial_passed"
	StateApproved       CaseState = "approved"
	StateTreating       CaseState = "treating"
	StatePaused         CaseState = "paused"
	StateTreated        CaseState = "treated"
	StateStable         CaseState = "stable"
	StateReleased       CaseState = "released"
	StateArchived       CaseState = "archived"
)

type ConservationCase struct {
	ID                    string                 `json:"id"`
	ManuscriptCode        string                 `json:"manuscript_code"`
	Title                 string                 `json:"title"`
	CustodianID           string                 `json:"custodian_id"`
	SignificanceNote      string                 `json:"significance_note"`
	TreatmentGoal         string                 `json:"treatment_goal"`
	InitialRisk           string                 `json:"initial_risk"`
	RequiredRegions       []RegionRequirement    `json:"required_regions"`
	State                 CaseState              `json:"state"`
	Revision              int64                  `json:"revision"`
	CreatedAt             time.Time              `json:"created_at"`
	ArchivedAt            *time.Time             `json:"archived_at,omitempty"`
	Conditions            []ConditionObservation `json:"conditions"`
	Plans                 []TreatmentPlan        `json:"plans"`
	Trials                []CompatibilityTrial   `json:"trials"`
	Checkpoints           []TreatmentCheckpoint  `json:"checkpoints"`
	StabilityObservations []StabilityObservation `json:"stability_observations"`
	Release               *ReleaseSignature      `json:"release,omitempty"`
	Archive               *ArchiveManifest       `json:"archive,omitempty"`
}

type RegionRequirement struct {
	LeafRef   string `json:"leaf_ref"`
	RegionRef string `json:"region_ref"`
}
type ConditionObservation struct {
	ID          string    `json:"id"`
	CaseID      string    `json:"case_id"`
	LeafRef     string    `json:"leaf_ref"`
	RegionRef   string    `json:"region_ref"`
	Medium      string    `json:"medium"`
	DamageType  string    `json:"damage_type"`
	Severity    int       `json:"severity"`
	Measurement string    `json:"measurement"`
	EvidenceRef string    `json:"evidence_ref"`
	RecordedBy  string    `json:"recorded_by"`
	RecordedAt  time.Time `json:"recorded_at"`
}
type PlanStep struct {
	Index          int                `json:"index"`
	Purpose        string             `json:"purpose"`
	Material       string             `json:"material"`
	Parameters     map[string]float64 `json:"parameters"`
	Tolerances     map[string]float64 `json:"tolerances"`
	Reversibility  string             `json:"reversibility"`
	StopCondition  string             `json:"stop_condition"`
	RiskMitigation string             `json:"risk_mitigation"`
}
type TreatmentPlan struct {
	ID                    string     `json:"id"`
	CaseID                string     `json:"case_id"`
	Version               int        `json:"version"`
	Steps                 []PlanStep `json:"steps"`
	ReversibilityNote     string     `json:"reversibility_note"`
	TracePreservationNote string     `json:"trace_preservation_note"`
	RiskControls          string     `json:"risk_controls"`
	Status                string     `json:"status"`
	SubmittedAt           *time.Time `json:"submitted_at,omitempty"`
	ReviewerID            string     `json:"reviewer_id,omitempty"`
	ReviewDecision        string     `json:"review_decision,omitempty"`
	ReviewReason          string     `json:"review_reason,omitempty"`
	ReviewedAt            *time.Time `json:"reviewed_at,omitempty"`
}
type MetricRule struct {
	Name string   `json:"name"`
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
}
type MetricValue struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
type CompatibilityTrial struct {
	ID           string        `json:"id"`
	CaseID       string        `json:"case_id"`
	PlanVersion  int           `json:"plan_version"`
	MaterialCode string        `json:"material_code"`
	Protocol     string        `json:"protocol"`
	Thresholds   []MetricRule  `json:"thresholds"`
	Measurements []MetricValue `json:"measurements"`
	Outcome      string        `json:"outcome"`
	Failures     []string      `json:"failures,omitempty"`
	EvidenceRef  string        `json:"evidence_ref"`
	ObservedAt   time.Time     `json:"observed_at"`
}
type TreatmentCheckpoint struct {
	ID               string             `json:"id"`
	CaseID           string             `json:"case_id"`
	StepIndex        int                `json:"step_index"`
	OperatorID       string             `json:"operator_id"`
	ActualParameters map[string]float64 `json:"actual_parameters"`
	Outcome          string             `json:"outcome"`
	DeviationNote    string             `json:"deviation_note,omitempty"`
	Remediation      string             `json:"remediation,omitempty"`
	VerifiedBy       string             `json:"verified_by,omitempty"`
	EvidenceRef      string             `json:"evidence_ref"`
	CompletedAt      time.Time          `json:"completed_at"`
}
type StabilityObservation struct {
	ID            string        `json:"id"`
	ObserverID    string        `json:"observer_id"`
	DurationHours int           `json:"duration_hours"`
	Thresholds    []MetricRule  `json:"thresholds"`
	Measurements  []MetricValue `json:"measurements"`
	Outcome       string        `json:"outcome"`
	Failures      []string      `json:"failures,omitempty"`
	EvidenceRef   string        `json:"evidence_ref"`
	ObservedAt    time.Time     `json:"observed_at"`
}
type ReleaseSignature struct {
	ReviewerID string    `json:"reviewer_id"`
	Statement  string    `json:"statement"`
	SignedAt   time.Time `json:"signed_at"`
}
type ArchiveManifest struct {
	CaseID          string            `json:"case_id"`
	CaseRevision    int64             `json:"case_revision"`
	GeneratedAt     time.Time         `json:"generated_at"`
	DocumentDigests map[string]string `json:"document_digests"`
	EventCount      int64             `json:"event_count"`
	AuditChainHead  string            `json:"audit_chain_head"`
	ManifestDigest  string            `json:"manifest_digest"`
	VerifiedAt      *time.Time        `json:"verified_at,omitempty"`
}
type AuditEvent struct {
	Sequence     int64     `json:"sequence"`
	CaseID       string    `json:"case_id"`
	Revision     int64     `json:"revision"`
	Type         string    `json:"type"`
	ActorID      string    `json:"actor_id"`
	OccurredAt   time.Time `json:"occurred_at"`
	Payload      []byte    `json:"payload"`
	PreviousHash string    `json:"previous_hash"`
	Hash         string    `json:"hash"`
}
