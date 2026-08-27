package concurrentcoveragerace

import (
	"context"
	"errors"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"sync"
	"testing"
	"time"
)

type barrierStore struct {
	items   map[string]*domain.ConservationCase
	mu      sync.Mutex
	blocked bool
	entered int
	release chan struct{}
}

func (s *barrierStore) Get(_ context.Context, id string) (*domain.ConservationCase, error) {
	if s.blocked {
		s.mu.Lock()
		s.entered++
		if s.entered == 2 {
			close(s.release)
		}
		s.mu.Unlock()
		<-s.release
	}
	item, ok := s.items[id]
	if !ok {
		return nil, domain.NotFound("个案不存在")
	}
	copy := *item
	return &copy, nil
}
func (*barrierStore) Create(context.Context, application.CommandMeta, *domain.ConservationCase) (*domain.ConservationCase, bool, error) {
	return nil, false, errors.New("unused")
}
func (*barrierStore) Mutate(context.Context, string, application.CommandMeta, func(*domain.ConservationCase) error) (*domain.ConservationCase, bool, error) {
	return nil, false, errors.New("unused")
}
func (*barrierStore) List(context.Context) ([]domain.ConservationCase, error) {
	return nil, errors.New("unused")
}
func (*barrierStore) Events(context.Context, string) ([]domain.AuditEvent, error) {
	return nil, errors.New("unused")
}
func (*barrierStore) VerifyAudit(context.Context, string) (string, int64, error) {
	return "", 0, errors.New("unused")
}
func (*barrierStore) Close() error { return nil }

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(0, 0).UTC() }

func makeCase(t *testing.T, id string, revision int64) *domain.ConservationCase {
	t.Helper()
	item, err := domain.CreateCase(domain.NewCase{
		ID: id, ManuscriptCode: id, Title: "并发覆盖测试", CustodianID: "custodian",
		SignificanceNote: "重要", TreatmentGoal: "稳定", InitialRisk: "低",
		RequiredRegions: []domain.RegionRequirement{{LeafRef: "1r", RegionRef: "center"}},
		CreatedAt:       time.Unix(0, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	item.Revision = revision
	return item
}

func TestConcurrentCoverageRequestsRace(t *testing.T) {
	store := &barrierStore{items: map[string]*domain.ConservationCase{
		"case-a": makeCase(t, "case-a", 1),
		"case-b": makeCase(t, "case-b", 2),
	}}
	service := application.NewService(store, nil, fakeClock{}, func() string { return "unused" })
	if _, err := service.Coverage(context.Background(), "case-a"); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.blocked = true
	store.entered = 0
	store.release = make(chan struct{})
	store.mu.Unlock()
	var wg sync.WaitGroup
	for _, id := range []string{"case-a", "case-b"} {
		wg.Add(1)
		go func(caseID string) {
			defer wg.Done()
			if _, err := service.Coverage(context.Background(), caseID); err != nil {
				t.Errorf("coverage %s: %v", caseID, err)
			}
		}(id)
	}
	wg.Wait()
}
