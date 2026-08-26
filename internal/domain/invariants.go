package domain

import (
	"fmt"
	"strings"
)

// ValidateSnapshot rejects persisted aggregates that no longer satisfy workflow invariants.
func ValidateSnapshot(c *ConservationCase) error {
	if c == nil {
		return Invalid("case", "聚合不能为空")
	}
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.ManuscriptCode) == "" || strings.TrimSpace(c.CustodianID) == "" {
		return Invalid("case", "身份字段不完整")
	}
	if c.Revision < 1 || c.CreatedAt.IsZero() {
		return Invalid("revision", "修订号或创建时间无效")
	}
	if !validState(c.State) {
		return Invalid("state", "未知状态 "+string(c.State))
	}
	if err := validateRequirements(c.RequiredRegions); err != nil {
		return err
	}
	if err := validateConditionSnapshot(c); err != nil {
		return err
	}
	if err := validatePlanSnapshot(c); err != nil {
		return err
	}
	if err := validateTrialSnapshot(c); err != nil {
		return err
	}
	if err := validateCheckpointSnapshot(c); err != nil {
		return err
	}
	if err := validateStabilitySnapshot(c); err != nil {
		return err
	}
	return validateTerminalSnapshot(c)
}

func validState(s CaseState) bool {
	switch s {
	case StateDraft, StateBaselineLocked, StatePlanDraft, StateTrialPassed, StateApproved, StateTreating, StatePaused, StateTreated, StateStable, StateReleased, StateArchived:
		return true
	}
	return false
}
func validateRequirements(rs []RegionRequirement) error {
	if len(rs) == 0 {
		return Invalid("required_regions", "不能为空")
	}
	seen := map[string]bool{}
	for i, r := range rs {
		if strings.TrimSpace(r.LeafRef) == "" || strings.TrimSpace(r.RegionRef) == "" {
			return Invalid(fmt.Sprintf("required_regions[%d]", i), "引用不能为空")
		}
		k := r.LeafRef + "\x00" + r.RegionRef
		if seen[k] {
			return Invalid("required_regions", "区域重复")
		}
		seen[k] = true
	}
	return nil
}
func validateConditionSnapshot(c *ConservationCase) error {
	required := map[string]bool{}
	for _, r := range c.RequiredRegions {
		required[r.LeafRef+"\x00"+r.RegionRef] = true
	}
	seen := map[string]bool{}
	for i, o := range c.Conditions {
		if o.CaseID != c.ID || o.ID == "" || o.EvidenceRef == "" || o.RecordedBy == "" || o.RecordedAt.IsZero() {
			return Invalid(fmt.Sprintf("conditions[%d]", i), "身份或证据不完整")
		}
		if o.Severity < 1 || o.Severity > 5 {
			return Invalid(fmt.Sprintf("conditions[%d].severity", i), "超出范围")
		}
		k := o.LeafRef + "\x00" + o.RegionRef
		if !required[k] {
			return Invalid(fmt.Sprintf("conditions[%d].region_ref", i), "不是必填区域")
		}
		if seen[k] {
			return Invalid("conditions", "区域记录重复")
		}
		seen[k] = true
	}
	if c.State != StateDraft {
		for k := range required {
			if !seen[k] {
				return Invalid("conditions", "锁定后缺少必填区域 "+k)
			}
		}
	}
	return nil
}
func validatePlanSnapshot(c *ConservationCase) error {
	approved := 0
	for i, p := range c.Plans {
		if p.CaseID != c.ID || p.Version != i+1 || p.ID == "" {
			return Invalid(fmt.Sprintf("plans[%d]", i), "身份或版本无效")
		}
		switch p.Status {
		case "draft", "submitted", "returned", "approved":
		default:
			return Invalid(fmt.Sprintf("plans[%d].status", i), "未知状态")
		}
		if p.Status != "draft" {
			if err := ValidatePlan(&p); err != nil {
				return err
			}
		}
		if p.Status == "approved" {
			approved++
			if p.ReviewerID == "" || p.ReviewedAt == nil {
				return Invalid(fmt.Sprintf("plans[%d]", i), "批准信息不完整")
			}
		}
		if p.Status == "returned" && (p.ReviewerID == "" || p.ReviewReason == "" || p.ReviewedAt == nil) {
			return Invalid(fmt.Sprintf("plans[%d]", i), "退回信息不完整")
		}
	}
	if c.State != StateDraft && c.State != StateBaselineLocked && len(c.Plans) == 0 {
		return Invalid("plans", "当前状态要求方案")
	}
	if approved > 1 {
		return Invalid("plans", "只能有一个批准版本")
	}
	if stateNeedsApproval(c.State) && approved != 1 {
		return Invalid("plans", "当前状态要求唯一批准方案")
	}
	return nil
}
func stateNeedsApproval(s CaseState) bool {
	switch s {
	case StateApproved, StateTreating, StatePaused, StateTreated, StateStable, StateReleased, StateArchived:
		return true
	}
	return false
}
func validateTrialSnapshot(c *ConservationCase) error {
	passed := false
	for i, t := range c.Trials {
		if t.CaseID != c.ID || t.ID == "" || t.PlanVersion < 1 || t.PlanVersion > len(c.Plans) {
			return Invalid(fmt.Sprintf("trials[%d]", i), "身份或方案版本无效")
		}
		out, failures, err := EvaluateMetrics(t.Thresholds, t.Measurements)
		if err != nil {
			return err
		}
		if out != t.Outcome || len(failures) != len(t.Failures) {
			return Invalid(fmt.Sprintf("trials[%d].outcome", i), "阈值重算不一致")
		}
		if out == "passed" && t.PlanVersion == len(c.Plans) {
			passed = true
		}
	}
	if stateNeedsTrial(c.State) && !passed {
		return Invalid("trials", "当前状态要求当前方案试验通过")
	}
	return nil
}
func stateNeedsTrial(s CaseState) bool {
	switch s {
	case StateTrialPassed, StateApproved, StateTreating, StatePaused, StateTreated, StateStable, StateReleased, StateArchived:
		return true
	}
	return false
}
func validateCheckpointSnapshot(c *ConservationCase) error {
	p := c.ApprovedPlan()
	operators := map[string]bool{}
	for i, cp := range c.Checkpoints {
		if cp.CaseID != c.ID || cp.ID == "" || cp.StepIndex != i+1 {
			return Invalid(fmt.Sprintf("checkpoints[%d]", i), "身份或步骤序列无效")
		}
		if p == nil || cp.StepIndex > len(p.Steps) {
			return Invalid(fmt.Sprintf("checkpoints[%d].step_index", i), "超出批准方案")
		}
		if cp.OperatorID == "" || cp.EvidenceRef == "" || cp.CompletedAt.IsZero() {
			return Invalid(fmt.Sprintf("checkpoints[%d]", i), "执行证据不完整")
		}
		operators[cp.OperatorID] = true
		switch cp.Outcome {
		case "completed":
		case "deviation":
			if i != len(c.Checkpoints)-1 || c.State != StatePaused {
				return Invalid(fmt.Sprintf("checkpoints[%d].outcome", i), "未处置偏差位置无效")
			}
		case "remediated":
			if cp.DeviationNote == "" || cp.Remediation == "" || cp.VerifiedBy == "" || cp.VerifiedBy == cp.OperatorID {
				return Invalid(fmt.Sprintf("checkpoints[%d]", i), "偏差处置无效")
			}
		default:
			return Invalid(fmt.Sprintf("checkpoints[%d].outcome", i), "未知结论")
		}
	}
	if isTreatmentComplete(c.State) && (p == nil || len(c.Checkpoints) != len(p.Steps)) {
		return Invalid("checkpoints", "未完成全部批准步骤")
	}
	if c.Release != nil && operators[c.Release.ReviewerID] {
		return Invalid("release.reviewer_id", "审核员参与过处理")
	}
	return nil
}
func isTreatmentComplete(s CaseState) bool {
	switch s {
	case StateTreated, StateStable, StateReleased, StateArchived:
		return true
	}
	return false
}
func validateStabilitySnapshot(c *ConservationCase) error {
	passed := false
	for i, o := range c.StabilityObservations {
		if o.ID == "" || o.ObserverID == "" || o.DurationHours <= 0 || o.EvidenceRef == "" || o.ObservedAt.IsZero() {
			return Invalid(fmt.Sprintf("stability_observations[%d]", i), "记录不完整")
		}
		out, failures, err := EvaluateMetrics(o.Thresholds, o.Measurements)
		if err != nil {
			return err
		}
		if out != o.Outcome || len(failures) != len(o.Failures) {
			return Invalid(fmt.Sprintf("stability_observations[%d].outcome", i), "阈值重算不一致")
		}
		passed = out == "passed"
	}
	totalHours := 0
	for _, o := range c.StabilityObservations {
		totalHours += o.DurationHours
	}
	if (c.State == StateStable || c.State == StateReleased || c.State == StateArchived) && (!passed || totalHours < 24) {
		return Invalid("stability_observations", "当前状态要求通过记录")
	}
	return nil
}
func validateTerminalSnapshot(c *ConservationCase) error {
	if c.State == StateArchived {
		if c.Archive == nil || c.ArchivedAt == nil || c.Release == nil {
			return Invalid("archive", "封存事实不完整")
		}
	} else if c.Archive != nil || c.ArchivedAt != nil {
		return Invalid("archive", "非封存状态存在归档清单")
	}
	if c.State == StateReleased && c.Release == nil {
		return Invalid("release", "缺少放行签署")
	}
	if c.Release != nil && (c.Release.ReviewerID == "" || c.Release.Statement == "" || c.Release.SignedAt.IsZero()) {
		return Invalid("release", "签署不完整")
	}
	return nil
}
