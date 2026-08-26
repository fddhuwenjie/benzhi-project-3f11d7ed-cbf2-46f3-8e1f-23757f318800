package domain

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"
)

type CoverageRegion struct {
	LeafRef     string `json:"leaf_ref"`
	RegionRef   string `json:"region_ref"`
	Recorded    bool   `json:"recorded"`
	ConditionID string `json:"condition_id,omitempty"`
}

type DataAnomaly struct {
	Field       string `json:"field"`
	ConditionID string `json:"condition_id,omitempty"`
	Type        string `json:"type"`
	Message     string `json:"message"`
}

type SeveritySummary struct {
	LeafRef      string        `json:"leaf_ref"`
	RegionRef    string        `json:"region_ref"`
	DamageType   string        `json:"damage_type"`
	Counts       map[int]int   `json:"counts"`
	HighestLevel int           `json:"highest_level"`
	Anomalies    []DataAnomaly `json:"anomalies"`
}

type CoverageReport struct {
	CaseID             string              `json:"case_id"`
	Revision           int64               `json:"revision"`
	State              CaseState           `json:"state"`
	ReadOnly           bool                `json:"read_only"`
	RequiredCount      int                 `json:"required_count"`
	RecordedCount      int                 `json:"recorded_count"`
	MissingCount       int                 `json:"missing_count"`
	CoveragePercentage float64             `json:"coverage_percentage"`
	Regions            []CoverageRegion    `json:"regions"`
	Missing            []RegionRequirement `json:"missing"`
	Severity           []SeveritySummary   `json:"severity"`
	Anomalies          []DataAnomaly       `json:"anomalies"`
}

func BuildCoverageReport(c *ConservationCase) CoverageReport {
	report := CoverageReport{CaseID: c.ID, Revision: c.Revision, State: c.State, ReadOnly: c.State != StateDraft, RequiredCount: len(c.RequiredRegions), Regions: []CoverageRegion{}, Missing: []RegionRequirement{}, Severity: []SeveritySummary{}, Anomalies: []DataAnomaly{}}
	byRegion := map[string][]ConditionObservation{}
	for _, o := range c.Conditions {
		key := o.LeafRef + "\x00" + o.RegionRef
		byRegion[key] = append(byRegion[key], o)
	}
	for _, required := range c.RequiredRegions {
		items := byRegion[required.LeafRef+"\x00"+required.RegionRef]
		region := CoverageRegion{LeafRef: required.LeafRef, RegionRef: required.RegionRef, Recorded: len(items) > 0}
		if len(items) > 0 {
			region.ConditionID = items[0].ID
			report.RecordedCount++
		} else {
			report.Missing = append(report.Missing, required)
		}
		report.Regions = append(report.Regions, region)
		if len(items) > 1 {
			report.Anomalies = append(report.Anomalies, DataAnomaly{Field: "conditions", Type: "duplicate_region", Message: "同一必填区域存在重复状况记录"})
		}
	}
	report.MissingCount = len(report.Missing)
	if report.RequiredCount > 0 {
		report.CoveragePercentage = math.Round(float64(report.RecordedCount)*1000/float64(report.RequiredCount)) / 10
	}
	type severityKey struct{ leaf, region, damage string }
	severity := map[severityKey]*SeveritySummary{}
	for i, o := range c.Conditions {
		key := severityKey{o.LeafRef, o.RegionRef, o.DamageType}
		entry := severity[key]
		if entry == nil {
			entry = &SeveritySummary{LeafRef: o.LeafRef, RegionRef: o.RegionRef, DamageType: o.DamageType, Counts: map[int]int{}, Anomalies: []DataAnomaly{}}
			severity[key] = entry
		}
		entry.Counts[o.Severity]++
		if o.Severity > entry.HighestLevel {
			entry.HighestLevel = o.Severity
		}
		if strings.TrimSpace(o.Measurement) == "" {
			a := DataAnomaly{Field: fmt.Sprintf("conditions[%d].measurement", i), ConditionID: o.ID, Type: "missing_measurement", Message: "缺少 measurement"}
			entry.Anomalies = append(entry.Anomalies, a)
			report.Anomalies = append(report.Anomalies, a)
		}
		if err := ValidateEvidenceRef(o.EvidenceRef); err != nil {
			a := DataAnomaly{Field: fmt.Sprintf("conditions[%d].evidence_ref", i), ConditionID: o.ID, Type: "invalid_evidence_ref", Message: "evidence_ref 无效"}
			entry.Anomalies = append(entry.Anomalies, a)
			report.Anomalies = append(report.Anomalies, a)
		}
	}
	for _, entry := range severity {
		report.Severity = append(report.Severity, *entry)
	}
	sort.Slice(report.Severity, func(i, j int) bool {
		a, b := report.Severity[i], report.Severity[j]
		if a.LeafRef != b.LeafRef {
			return a.LeafRef < b.LeafRef
		}
		if a.RegionRef != b.RegionRef {
			return a.RegionRef < b.RegionRef
		}
		return a.DamageType < b.DamageType
	})
	return report
}

