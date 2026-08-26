package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"reflect"
	"time"
)

func corrupt(name string) error { return fmt.Errorf("%w: %s", application.ErrProjectionCorrupt, name) }
func sameJSON(raw []byte, want any) bool {
	var actual, expected any
	if json.Unmarshal(raw, &actual) != nil {
		return false
	}
	b, _ := json.Marshal(want)
	if json.Unmarshal(b, &expected) != nil {
		return false
	}
	return reflect.DeepEqual(actual, expected)
}
func (s *Store) verifyProjections(ctx context.Context, c *domain.ConservationCase) error {
	checks := []func(context.Context, *domain.ConservationCase) error{s.verifyConditionRows, s.verifyPlanRows, s.verifyTrialRows, s.verifyCheckpointRows, s.verifyStabilityRows, s.verifyReleaseRow, s.verifyArchiveRow}
	for _, check := range checks {
		if err := check(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) verifyConditionRows(ctx context.Context, c *domain.ConservationCase) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,leaf_ref,region_ref,medium,damage_type,severity,measurement,evidence_ref,recorded_by FROM condition_observations WHERE case_id=? ORDER BY id`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	actual := map[string][]any{}
	for rows.Next() {
		var id, leaf, region, medium, damage, measurement, evidence, recorded string
		var severity int
		if err := rows.Scan(&id, &leaf, &region, &medium, &damage, &severity, &measurement, &evidence, &recorded); err != nil {
			return err
		}
		actual[id] = []any{leaf, region, medium, damage, severity, measurement, evidence, recorded}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(actual) != len(c.Conditions) {
		return corrupt("condition_observations 数量")
	}
	for _, o := range c.Conditions {
		want := []any{o.LeafRef, o.RegionRef, o.Medium, o.DamageType, o.Severity, o.Measurement, o.EvidenceRef, o.RecordedBy}
		if !reflect.DeepEqual(actual[o.ID], want) {
			return corrupt("condition_observations 内容")
		}
	}
	return nil
}
func (s *Store) verifyPlanRows(ctx context.Context, c *domain.ConservationCase) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,version,status,steps_json,reversibility_note,trace_preservation_note,risk_controls,submitted_at,reviewer_id,review_decision,review_reason FROM treatment_plans WHERE case_id=? ORDER BY version`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		if i >= len(c.Plans) {
			return corrupt("treatment_plans 数量")
		}
		var id, status, rev, trace, risk, reviewer, decision, reason string
		var version int
		var steps []byte
		var submitted sql.NullString
		if err := rows.Scan(&id, &version, &status, &steps, &rev, &trace, &risk, &submitted, &reviewer, &decision, &reason); err != nil {
			return err
		}
		p := c.Plans[i]
		wantSubmitted := ""
		if p.SubmittedAt != nil {
			wantSubmitted = p.SubmittedAt.UTC().Format(time.RFC3339Nano)
		}
		if id != p.ID || version != p.Version || status != p.Status || rev != p.ReversibilityNote || trace != p.TracePreservationNote || risk != p.RiskControls || submitted.String != wantSubmitted || reviewer != p.ReviewerID || decision != p.ReviewDecision || reason != p.ReviewReason || !sameJSON(steps, p.Steps) {
			return corrupt("treatment_plans 内容")
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if i != len(c.Plans) {
		return corrupt("treatment_plans 数量")
	}
	return nil
}
func (s *Store) verifyTrialRows(ctx context.Context, c *domain.ConservationCase) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,plan_version,material_code,protocol,thresholds_json,measurements_json,outcome,failures_json,evidence_ref FROM compatibility_trials WHERE case_id=? ORDER BY rowid`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		if i >= len(c.Trials) {
			return corrupt("compatibility_trials 数量")
		}
		var id, material, protocol, outcome, evidence string
		var plan int
		var rules, values, failures []byte
		if err := rows.Scan(&id, &plan, &material, &protocol, &rules, &values, &outcome, &failures, &evidence); err != nil {
			return err
		}
		t := c.Trials[i]
		if id != t.ID || plan != t.PlanVersion || material != t.MaterialCode || protocol != t.Protocol || outcome != t.Outcome || evidence != t.EvidenceRef || !sameJSON(rules, t.Thresholds) || !sameJSON(values, t.Measurements) || !sameJSON(failures, t.Failures) {
			return corrupt("compatibility_trials 内容")
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if i != len(c.Trials) {
		return corrupt("compatibility_trials 数量")
	}
	return nil
}
func (s *Store) verifyCheckpointRows(ctx context.Context, c *domain.ConservationCase) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,step_index,operator_id,actual_parameters_json,outcome,deviation_note,remediation,verified_by,evidence_ref FROM treatment_checkpoints WHERE case_id=? ORDER BY step_index`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		if i >= len(c.Checkpoints) {
			return corrupt("treatment_checkpoints 数量")
		}
		var id, operator, outcome, note, remediation, verified, evidence string
		var step int
		var params []byte
		if err := rows.Scan(&id, &step, &operator, &params, &outcome, &note, &remediation, &verified, &evidence); err != nil {
			return err
		}
		cp := c.Checkpoints[i]
		if id != cp.ID || step != cp.StepIndex || operator != cp.OperatorID || outcome != cp.Outcome || note != cp.DeviationNote || remediation != cp.Remediation || verified != cp.VerifiedBy || evidence != cp.EvidenceRef || !sameJSON(params, cp.ActualParameters) {
			return corrupt("treatment_checkpoints 内容")
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if i != len(c.Checkpoints) {
		return corrupt("treatment_checkpoints 数量")
	}
	return nil
}
func (s *Store) verifyStabilityRows(ctx context.Context, c *domain.ConservationCase) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,observer_id,duration_hours,thresholds_json,measurements_json,outcome,failures_json,evidence_ref FROM stability_observations WHERE case_id=? ORDER BY rowid`, c.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		if i >= len(c.StabilityObservations) {
			return corrupt("stability_observations 数量")
		}
		var id, observer, outcome, evidence string
		var duration int
		var rules, values, failures []byte
		if err := rows.Scan(&id, &observer, &duration, &rules, &values, &outcome, &failures, &evidence); err != nil {
			return err
		}
		o := c.StabilityObservations[i]
		if id != o.ID || observer != o.ObserverID || duration != o.DurationHours || outcome != o.Outcome || evidence != o.EvidenceRef || !sameJSON(rules, o.Thresholds) || !sameJSON(values, o.Measurements) || !sameJSON(failures, o.Failures) {
			return corrupt("stability_observations 内容")
		}
		i++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if i != len(c.StabilityObservations) {
		return corrupt("stability_observations 数量")
	}
	return nil
}
func (s *Store) verifyReleaseRow(ctx context.Context, c *domain.ConservationCase) error {
	var reviewer, statement string
	err := s.db.QueryRowContext(ctx, `SELECT reviewer_id,statement FROM release_signatures WHERE case_id=?`, c.ID).Scan(&reviewer, &statement)
	if err == sql.ErrNoRows {
		if c.Release == nil {
			return nil
		}
		return corrupt("release_signatures 缺失")
	}
	if err != nil {
		return err
	}
	if c.Release == nil || reviewer != c.Release.ReviewerID || statement != c.Release.Statement {
		return corrupt("release_signatures 内容")
	}
	return nil
}
func (s *Store) verifyArchiveRow(ctx context.Context, c *domain.ConservationCase) error {
	var revision, count int64
	var head, digest string
	err := s.db.QueryRowContext(ctx, `SELECT case_revision,event_count,audit_chain_head,manifest_digest FROM archive_manifests WHERE case_id=?`, c.ID).Scan(&revision, &count, &head, &digest)
	if err == sql.ErrNoRows {
		if c.Archive == nil {
			return nil
		}
		return corrupt("archive_manifests 缺失")
	}
	if err != nil {
		return err
	}
	if c.Archive == nil || revision != c.Archive.CaseRevision || count != c.Archive.EventCount || head != c.Archive.AuditChainHead || digest != c.Archive.ManifestDigest {
		return corrupt("archive_manifests 内容")
	}
	return nil
}
