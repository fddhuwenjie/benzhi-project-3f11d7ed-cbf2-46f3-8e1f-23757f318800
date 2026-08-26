package application

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
)

func (s *Service) CreateCase(ctx context.Context, c CreateCaseCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	if c.ExpectedRevision != 0 {
		return nil, false, domain.Invalid("expected_revision", "创建时必须为 0")
	}
	m, err := s.meta(c.WriteContext, "case.created", c)
	if err != nil {
		return nil, false, err
	}
	now := s.clock.Now()
	item, err := domain.CreateCase(domain.NewCase{ID: s.ids(), ManuscriptCode: c.ManuscriptCode, Title: c.Title, CustodianID: c.CustodianID, SignificanceNote: c.SignificanceNote, TreatmentGoal: c.TreatmentGoal, InitialRisk: c.InitialRisk, RequiredRegions: c.RequiredRegions, CreatedAt: now})
	if err != nil {
		return nil, false, err
	}
	return s.store.Create(context.WithoutCancel(ctx), m, item)
}

func (s *Service) AddCondition(ctx context.Context, id string, c AddConditionCommand) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c.WriteContext, "condition.recorded", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(context.WithoutCancel(ctx), id, m, func(item *domain.ConservationCase) error {
		return item.AddCondition(domain.ConditionObservation{ID: s.ids(), LeafRef: c.LeafRef, RegionRef: c.RegionRef, Medium: c.Medium, DamageType: c.DamageType, Severity: c.Severity, Measurement: c.Measurement, EvidenceRef: c.EvidenceRef, RecordedBy: c.ActorID, RecordedAt: s.clock.Now()})
	})
}

func (s *Service) LockBaseline(ctx context.Context, id string, c WriteContext) (*domain.ConservationCase, bool, error) {
	if err := requireRole(c.Role, "conservator"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c, "baseline.locked", c)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(context.WithoutCancel(ctx), id, m, func(item *domain.ConservationCase) error { return item.LockBaseline() })
}
