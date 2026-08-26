package application

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
)

func (s *Service) CompleteCheckpoint(ctx context.Context, id string, c CheckpointCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "checkpoint.completed", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(ctx, id, m, func(item *domain.ConservationCase) error {
		return item.CompleteCheckpoint(domain.TreatmentCheckpoint{ID: s.ids(), StepIndex: c.StepIndex, OperatorID: c.ActorID, ActualParameters: c.ActualParameters, EvidenceRef: c.EvidenceRef, CompletedAt: s.clock.Now()})
	})
}
func (s *Service) ResolveDeviation(ctx context.Context, id string, c ResolveDeviationCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "custodian", "reviewer"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "deviation.resolved", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(ctx, id, m, func(item *domain.ConservationCase) error {
		return item.ResolveDeviation(c.Impact, c.Remediation, c.VerifiedBy)
	})
}
func (s *Service) RecordStability(ctx context.Context, id string, c StabilityCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "stability.recorded", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(ctx, id, m, func(item *domain.ConservationCase) error {
		return item.AddStability(domain.StabilityObservation{ID: s.ids(), ObserverID: c.ActorID, DurationHours: c.DurationHours, Thresholds: c.Thresholds, Measurements: c.Measurements, EvidenceRef: c.EvidenceRef, ObservedAt: s.clock.Now()})
	})
}
func (s *Service) Release(ctx context.Context, id string, c ReleaseCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "reviewer"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "case.released", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(ctx, id, m, func(item *domain.ConservationCase) error {
		return item.ReleaseCase(c.ActorID, c.Statement, s.clock.Now())
	})
}