func ValidateBaselineCoverage(c *ConservationCase) error {
	report := BuildCoverageReport(c)
	if len(report.Missing) > 0 {
		missing := report.Missing[0]
		return Invalid("required_regions", "必填区域 "+missing.LeafRef+"/"+missing.RegionRef+" 尚未记录")
	}
	for _, anomaly := range report.Anomalies {
		if anomaly.Type == "duplicate_region" || anomaly.Type == "invalid_evidence_ref" {
			return Invalid(anomaly.Field, anomaly.Message)
		}
	}
	return nil
}

type PlanHistoryItem struct {
	Version        int        `json:"version"`
	Status         string     `json:"status"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	ReviewDecision string     `json:"review_decision,omitempty"`
	ReviewReason   string     `json:"review_reason,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	Current        bool       `json:"current"`
	Approved       bool       `json:"approved"`
}

type PlanHistoryReport struct {
	CaseID          string            `json:"case_id"`
	Revision        int64             `json:"revision"`
	CurrentVersion  int               `json:"current_version,omitempty"`
	ApprovedVersion int               `json:"approved_version,omitempty"`
	Items           []PlanHistoryItem `json:"items"`
}

func BuildPlanHistory(c *ConservationCase) PlanHistoryReport {
	report := PlanHistoryReport{CaseID: c.ID, Revision: c.Revision, Items: []PlanHistoryItem{}}
	if current := c.CurrentPlan(); current != nil {
		report.CurrentVersion = current.Version
	}
	if approved := c.ApprovedPlan(); approved != nil {
		report.ApprovedVersion = approved.Version
	}
	for _, p := range c.Plans {
		decision := p.ReviewDecision
		if decision == "" && p.Status == "approved" {
			decision = "approve"
		} else if decision == "" && p.Status == "returned" {
			decision = "return"
		}
		report.Items = append(report.Items, PlanHistoryItem{Version: p.Version, Status: p.Status, SubmittedAt: p.SubmittedAt, ReviewDecision: decision, ReviewReason: p.ReviewReason, ReviewedAt: p.ReviewedAt, Current: p.Version == report.CurrentVersion, Approved: p.Version == report.ApprovedVersion})
	}
	return report
}

type PlanFieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

type PlanStepDiff struct {
	StepIndex int               `json:"step_index"`
	Changes   []PlanFieldChange `json:"changes"`
}

type PlanDiffReport struct {
	CaseID      string         `json:"case_id"`
	Revision    int64          `json:"revision"`
	FromVersion int            `json:"from_version"`
	ToVersion   int            `json:"to_version"`
	Steps       []PlanStepDiff `json:"steps"`
}

