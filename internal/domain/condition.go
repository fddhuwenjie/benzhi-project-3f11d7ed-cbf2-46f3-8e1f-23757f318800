package domain

import "strings"

func (c *ConservationCase) AddCondition(o ConditionObservation) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.State != StateDraft {
		return Conflict("只有建档状态可维护处理前状况")
	}
	if strings.TrimSpace(o.LeafRef) == "" {
		return Invalid("leaf_ref", "不能为空")
	}
	if strings.TrimSpace(o.RegionRef) == "" {
		return Invalid("region_ref", "不能为空")
	}
	if strings.TrimSpace(o.Medium) == "" || strings.TrimSpace(o.DamageType) == "" {
		return Invalid("medium", "介质与破损类型不能为空")
	}
	if o.Severity < 1 || o.Severity > 5 {
		return Invalid("severity", "必须为 1 至 5")
	}
	if strings.TrimSpace(o.EvidenceRef) == "" || strings.TrimSpace(o.RecordedBy) == "" {
		return Invalid("evidence_ref", "证据与记录人不能为空")
	}
	if err := ValidateEvidenceRef(o.EvidenceRef); err != nil {
		return err
	}
	valid := false
	for _, r := range c.RequiredRegions {
		if r.LeafRef == o.LeafRef && r.RegionRef == o.RegionRef {
			valid = true
			break
		}
	}
	if !valid {
		return Invalid("region_ref", "不属于必填区域")
	}
	for _, existing := range c.Conditions {
		if existing.LeafRef == o.LeafRef && existing.RegionRef == o.RegionRef {
			return Conflict("该区域已有状况记录")
		}
	}
	o.CaseID = c.ID
	c.Conditions = append(c.Conditions, o)
	return nil
}

func (c *ConservationCase) LockBaseline() error {
	if c.State != StateDraft {
		return Conflict("当前状态不能锁定基线")
	}
	if err := ValidateBaselineCoverage(c); err != nil {
		return err
	}
	c.State = StateBaselineLocked
	return nil
}
