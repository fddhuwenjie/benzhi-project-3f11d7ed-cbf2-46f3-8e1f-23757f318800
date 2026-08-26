package domain

import (
	"fmt"
	"strings"
)

func EvaluateMetrics(rules []MetricRule, values []MetricValue) (string, []string, error) {
	if len(rules) == 0 {
		return "", nil, Invalid("thresholds", "至少声明一个阈值")
	}
	if err := ValidateMetrics(rules, values); err != nil {
		return "", nil, err
	}
	measured := map[string]float64{}
	for _, v := range values {
		if strings.TrimSpace(v.Name) == "" {
			return "", nil, Invalid("measurements.name", "不能为空")
		}
		if _, ok := measured[v.Name]; ok {
			return "", nil, Invalid("measurements", "指标重复")
		}
		measured[v.Name] = v.Value
	}
	failures := []string{}
	for _, r := range rules {
		v, ok := measured[r.Name]
		if !ok {
			return "", nil, Invalid("measurements", "缺少指标 "+r.Name)
		}
		if r.Min == nil && r.Max == nil {
			return "", nil, Invalid("thresholds", "指标 "+r.Name+" 未声明边界")
		}
		if r.Min != nil && v < *r.Min {
			failures = append(failures, fmt.Sprintf("%s %.4g 低于下限 %.4g", r.Name, v, *r.Min))
		}
		if r.Max != nil && v > *r.Max {
			failures = append(failures, fmt.Sprintf("%s %.4g 高于上限 %.4g", r.Name, v, *r.Max))
		}
	}
	if len(failures) > 0 {
		return "failed", failures, nil
	}
	return "passed", nil, nil
}

func (c *ConservationCase) AddTrial(t CompatibilityTrial) error {
	if c.State != StatePlanDraft {
		return Conflict("当前状态不能登记材料试验")
	}
	p := c.CurrentPlan()
	if p == nil || p.Status != "submitted" {
		return Conflict("必须先提交完整方案")
	}
	if t.PlanVersion != p.Version {
		return Invalid("plan_version", "必须针对当前方案版本")
	}
	if strings.TrimSpace(t.MaterialCode) == "" || strings.TrimSpace(t.Protocol) == "" || strings.TrimSpace(t.EvidenceRef) == "" {
		return Invalid("trial", "材料、规程和证据均必填")
	}
	if err := ValidateEvidenceRef(t.EvidenceRef); err != nil {
		return err
	}
	outcome, failures, err := EvaluateMetrics(t.Thresholds, t.Measurements)
	if err != nil {
		return err
	}
	t.CaseID = c.ID
	t.Outcome = outcome
	t.Failures = failures
	c.Trials = append(c.Trials, t)
	if outcome == "passed" {
		c.State = StateTrialPassed
	}
	return nil
}
