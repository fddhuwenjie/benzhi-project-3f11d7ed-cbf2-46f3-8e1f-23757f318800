package application

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
)

func (s *Service) Archive(ctx context.Context, id string, c WriteContext) (*domain.ConservationCase, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if err := requireRole(c.Role, "custodian"); err != nil {
		return nil, false, err
	}
	m, err := s.meta(c, "case.archived", c)
	if err != nil {
		return nil, false, err
	}
	current, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if current.Revision != c.ExpectedRevision {
		return nil, false, ErrRevisionConflict
	}
	events, err := s.store.Events(ctx, id)
	if err != nil {
		return nil, false, err
	}
	head, _, err := s.store.VerifyAudit(ctx, id)
	if err != nil {
		return nil, false, err
	}
	manifest, err := s.evidence.Generate(ctx, current, events, head)
	if err != nil {
		return nil, false, err
	}
	return s.store.Mutate(ctx, id, m, func(item *domain.ConservationCase) error { return item.MarkArchived(manifest, s.clock.Now()) })
}
func (s *Service) ReadArchive(ctx context.Context, id string) ([]byte, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.State != domain.StateArchived {
		return nil, domain.Conflict("个案尚未封存")
	}
	return s.evidence.Read(ctx, id)
}
func (s *Service) VerifyArchive(ctx context.Context, id string) (VerificationResult, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return VerificationResult{}, err
	}
	if item.State != domain.StateArchived {
		return VerificationResult{}, domain.Conflict("个案尚未封存")
	}
	events, err := s.store.Events(ctx, id)
	if err != nil {
		return VerificationResult{}, err
	}
	return s.evidence.Verify(ctx, item, events)
}
