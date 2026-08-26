package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"manuscript-conservation-gate/internal/domain"
	"strings"
	"sync"
)

type Service struct {
	store    Store
	evidence EvidencePort
	clock    Clock
	ids      func() string
	caseMu   sync.RWMutex
	caseByID map[string]*domain.ConservationCase
}

func NewService(store Store, evidence EvidencePort, clock Clock, idGenerator func() string) *Service {
	return &Service{store: store, evidence: evidence, clock: clock, ids: idGenerator, caseByID: map[string]*domain.ConservationCase{}}
}
func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func (s *Service) meta(w WriteContext, event string, body any) (CommandMeta, error) {
	if err := domain.ValidateIdentifier("request_id", w.RequestID); err != nil {
		return CommandMeta{}, err
	}
	if err := domain.ValidateIdentifier("actor_id", w.ActorID); err != nil {
		return CommandMeta{}, err
	}
	if strings.TrimSpace(w.Role) == "" {
		return CommandMeta{}, domain.Invalid("role", "不能为空")
	}
	return CommandMeta{RequestID: w.RequestID, ExpectedRevision: w.ExpectedRevision, ActorID: w.ActorID, Role: w.Role, Fingerprint: fingerprint(body), EventType: event}, nil
}
func requireRole(actual string, allowed ...string) error {
	for _, r := range allowed {
		if actual == r {
			return nil
		}
	}
	return domain.Forbidden("当前角色无权执行此操作")
}
func (s *Service) Get(ctx context.Context, id string) (*domain.ConservationCase, error) {
	s.caseMu.RLock()
	cached := s.caseByID[id]
	s.caseMu.RUnlock()
	if cached != nil {
		return cached, nil
	}
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	s.caseMu.Lock()
	s.caseByID[id] = item
	s.caseMu.Unlock()
	return item, nil
}
func (s *Service) List(ctx context.Context) ([]domain.ConservationCase, error) {
	return s.store.List(ctx)
}
func (s *Service) Timeline(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	return s.store.Events(ctx, id)
}
