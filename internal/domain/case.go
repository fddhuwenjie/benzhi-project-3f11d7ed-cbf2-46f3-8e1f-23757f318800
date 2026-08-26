package domain

import (
	"strings"
	"time"
)

type NewCase struct {
	ID, ManuscriptCode, Title, CustodianID, SignificanceNote, TreatmentGoal, InitialRisk string
	RequiredRegions                                                                      []RegionRequirement
	CreatedAt                                                                            time.Time
}

func CreateCase(in NewCase) (*ConservationCase, error) {
	checks := []struct{ field, value string }{{"id", in.ID}, {"manuscript_code", in.ManuscriptCode}, {"title", in.Title}, {"custodian_id", in.CustodianID}, {"significance_note", in.SignificanceNote}, {"treatment_goal", in.TreatmentGoal}, {"initial_risk", in.InitialRisk}}
	for _, c := range checks {
		if strings.TrimSpace(c.value) == "" {
			return nil, Invalid(c.field, "不能为空")
		}
	}
	if len(in.RequiredRegions) == 0 {
		return nil, Invalid("required_regions", "至少声明一个必填区域")
	}
	seen := map[string]bool{}
	for i, r := range in.RequiredRegions {
		if strings.TrimSpace(r.LeafRef) == "" || strings.TrimSpace(r.RegionRef) == "" {
			return nil, Invalid("required_regions", "页叶与区域不能为空")
		}
		k := r.LeafRef + "\x00" + r.RegionRef
		if seen[k] {
			return nil, Invalid("required_regions", "存在重复区域")
		}
		seen[k] = true
		in.RequiredRegions[i] = r
	}
	return &ConservationCase{ID: in.ID, ManuscriptCode: in.ManuscriptCode, Title: in.Title, CustodianID: in.CustodianID, SignificanceNote: in.SignificanceNote, TreatmentGoal: in.TreatmentGoal, InitialRisk: in.InitialRisk, RequiredRegions: in.RequiredRegions, State: StateDraft, Revision: 1, CreatedAt: in.CreatedAt, Conditions: []ConditionObservation{}, Plans: []TreatmentPlan{}, Trials: []CompatibilityTrial{}, Checkpoints: []TreatmentCheckpoint{}, StabilityObservations: []StabilityObservation{}}, nil
}

func (c *ConservationCase) EnsureMutable() error {
	if c.State == StateArchived {
		return Conflict("个案已封存，禁止修改")
	}
	return nil
}
func (c *ConservationCase) CurrentPlan() *TreatmentPlan {
	if len(c.Plans) == 0 {
		return nil
	}
	return &c.Plans[len(c.Plans)-1]
}
func (c *ConservationCase) ApprovedPlan() *TreatmentPlan {
	for i := len(c.Plans) - 1; i >= 0; i-- {
		if c.Plans[i].Status == "approved" {
			return &c.Plans[i]
		}
	}
	return nil
}
