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
	store         Store
	evidence      EvidencePort
	clock         Clock
	ids           func() string
	timelineMu    sync.RWMutex
	timelineCache map[string]timelineCacheEntry
}

type timelineCacheEntry struct {
	revision int64
	events   []domain.AuditEvent
}

func NewService(store Store, evidence EvidencePort, clock Clock, idGenerator func() string) *Service {
	return &Service{store: store, evidence: evidence, clock: clock, ids: idGenerator, timelineCache: map[string]timelineCacheEntry{}}
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
	return s.store.Get(ctx, id)
}
func (s *Service) List(ctx context.Context) ([]domain.ConservationCase, error) {
	return s.store.List(ctx)
}

// mutate wraps Store.Mutate and invalidates the timeline cache whenever the
// mutation is not an idempotent replay. Rejected commands (e.g. ethics gate
// failures) persist audit events without advancing the case revision, so a
// revision-keyed cache would otherwise hide the newly appended events from
// subsequent timeline reads.
func (s *Service) mutate(ctx context.Context, id string, m CommandMeta, fn func(*domain.ConservationCase) error) (*domain.ConservationCase, bool, error) {
	item, replayed, err := s.store.Mutate(ctx, id, m, fn)
	if !replayed {
		s.invalidateTimeline(id)
	}
	return item, replayed, err
}

func (s *Service) invalidateTimeline(id string) {
	s.timelineMu.Lock()
	delete(s.timelineCache, id)
	s.timelineMu.Unlock()
}

func (s *Service) Timeline(ctx context.Context, id string) ([]domain.AuditEvent, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	s.timelineMu.RLock()
	cached, ok := s.timelineCache[id]
	s.timelineMu.RUnlock()
	if ok && cached.revision == item.Revision {
		return cloneAuditEvents(cached.events), nil
	}
	events, err := s.store.Events(ctx, id)
	if err != nil {
		return nil, err
	}
	s.timelineMu.Lock()
	s.timelineCache[id] = timelineCacheEntry{revision: item.Revision, events: cloneAuditEvents(events)}
	s.timelineMu.Unlock()
	return events, nil
}

func cloneAuditEvents(events []domain.AuditEvent) []domain.AuditEvent {
	cloned := append([]domain.AuditEvent(nil), events...)
	for i := range cloned {
		cloned[i].Payload = append([]byte(nil), cloned[i].Payload...)
	}
	return cloned
}