func BuildPlanDiff(c *ConservationCase, version int) (PlanDiffReport, error) {
	if version < 2 || version > len(c.Plans) {
		return PlanDiffReport{}, Invalid("version", "必须指定存在且具有前一版本的方案版本")
	}
	before, after := c.Plans[version-2], c.Plans[version-1]
	report := PlanDiffReport{CaseID: c.ID, Revision: c.Revision, FromVersion: before.Version, ToVersion: after.Version, Steps: []PlanStepDiff{}}
	maxSteps := len(before.Steps)
	if len(after.Steps) > maxSteps {
		maxSteps = len(after.Steps)
	}
	for i := 0; i < maxSteps; i++ {
		changes := []PlanFieldChange{}
		var oldStep, newStep *PlanStep
		if i < len(before.Steps) {
			oldStep = &before.Steps[i]
		}
		if i < len(after.Steps) {
			newStep = &after.Steps[i]
		}
		if oldStep == nil || newStep == nil {
			changes = append(changes, PlanFieldChange{Field: "step", Before: oldStep, After: newStep})
		} else {
			appendChange := func(field string, old, next any) {
				if !reflect.DeepEqual(old, next) {
					changes = append(changes, PlanFieldChange{Field: field, Before: old, After: next})
				}
			}
			appendChange("purpose", oldStep.Purpose, newStep.Purpose)
			appendChange("material", oldStep.Material, newStep.Material)
			appendChange("parameters", oldStep.Parameters, newStep.Parameters)
			appendChange("tolerances", oldStep.Tolerances, newStep.Tolerances)
			appendChange("stop_condition", oldStep.StopCondition, newStep.StopCondition)
			appendChange("risk_mitigation", oldStep.RiskMitigation, newStep.RiskMitigation)
		}
		if len(changes) > 0 {
			report.Steps = append(report.Steps, PlanStepDiff{StepIndex: i + 1, Changes: changes})
		}
	}
	return report, nil
}

type MetricFailureDetail struct {
	Name   string   `json:"name"`
	Value  float64  `json:"value"`
	Min    *float64 `json:"min,omitempty"`
	Max    *float64 `json:"max,omitempty"`
	Reason string   `json:"reason"`
}

func ExplainMetricFailures(rules []MetricRule, values []MetricValue) []MetricFailureDetail {
	measured := map[string]float64{}
	for _, value := range values {
		measured[value.Name] = value.Value
	}
	out := []MetricFailureDetail{}
	for _, rule := range rules {
		value, ok := measured[rule.Name]
		if !ok {
			continue
		}
		if rule.Min != nil && value < *rule.Min {
			out = append(out, MetricFailureDetail{Name: rule.Name, Value: value, Min: rule.Min, Max: rule.Max, Reason: "低于下限"})
		}
		if rule.Max != nil && value > *rule.Max {
			out = append(out, MetricFailureDetail{Name: rule.Name, Value: value, Min: rule.Min, Max: rule.Max, Reason: "高于上限"})
		}
	}
	return out
}

type TrialAttempt struct {
	ID          string                `json:"id"`
	ObservedAt  time.Time             `json:"observed_at"`
	Outcome     string                `json:"outcome"`
	EvidenceRef string                `json:"evidence_ref"`
	Failures    []MetricFailureDetail `json:"failures"`
}

type TrialSummary struct {
	PlanVersion      int            `json:"plan_version"`
	MaterialCode     string         `json:"material_code"`
	TotalCount       int            `json:"total_count"`
	PassedCount      int            `json:"passed_count"`
	FailedCount      int            `json:"failed_count"`
	LatestObservedAt time.Time      `json:"latest_observed_at"`
	GateStatus       string         `json:"gate_status"`
	Expired          bool           `json:"expired"`
	Attempts         []TrialAttempt `json:"attempts"`
}

type TrialReport struct {
	CaseID             string         `json:"case_id"`
	Revision           int64          `json:"revision"`
	CurrentPlanVersion int            `json:"current_plan_version,omitempty"`
	CurrentGateStatus  string         `json:"current_gate_status"`
	Items              []TrialSummary `json:"items"`
}

