package domain

import (
	"fmt"
	"strings"
	"time"
)

func (c *ConservationCase) SavePlan(plan TreatmentPlan) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.State != StateBaselineLocked && c.State != StatePlanDraft {
		return Conflict("当前状态不能修订方案")
	}
	plan.CaseID = c.ID
	plan.Version = len(c.Plans) + 1
	plan.Status = "draft"
	c.Plans = append(c.Plans, plan)
	c.State = StatePlanDraft
	return nil
}

func ValidatePlan(p *TreatmentPlan) error {
	if len(p.Steps) == 0 {
		return Invalid("steps", "至少包含一个处理步骤")
	}
	for i, s := range p.Steps {
		if s.Index != i+1 {
			return Invalid(fmt.Sprintf("steps[%d].index", i), "步骤必须从 1 连续递增")
		}
		if strings.TrimSpace(s.Purpose) == "" || strings.TrimSpace(s.Material) == "" || strings.TrimSpace(s.Reversibility) == "" || strings.TrimSpace(s.StopCondition) == "" || strings.TrimSpace(s.RiskMitigation) == "" {
			return Invalid(fmt.Sprintf("steps[%d]", i), "目的、材料、可逆性、停止条件和风险缓解均必填")
		}
		if len(s.Parameters) == 0 {
			return Invalid(fmt.Sprintf("steps[%d].parameters", i), "至少声明一个参数")
		}
		for name, t := range s.Tolerances {
			if strings.TrimSpace(name) == "" || t < 0 {
				return Invalid(fmt.Sprintf("steps[%d].tolerances", i), "容差名称不能为空且不得为负")
			}
		}
	}
	if strings.TrimSpace(p.ReversibilityNote) == "" || strings.TrimSpace(p.TracePreservationNote) == "" || strings.TrimSpace(p.RiskControls) == "" {
		return Invalid("plan", "可逆性论证、历史痕迹保留与风险控制均必填")
	}
	return nil
}

func (c *ConservationCase) SubmitPlan(at ...time.Time) error {
	if c.State != StatePlanDraft {
		return Conflict("当前状态不能提交方案")
	}
	p := c.CurrentPlan()
	if p == nil {
		return Conflict("尚无方案")
	}
	if p.Status != "draft" {
		return Conflict("只能提交当前草稿版本")
	}
	if err := ValidatePlan(p); err != nil {
		return err
	}
	submittedAt := time.Now().UTC()
	if len(at) > 0 {
		submittedAt = at[0]
	}
	p.Status = "submitted"
	p.SubmittedAt = &submittedAt
	return nil
}

func (c *ConservationCase) ReviewEthics(reviewer, decision, reason string, at time.Time) error {
	if c.State != StateTrialPassed {
		return Conflict("缺少当前方案版本的通过试验")
	}
	p := c.CurrentPlan()
	if p == nil || p.Status != "submitted" {
		return Conflict("没有待审方案")
	}
	if reviewer != c.CustodianID {
		return Forbidden("只有登记的馆藏责任人可执行伦理审查")
	}
	if decision != "approve" && decision != "return" {
		return Invalid("decision", "必须为 approve 或 return")
	}
	if decision == "return" && strings.TrimSpace(reason) == "" {
		return Invalid("reason", "退回必须说明理由")
	}
	p.ReviewerID = reviewer
	p.ReviewReason = reason
	p.ReviewedAt = &at
	if decision == "return" {
		p.ReviewDecision = "return"
		p.Status = "returned"
		c.State = StatePlanDraft
	} else {
		p.ReviewDecision = "approve"
		p.Status = "approved"
		c.State = StateApproved
	}
	return nil
}
