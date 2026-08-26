package domain

import (
	"fmt"
	"math"
	"strings"
)

func (c *ConservationCase) CompleteCheckpoint(cp TreatmentCheckpoint) error {
	if c.State != StateApproved && c.State != StateTreating {
		return Conflict("当前状态不能执行处理检查点")
	}
	p := c.ApprovedPlan()
	if p == nil {
		return Conflict("没有已批准方案")
	}
	expected := len(c.Checkpoints) + 1
	if cp.StepIndex != expected {
		return Invalid("step_index", fmt.Sprintf("必须依序完成下一步骤，期望 step_index=%d", expected))
	}
	if cp.StepIndex > len(p.Steps) {
		return Invalid("step_index", "超出批准步骤")
	}
	if strings.TrimSpace(cp.OperatorID) == "" || strings.TrimSpace(cp.EvidenceRef) == "" {
		return Invalid("checkpoint", "操作者与证据均必填")
	}
	if err := ValidateEvidenceRef(cp.EvidenceRef); err != nil {
		return err
	}
	if err := ValidateParameters("actual_parameters", cp.ActualParameters); err != nil {
		return err
	}
	step := p.Steps[cp.StepIndex-1]
	deviated := false
	for name, target := range step.Parameters {
		actual, ok := cp.ActualParameters[name]
		if !ok {
			return Invalid("actual_parameters", "缺少参数 "+name)
		}
		tol := step.Tolerances[name]
		if math.Abs(actual-target) > tol {
			deviated = true
		}
	}
	cp.CaseID = c.ID
	if deviated {
		cp.Outcome = "deviation"
		c.State = StatePaused
	} else {
		cp.Outcome = "completed"
		if cp.StepIndex == len(p.Steps) {
			c.State = StateTreated
		} else {
			c.State = StateTreating
		}
	}
	c.Checkpoints = append(c.Checkpoints, cp)
	return nil
}

func (c *ConservationCase) ResolveDeviation(note, remediation, verifiedBy string) error {
	if c.State != StatePaused {
		return Conflict("个案未处于暂停状态")
	}
	if strings.TrimSpace(note) == "" || strings.TrimSpace(remediation) == "" || strings.TrimSpace(verifiedBy) == "" {
		return Invalid("deviation", "影响、补救措施和复核人均必填")
	}
	cp := &c.Checkpoints[len(c.Checkpoints)-1]
	if verifiedBy == cp.OperatorID {
		return Forbidden("偏差复核人不得为原操作者")
	}
	cp.DeviationNote = note
	cp.Remediation = remediation
	cp.VerifiedBy = verifiedBy
	cp.Outcome = "remediated"
	p := c.ApprovedPlan()
	if cp.StepIndex == len(p.Steps) {
		c.State = StateTreated
	} else {
		c.State = StateTreating
	}
	return nil
}
