package domain

import (
	"errors"
	"testing"
	"time"
)

func number(v float64) *float64 { return &v }
func newTestCase(t *testing.T) *ConservationCase {
	t.Helper()
	c, err := CreateCase(NewCase{ID: "case-1", ManuscriptCode: "MS-1", Title: "测试手稿", CustodianID: "owner", SignificanceNote: "重要", TreatmentGoal: "稳定", InitialRisk: "脆化", RequiredRegions: []RegionRequirement{{LeafRef: "1r", RegionRef: "center"}}, CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestWorkflowAndDeviationGate(t *testing.T) {
	c := newTestCase(t)
	now := time.Now().UTC()
	if err := c.LockBaseline(); err == nil {
		t.Fatal("缺少区域时不应锁定")
	}
	if err := c.AddCondition(ConditionObservation{ID: "condition", LeafRef: "1r", RegionRef: "center", Medium: "墨", DamageType: "裂口", Severity: 3, EvidenceRef: "sha256:a", RecordedBy: "worker", RecordedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := c.LockBaseline(); err != nil {
		t.Fatal(err)
	}
	plan := TreatmentPlan{ID: "plan", Steps: []PlanStep{{Index: 1, Purpose: "加固", Material: "纸", Parameters: map[string]float64{"humidity": 5}, Tolerances: map[string]float64{"humidity": 0.5}, Reversibility: "可移除", StopCondition: "变色", RiskMitigation: "低湿"}}, ReversibilityNote: "可逆", TracePreservationNote: "保留痕迹", RiskControls: "停止点"}
	if err := c.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlan(); err != nil {
		t.Fatal(err)
	}
	if err := c.AddTrial(CompatibilityTrial{ID: "trial", PlanVersion: 1, MaterialCode: "M1", Protocol: "老化", Thresholds: []MetricRule{{Name: "delta", Max: number(1)}}, Measurements: []MetricValue{{Name: "delta", Value: .2}}, EvidenceRef: "sha256:b", ObservedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := c.ReviewEthics("owner", "approve", "符合原则", now); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteCheckpoint(TreatmentCheckpoint{ID: "cp", StepIndex: 1, OperatorID: "worker", ActualParameters: map[string]float64{"humidity": 6}, EvidenceRef: "sha256:c", CompletedAt: now}); err != nil {
		t.Fatal(err)
	}
	if c.State != StatePaused {
		t.Fatalf("state=%s", c.State)
	}
	if err := c.ResolveDeviation("无迁移", "降低湿度", "worker"); err == nil {
		t.Fatal("操作者不应复核自己的偏差")
	}
	if err := c.ResolveDeviation("无迁移", "降低湿度", "reviewer"); err != nil {
		t.Fatal(err)
	}
	if c.State != StateTreated {
		t.Fatalf("state=%s", c.State)
	}
	if err := ValidateSnapshot(c); err != nil {
		t.Fatal(err)
	}
}
func TestMetricAndReleaseRules(t *testing.T) {
	out, failures, err := EvaluateMetrics([]MetricRule{{Name: "x", Min: number(1), Max: number(2)}}, []MetricValue{{Name: "x", Value: 3}})
	if err != nil || out != "failed" || len(failures) != 1 {
		t.Fatalf("out=%s failures=%v err=%v", out, failures, err)
	}
	_, _, err = EvaluateMetrics([]MetricRule{{Name: "x", Max: number(2)}}, nil)
	if err == nil {
		t.Fatal("缺失读数应失败")
	}
	c := newTestCase(t)
	c.State = StateStable
	c.Checkpoints = []TreatmentCheckpoint{{OperatorID: "same"}}
	if err := c.ReleaseCase("same", "签署", time.Now()); err == nil {
		t.Fatal("职责冲突应拒绝")
	}
	var de *Error
	if !errors.As(c.ReleaseCase("other", "", time.Now()), &de) || de.Field != "release" {
		t.Fatal("应返回字段化错误")
	}
}
