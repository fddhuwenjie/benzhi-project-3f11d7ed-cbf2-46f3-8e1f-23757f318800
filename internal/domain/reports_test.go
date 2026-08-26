package domain

import (
	"testing"
	"time"
)

func reportPlan(parameters map[string]float64) TreatmentPlan {
	return TreatmentPlan{ID: "plan", Steps: []PlanStep{{Index: 1, Purpose: "加固", Material: "日本纸", Parameters: parameters, Tolerances: map[string]float64{"humidity": 0.5}, Reversibility: "可移除", StopCondition: "变色", RiskMitigation: "低湿"}}, ReversibilityNote: "可逆", TracePreservationNote: "保留痕迹", RiskControls: "设置停止点"}
}

func TestCoverageReportAndLockPrecheck(t *testing.T) {
	c, err := CreateCase(NewCase{ID: "coverage", ManuscriptCode: "MS-C", Title: "状况覆盖", CustodianID: "owner", SignificanceNote: "重要", TreatmentGoal: "稳定", InitialRisk: "裂口", RequiredRegions: []RegionRequirement{{LeafRef: "1r", RegionRef: "top"}, {LeafRef: "1r", RegionRef: "bottom"}}, CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.AddCondition(ConditionObservation{ID: "c1", LeafRef: "1r", RegionRef: "top", Medium: "墨", DamageType: "裂口", Severity: 4, Measurement: "2mm", EvidenceRef: "sha256:a", RecordedBy: "worker", RecordedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	report := BuildCoverageReport(c)
	if report.CoveragePercentage != 50 || len(report.Missing) != 1 || report.Missing[0].RegionRef != "bottom" {
		t.Fatalf("report=%+v", report)
	}
	if err = c.LockBaseline(); err == nil || c.State != StateDraft {
		t.Fatalf("缺口应拒绝锁定: %v", err)
	}
	if err = c.AddCondition(ConditionObservation{ID: "c2", LeafRef: "1r", RegionRef: "bottom", Medium: "墨", DamageType: "褪色", Severity: 2, EvidenceRef: "sha256:b", RecordedBy: "worker", RecordedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	report = BuildCoverageReport(c)
	if len(report.Anomalies) != 1 || report.Anomalies[0].Type != "missing_measurement" {
		t.Fatalf("anomalies=%+v", report.Anomalies)
	}
	if err = c.LockBaseline(); err != nil {
		t.Fatal(err)
	}
	if !BuildCoverageReport(c).ReadOnly {
		t.Fatal("锁定后报告应只读")
	}
}

func TestPlanHistoryDiffAndTrialVersionGate(t *testing.T) {
	c := newTestCase(t)
	now := time.Now().UTC()
	if err := c.AddCondition(ConditionObservation{ID: "condition", LeafRef: "1r", RegionRef: "center", Medium: "墨", DamageType: "裂口", Severity: 2, Measurement: "1mm", EvidenceRef: "sha256:a", RecordedBy: "worker", RecordedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := c.LockBaseline(); err != nil {
		t.Fatal(err)
	}
	if err := c.SavePlan(reportPlan(map[string]float64{"humidity": 5})); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlan(now); err != nil {
		t.Fatal(err)
	}
	max := 1.0
	if err := c.AddTrial(CompatibilityTrial{ID: "failed", PlanVersion: 1, MaterialCode: "M1", Protocol: "test", Thresholds: []MetricRule{{Name: "delta", Max: &max}}, Measurements: []MetricValue{{Name: "delta", Value: 2}}, EvidenceRef: "sha256:f", ObservedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddTrial(CompatibilityTrial{ID: "passed", PlanVersion: 1, MaterialCode: "M1", Protocol: "test", Thresholds: []MetricRule{{Name: "delta", Max: &max}}, Measurements: []MetricValue{{Name: "delta", Value: .2}}, EvidenceRef: "sha256:p", ObservedAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	trials := BuildTrialReport(c)
	if trials.CurrentGateStatus != "passed" || trials.Items[0].TotalCount != 2 || trials.Items[0].FailedCount != 1 {
		t.Fatalf("trials=%+v", trials)
	}
	if err := c.ReviewEthics("owner", "return", "调整湿度", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	v2 := reportPlan(map[string]float64{"humidity": 4})
	v2.ID = "plan-2"
	if err := c.SavePlan(v2); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlan(now.Add(3 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	history := BuildPlanHistory(c)
	if history.CurrentVersion != 2 || history.Items[0].Status != "returned" || history.Items[0].ReviewReason == "" {
		t.Fatalf("history=%+v", history)
	}
	diff, err := BuildPlanDiff(c, 2)
	if err != nil || len(diff.Steps) != 1 || len(diff.Steps[0].Changes) != 1 || diff.Steps[0].Changes[0].Field != "parameters" {
		t.Fatalf("diff=%+v err=%v", diff, err)
	}
	trials = BuildTrialReport(c)
	if trials.CurrentGateStatus != "blocked" || !trials.Items[0].Expired {
		t.Fatalf("trials=%+v", trials)
	}
	if err := c.ReviewEthics("owner", "approve", "", now); err == nil {
		t.Fatal("新版本无通过试验应拒绝审查")
	}
}

func TestExecutionStatusAndStabilityEligibility(t *testing.T) {
	now := time.Now().UTC()
	c := newTestCase(t)
	approved := reportPlan(map[string]float64{"humidity": 5})
	approved.CaseID, approved.Version, approved.Status, approved.ReviewerID, approved.ReviewDecision = c.ID, 1, "approved", "owner", "approve"
	approved.SubmittedAt, approved.ReviewedAt = &now, &now
	approved.Steps = append(approved.Steps, PlanStep{Index: 2, Purpose: "整平", Material: "吸水纸", Parameters: map[string]float64{"pressure": 2}, Tolerances: map[string]float64{"pressure": .2}, Reversibility: "可撤除", StopCondition: "变形", RiskMitigation: "逐步加压"}, PlanStep{Index: 3, Purpose: "干燥", Material: "吸水纸", Parameters: map[string]float64{"hours": 4}, Tolerances: map[string]float64{"hours": .5}, Reversibility: "可调整", StopCondition: "变形", RiskMitigation: "分段观察"})
	c.Plans, c.State = []TreatmentPlan{approved}, StateApproved
	if err := c.CompleteCheckpoint(TreatmentCheckpoint{ID: "cp1", StepIndex: 1, OperatorID: "worker", ActualParameters: map[string]float64{"humidity": 5}, EvidenceRef: "sha256:cp1", CompletedAt: now}); err != nil {
		t.Fatal(err)
	}
	status := BuildExecutionStatus(c)
	if status.CompletedSteps != 1 || status.NextStepIndex == nil || *status.NextStepIndex != 2 || status.CompletionPercentage != 33.3 {
		t.Fatalf("status=%+v", status)
	}
	if err := c.CompleteCheckpoint(TreatmentCheckpoint{ID: "cp3", StepIndex: 3, OperatorID: "worker", ActualParameters: map[string]float64{"hours": 4}, EvidenceRef: "sha256:cp3", CompletedAt: now}); err == nil {
		t.Fatal("跳步应拒绝")
	}
	if err := c.CompleteCheckpoint(TreatmentCheckpoint{ID: "cp2", StepIndex: 2, OperatorID: "worker", ActualParameters: map[string]float64{"pressure": 3}, EvidenceRef: "sha256:cp2", CompletedAt: now}); err != nil {
		t.Fatal(err)
	}
	status = BuildExecutionStatus(c)
	if len(status.PendingDeviations) != 1 || len(status.PendingDeviations[0].ParameterIssues) != 1 {
		t.Fatalf("status=%+v", status)
	}
	if err := c.ResolveDeviation("无影响", "恢复压力", "reviewer"); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteCheckpoint(TreatmentCheckpoint{ID: "cp3", StepIndex: 3, OperatorID: "worker", ActualParameters: map[string]float64{"hours": 4}, EvidenceRef: "sha256:cp3", CompletedAt: now}); err != nil {
		t.Fatal(err)
	}
	max := 1.0
	if err := c.AddStability(StabilityObservation{ID: "s1", ObserverID: "observer", DurationHours: 12, Thresholds: []MetricRule{{Name: "delta", Max: &max}}, Measurements: []MetricValue{{Name: "delta", Value: .2}}, EvidenceRef: "sha256:s1", ObservedAt: now}); err != nil {
		t.Fatal(err)
	}
	if c.State != StateTreated {
		t.Fatal("累计不足 24 小时不应稳定")
	}
	if err := c.AddStability(StabilityObservation{ID: "s2", ObserverID: "observer", DurationHours: 12, Thresholds: []MetricRule{{Name: "delta", Max: &max}}, Measurements: []MetricValue{{Name: "delta", Value: .3}}, EvidenceRef: "sha256:s2", ObservedAt: now.Add(12 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	report := BuildStabilityReport(c)
	if !report.ReleaseEligibility.Eligible || report.CumulativeHours != 24 || report.Trends[0].FirstValue != .2 || report.Trends[0].LastValue != .3 {
		t.Fatalf("report=%+v", report)
	}
	if err := c.ReleaseCase("worker", "签署", now); err == nil {
		t.Fatal("处理操作者不得放行")
	}
}