func BuildTrialReport(c *ConservationCase) TrialReport {
	report := TrialReport{CaseID: c.ID, Revision: c.Revision, CurrentGateStatus: "blocked", Items: []TrialSummary{}}
	if p := c.CurrentPlan(); p != nil {
		report.CurrentPlanVersion = p.Version
	}
	type key struct {
		version  int
		material string
	}
	groups := map[key]*TrialSummary{}
	trials := append([]CompatibilityTrial(nil), c.Trials...)
	sort.SliceStable(trials, func(i, j int) bool {
		if trials[i].ObservedAt.Equal(trials[j].ObservedAt) {
			return trials[i].ID < trials[j].ID
		}
		return trials[i].ObservedAt.Before(trials[j].ObservedAt)
	})
	for _, trial := range trials {
		k := key{trial.PlanVersion, trial.MaterialCode}
		item := groups[k]
		if item == nil {
			item = &TrialSummary{PlanVersion: trial.PlanVersion, MaterialCode: trial.MaterialCode, GateStatus: "blocked", Expired: trial.PlanVersion != report.CurrentPlanVersion, Attempts: []TrialAttempt{}}
			groups[k] = item
		}
		item.TotalCount++
		item.LatestObservedAt = trial.ObservedAt
		if trial.Outcome == "passed" {
			item.PassedCount++
			item.GateStatus = "passed"
		} else {
			item.FailedCount++
		}
		item.Attempts = append(item.Attempts, TrialAttempt{ID: trial.ID, ObservedAt: trial.ObservedAt, Outcome: trial.Outcome, EvidenceRef: trial.EvidenceRef, Failures: ExplainMetricFailures(trial.Thresholds, trial.Measurements)})
	}
	for _, item := range groups {
		report.Items = append(report.Items, *item)
		if !item.Expired && item.GateStatus == "passed" {
			report.CurrentGateStatus = "passed"
		}
	}
	sort.Slice(report.Items, func(i, j int) bool {
		if report.Items[i].PlanVersion != report.Items[j].PlanVersion {
			return report.Items[i].PlanVersion < report.Items[j].PlanVersion
		}
		return report.Items[i].MaterialCode < report.Items[j].MaterialCode
	})
	return report
}

func FindTrial(c *ConservationCase, id string) (CompatibilityTrial, error) {
	for _, trial := range c.Trials {
		if trial.ID == id {
			return trial, nil
		}
	}
	return CompatibilityTrial{}, NotFound("材料试验不存在")
}

type ParameterDeviation struct {
	Name            string  `json:"name"`
	Target          float64 `json:"target"`
	Actual          float64 `json:"actual"`
	Tolerance       float64 `json:"tolerance"`
	Difference      float64 `json:"difference"`
	WithinTolerance bool    `json:"within_tolerance"`
}

type CheckpointProgress struct {
	StepIndex  int                  `json:"step_index"`
	Outcome    string               `json:"outcome"`
	OperatorID string               `json:"operator_id"`
	Parameters []ParameterDeviation `json:"parameters"`
}

type PendingDeviation struct {
	StepIndex       int                  `json:"step_index"`
	OperatorID      string               `json:"operator_id"`
	Impact          string               `json:"impact,omitempty"`
	Remediation     string               `json:"remediation,omitempty"`
	VerifiedBy      string               `json:"verified_by,omitempty"`
	MissingFields   []string             `json:"missing_fields"`
	ParameterIssues []ParameterDeviation `json:"parameter_issues"`
}

