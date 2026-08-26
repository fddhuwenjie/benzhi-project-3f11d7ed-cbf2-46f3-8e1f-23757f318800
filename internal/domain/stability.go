package domain

import (
	"strings"
	"time"
)

func (c *ConservationCase) AddStability(o StabilityObservation) error {
	if c.State != StateTreated && c.State != StateStable {
		return Conflict("处理完成后才能观察稳定性")
	}
	if o.DurationHours <= 0 {
		return Invalid("duration_hours", "稳定性观察时长必须大于 0")
	}
	if strings.TrimSpace(o.ObserverID) == "" || strings.TrimSpace(o.EvidenceRef) == "" {
		return Invalid("stability", "观察人和证据均必填")
	}
	if err := ValidateEvidenceRef(o.EvidenceRef); err != nil {
		return err
	}
	outcome, failures, err := EvaluateMetrics(o.Thresholds, o.Measurements)
	if err != nil {
		return err
	}
	o.Outcome = outcome
	o.Failures = failures
	c.StabilityObservations = append(c.StabilityObservations, o)
	totalHours := 0
	for _, existing := range c.StabilityObservations {
		totalHours += existing.DurationHours
	}
	if outcome == "passed" && totalHours >= 24 {
		c.State = StateStable
	} else {
		c.State = StateTreated
	}
	return nil
}

func (c *ConservationCase) ReleaseCase(reviewer, statement string, at time.Time) error {
	precheck := BuildReleasePrecheck(c, reviewer, statement)
	if !precheck.Eligible {
		return precheck.Error()
	}
	c.Release = &ReleaseSignature{ReviewerID: reviewer, Statement: statement, SignedAt: at}
	c.State = StateReleased
	return nil
}

func (c *ConservationCase) MarkArchived(m ArchiveManifest, at time.Time) error {
	if c.State != StateReleased {
		return Conflict("只有已放行个案可封存")
	}
	m.CaseID = c.ID
	c.Archive = &m
	c.ArchivedAt = &at
	c.State = StateArchived
	return nil
}
