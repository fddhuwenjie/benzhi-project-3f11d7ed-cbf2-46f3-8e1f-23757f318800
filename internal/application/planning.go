package application

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
)

func (s *Service) SavePlan(ctx context.Context, id string, c SavePlanCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "plan.saved", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(context.WithoutCancel(ctx), id, m, func(item *domain.ConservationCase) error {
		return item.SavePlan(domain.TreatmentPlan{ID: s.ids(), Steps: c.Steps, ReversibilityNote: c.ReversibilityNote, TracePreservationNote: c.TracePreservationNote, RiskControls: c.RiskControls})
	})
}
func (s *Service) SubmitPlan(ctx context.Context, id string, c WriteContext) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c, "plan.submitted", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(context.WithoutCancel(ctx), id, m, func(item *domain.ConservationCase) error { return item.SubmitPlan(s.clock.Now()) })
}
func (s *Service) RecordTrial(ctx context.Context, id string, c TrialCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "trial.recorded", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(context.WithoutCancel(ctx), id, m, func(item *domain.ConservationCase) error {
		return item.AddTrial(domain.CompatibilityTrial{ID: s.ids(), PlanVersion: c.PlanVersion, MaterialCode: c.MaterialCode, Protocol: c.Protocol, Thresholds: c.Thresholds, Measurements: c.Measurements, EvidenceRef: c.EvidenceRef, ObservedAt: s.clock.Now()})
	})
}
func (s *Service) ReviewEthics(ctx context.Context, id string, c EthicsCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "custodian"); err != nil {
		return nil, false, err
	}
	eventType := "ethics.approved"
	if c.Decision == "return" {
		eventType = "ethics.returned"
	}
	m, err := s.meta(c.WriteContext, eventType, c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(context.WithoutCancel(ctx), id, m, func(item *domain.ConservationCase) error {
		return item.ReviewEthics(c.ActorID, c.Decision, c.Reason, s.clock.Now())
	})
}