type ExecutionStatus struct {
	CaseID               string               `json:"case_id"`
	Revision             int64                `json:"revision"`
	State                CaseState            `json:"state"`
	ReadOnly             bool                 `json:"read_only"`
	TotalSteps           int                  `json:"total_steps"`
	CompletedSteps       int                  `json:"completed_steps"`
	NextStepIndex        *int                 `json:"next_step_index,omitempty"`
	CompletionPercentage float64              `json:"completion_percentage"`
	Checkpoints          []CheckpointProgress `json:"checkpoints"`
	PendingDeviations    []PendingDeviation   `json:"pending_deviations"`
}

func BuildExecutionStatus(c *ConservationCase) ExecutionStatus {
	report := ExecutionStatus{CaseID: c.ID, Revision: c.Revision, State: c.State, ReadOnly: c.State == StateArchived, Checkpoints: []CheckpointProgress{}, PendingDeviations: []PendingDeviation{}}
	plan := c.ApprovedPlan()
	if plan == nil {
		return report
	}
	report.TotalSteps = len(plan.Steps)
	for _, cp := range c.Checkpoints {
		step := plan.Steps[cp.StepIndex-1]
		params := []ParameterDeviation{}
		names := make([]string, 0, len(step.Parameters))
		for name := range step.Parameters {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			target, actual, tolerance := step.Parameters[name], cp.ActualParameters[name], step.Tolerances[name]
			diff := actual - target
			params = append(params, ParameterDeviation{Name: name, Target: target, Actual: actual, Tolerance: tolerance, Difference: diff, WithinTolerance: math.Abs(diff) <= tolerance})
		}
		report.Checkpoints = append(report.Checkpoints, CheckpointProgress{StepIndex: cp.StepIndex, Outcome: cp.Outcome, OperatorID: cp.OperatorID, Parameters: params})
		if cp.Outcome == "completed" || cp.Outcome == "remediated" {
			report.CompletedSteps++
		}
		if cp.Outcome == "deviation" {
			missing := []string{}
			if strings.TrimSpace(cp.DeviationNote) == "" {
				missing = append(missing, "impact")
			}
			if strings.TrimSpace(cp.Remediation) == "" {
				missing = append(missing, "remediation")
			}
			if strings.TrimSpace(cp.VerifiedBy) == "" {
				missing = append(missing, "verified_by")
			}
			issues := []ParameterDeviation{}
			for _, p := range params {
				if !p.WithinTolerance {
					issues = append(issues, p)
				}
			}
			report.PendingDeviations = append(report.PendingDeviations, PendingDeviation{StepIndex: cp.StepIndex, OperatorID: cp.OperatorID, Impact: cp.DeviationNote, Remediation: cp.Remediation, VerifiedBy: cp.VerifiedBy, MissingFields: missing, ParameterIssues: issues})
		}
	}
	if report.TotalSteps > 0 {
		report.CompletionPercentage = math.Round(float64(report.CompletedSteps)*1000/float64(report.TotalSteps)) / 10
	}
	if c.State != StatePaused && len(c.Checkpoints) < report.TotalSteps {
		next := len(c.Checkpoints) + 1
		report.NextStepIndex = &next
	}
	return report
}

type StabilityMetricTrend struct {
	Name       string  `json:"name"`
	FirstValue float64 `json:"first_value"`
	LastValue  float64 `json:"last_value"`
	MinValue   float64 `json:"min_value"`
	MaxValue   float64 `json:"max_value"`
}

type StabilityAttempt struct {
	ID            string                `json:"id"`
	ObservedAt    time.Time             `json:"observed_at"`
	DurationHours int                   `json:"duration_hours"`
	Outcome       string                `json:"outcome"`
	Failures      []MetricFailureDetail `json:"failures"`
}

type ReleaseReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ReleasePrecheck struct {
	Eligible bool            `json:"eligible"`
	Reasons  []ReleaseReason `json:"reasons"`
}

func (p ReleasePrecheck) Error() error {
	if len(p.Reasons) == 0 {
		return nil
	}
	switch p.Reasons[0].Code {
	case "duty_conflict":
		return Forbidden(p.Reasons[0].Message)
	case "missing_reviewer", "missing_statement":
		return Invalid("release", p.Reasons[0].Message)
	default:
		return Conflict(p.Reasons[0].Message)
	}
}

