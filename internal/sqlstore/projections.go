package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"manuscript-conservation-gate/internal/domain"
	"time"
)

func projectionSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS condition_observations(id TEXT PRIMARY KEY,case_id TEXT NOT NULL,leaf_ref TEXT NOT NULL,region_ref TEXT NOT NULL,medium TEXT NOT NULL,damage_type TEXT NOT NULL,severity INTEGER NOT NULL,measurement TEXT NOT NULL,evidence_ref TEXT NOT NULL,recorded_by TEXT NOT NULL,recorded_at TEXT NOT NULL,UNIQUE(case_id,leaf_ref,region_ref),FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS treatment_plans(id TEXT PRIMARY KEY,case_id TEXT NOT NULL,version INTEGER NOT NULL,status TEXT NOT NULL,steps_json BLOB NOT NULL,reversibility_note TEXT NOT NULL,trace_preservation_note TEXT NOT NULL,risk_controls TEXT NOT NULL,submitted_at TEXT,reviewer_id TEXT NOT NULL,review_decision TEXT NOT NULL,review_reason TEXT NOT NULL,reviewed_at TEXT,UNIQUE(case_id,version),FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS compatibility_trials(id TEXT PRIMARY KEY,case_id TEXT NOT NULL,plan_version INTEGER NOT NULL,material_code TEXT NOT NULL,protocol TEXT NOT NULL,thresholds_json BLOB NOT NULL,measurements_json BLOB NOT NULL,outcome TEXT NOT NULL,failures_json BLOB NOT NULL,evidence_ref TEXT NOT NULL,observed_at TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS treatment_checkpoints(id TEXT PRIMARY KEY,case_id TEXT NOT NULL,step_index INTEGER NOT NULL,operator_id TEXT NOT NULL,actual_parameters_json BLOB NOT NULL,outcome TEXT NOT NULL,deviation_note TEXT NOT NULL,remediation TEXT NOT NULL,verified_by TEXT NOT NULL,evidence_ref TEXT NOT NULL,completed_at TEXT NOT NULL,UNIQUE(case_id,step_index),FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS stability_observations(id TEXT PRIMARY KEY,case_id TEXT NOT NULL,observer_id TEXT NOT NULL,duration_hours INTEGER NOT NULL,thresholds_json BLOB NOT NULL,measurements_json BLOB NOT NULL,outcome TEXT NOT NULL,failures_json BLOB NOT NULL,evidence_ref TEXT NOT NULL,observed_at TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS release_signatures(case_id TEXT PRIMARY KEY,reviewer_id TEXT NOT NULL,statement TEXT NOT NULL,signed_at TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS archive_manifests(case_id TEXT PRIMARY KEY,case_revision INTEGER NOT NULL,generated_at TEXT NOT NULL,document_digests_json BLOB NOT NULL,event_count INTEGER NOT NULL,audit_chain_head TEXT NOT NULL,manifest_digest TEXT NOT NULL,verified_at TEXT,FOREIGN KEY(case_id) REFERENCES conservation_cases(id) ON DELETE CASCADE)`,
		`CREATE INDEX IF NOT EXISTS idx_conditions_case ON condition_observations(case_id)`, `CREATE INDEX IF NOT EXISTS idx_trials_case_plan ON compatibility_trials(case_id,plan_version)`, `CREATE INDEX IF NOT EXISTS idx_trials_case_plan_material ON compatibility_trials(case_id,plan_version,material_code,observed_at)`, `CREATE INDEX IF NOT EXISTS idx_checkpoints_case_step ON treatment_checkpoints(case_id,step_index)`, `CREATE INDEX IF NOT EXISTS idx_stability_case_observed ON stability_observations(case_id,observed_at)`,
	}
}
func jsonBytes(v any) []byte { b, _ := json.Marshal(v); return b }
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
func syncProjections(ctx context.Context, tx *sql.Tx, c *domain.ConservationCase) error {
	tables := []string{"condition_observations", "treatment_plans", "compatibility_trials", "treatment_checkpoints", "stability_observations", "release_signatures", "archive_manifests"}
	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE case_id=?", c.ID); err != nil {
			return err
		}
	}
	for _, o := range c.Conditions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO condition_observations VALUES(?,?,?,?,?,?,?,?,?,?,?)`, o.ID, c.ID, o.LeafRef, o.RegionRef, o.Medium, o.DamageType, o.Severity, o.Measurement, o.EvidenceRef, o.RecordedBy, o.RecordedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	for _, p := range c.Plans {
		if _, err := tx.ExecContext(ctx, `INSERT INTO treatment_plans(id,case_id,version,status,steps_json,reversibility_note,trace_preservation_note,risk_controls,submitted_at,reviewer_id,review_decision,review_reason,reviewed_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, p.ID, c.ID, p.Version, p.Status, jsonBytes(p.Steps), p.ReversibilityNote, p.TracePreservationNote, p.RiskControls, nullableTime(p.SubmittedAt), p.ReviewerID, p.ReviewDecision, p.ReviewReason, nullableTime(p.ReviewedAt)); err != nil {
			return err
		}
	}
	for _, t := range c.Trials {
		if _, err := tx.ExecContext(ctx, `INSERT INTO compatibility_trials VALUES(?,?,?,?,?,?,?,?,?,?,?)`, t.ID, c.ID, t.PlanVersion, t.MaterialCode, t.Protocol, jsonBytes(t.Thresholds), jsonBytes(t.Measurements), t.Outcome, jsonBytes(t.Failures), t.EvidenceRef, t.ObservedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	for _, cp := range c.Checkpoints {
		if _, err := tx.ExecContext(ctx, `INSERT INTO treatment_checkpoints VALUES(?,?,?,?,?,?,?,?,?,?,?)`, cp.ID, c.ID, cp.StepIndex, cp.OperatorID, jsonBytes(cp.ActualParameters), cp.Outcome, cp.DeviationNote, cp.Remediation, cp.VerifiedBy, cp.EvidenceRef, cp.CompletedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	for _, o := range c.StabilityObservations {
		if _, err := tx.ExecContext(ctx, `INSERT INTO stability_observations VALUES(?,?,?,?,?,?,?,?,?,?)`, o.ID, c.ID, o.ObserverID, o.DurationHours, jsonBytes(o.Thresholds), jsonBytes(o.Measurements), o.Outcome, jsonBytes(o.Failures), o.EvidenceRef, o.ObservedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	if c.Release != nil {
		if _, err := tx.ExecContext(ctx, `INSERT INTO release_signatures VALUES(?,?,?,?)`, c.ID, c.Release.ReviewerID, c.Release.Statement, c.Release.SignedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	if c.Archive != nil {
		if _, err := tx.ExecContext(ctx, `INSERT INTO archive_manifests VALUES(?,?,?,?,?,?,?,?)`, c.ID, c.Archive.CaseRevision, c.Archive.GeneratedAt.UTC().Format(time.RFC3339Nano), jsonBytes(c.Archive.DocumentDigests), c.Archive.EventCount, c.Archive.AuditChainHead, c.Archive.ManifestDigest, nullableTime(c.Archive.VerifiedAt)); err != nil {
			return err
		}
	}
	return nil
}