type StabilityReport struct {
	CaseID             string                 `json:"case_id"`
	Revision           int64                  `json:"revision"`
	State              CaseState              `json:"state"`
	ReadOnly           bool                   `json:"read_only"`
	CumulativeHours    int                    `json:"cumulative_hours"`
	Trends             []StabilityMetricTrend `json:"trends"`
	Observations       []StabilityAttempt     `json:"observations"`
	ReleaseEligibility ReleasePrecheck        `json:"release_eligibility"`
}

func BuildStabilityReport(c *ConservationCase) StabilityReport {
	report := StabilityReport{CaseID: c.ID, Revision: c.Revision, State: c.State, ReadOnly: c.State == StateArchived, Trends: []StabilityMetricTrend{}, Observations: []StabilityAttempt{}}
	observations := append([]StabilityObservation(nil), c.StabilityObservations...)
	sort.SliceStable(observations, func(i, j int) bool {
		if observations[i].ObservedAt.Equal(observations[j].ObservedAt) {
			return observations[i].ID < observations[j].ID
		}
		return observations[i].ObservedAt.Before(observations[j].ObservedAt)
	})
	trendByName := map[string]*StabilityMetricTrend{}
	for _, observation := range observations {
		report.CumulativeHours += observation.DurationHours
		report.Observations = append(report.Observations, StabilityAttempt{ID: observation.ID, ObservedAt: observation.ObservedAt, DurationHours: observation.DurationHours, Outcome: observation.Outcome, Failures: ExplainMetricFailures(observation.Thresholds, observation.Measurements)})
		for _, metric := range observation.Measurements {
			trend := trendByName[metric.Name]
			if trend == nil {
				trend = &StabilityMetricTrend{Name: metric.Name, FirstValue: metric.Value, MinValue: metric.Value, MaxValue: metric.Value}
				trendByName[metric.Name] = trend
			}
			trend.LastValue = metric.Value
			if metric.Value < trend.MinValue {
				trend.MinValue = metric.Value
			}
			if metric.Value > trend.MaxValue {
				trend.MaxValue = metric.Value
			}
		}
	}
	for _, trend := range trendByName {
		report.Trends = append(report.Trends, *trend)
	}
	sort.Slice(report.Trends, func(i, j int) bool { return report.Trends[i].Name < report.Trends[j].Name })
	report.ReleaseEligibility = ReleasePrecheck{Eligible: c.State == StateStable || ((c.State == StateReleased || c.State == StateArchived) && c.Release != nil), Reasons: []ReleaseReason{}}
	if !report.ReleaseEligibility.Eligible {
		report.ReleaseEligibility.Reasons = append(report.ReleaseEligibility.Reasons, ReleaseReason{Code: "state_not_stable", Message: "最新稳定性观察未通过或累计观察不足 24 小时"})
	}
	return report
}

func BuildReleasePrecheck(c *ConservationCase, reviewer, statement string) ReleasePrecheck {
	precheck := ReleasePrecheck{Eligible: true, Reasons: []ReleaseReason{}}
	add := func(code, message string) {
		precheck.Eligible = false
		precheck.Reasons = append(precheck.Reasons, ReleaseReason{Code: code, Message: message})
	}
	if c.State != StateStable {
		add("state_not_stable", "最新稳定性观察未通过或累计观察不足 24 小时")
	}
	if strings.TrimSpace(reviewer) == "" {
		add("missing_reviewer", "审核员不能为空")
	}
	if strings.TrimSpace(statement) == "" {
		add("missing_statement", "签署声明不能为空")
	}
	for _, cp := range c.Checkpoints {
		if cp.OperatorID == reviewer && reviewer != "" {
			add("duty_conflict", "独立审核员不得参与处理执行")
			break
		}
	}
	return precheck
}
