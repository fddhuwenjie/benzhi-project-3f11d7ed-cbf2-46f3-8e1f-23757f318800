package crosscaseidempotency

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"manuscript-conservation-gate/internal/sqlstore"
)

func draftCase(t *testing.T, id string) *domain.ConservationCase {
	t.Helper()
	item, err := domain.CreateCase(domain.NewCase{
		ID: id, ManuscriptCode: "MS-" + id, Title: "测试手稿", CustodianID: "owner",
		SignificanceNote: "重要", TreatmentGoal: "稳定", InitialRisk: "脆化",
		RequiredRegions: []domain.RegionRequirement{{LeafRef: "1r", RegionRef: "center"}},
		CreatedAt:       time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func create(t *testing.T, store *sqlstore.Store, id, requestID string) *domain.ConservationCase {
	t.Helper()
	item, _, err := store.Create(context.Background(), application.CommandMeta{
		RequestID: requestID, ActorID: "worker", Role: "conservator",
		Fingerprint: "create-" + id, EventType: "case.created",
	}, draftCase(t, id))
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func TestRequestIDCannotReplayAcrossCases(t *testing.T) {
	store, err := sqlstore.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	first := create(t, store, "case-a", "create-a")
	second := create(t, store, "case-b", "create-b")
	shared := application.CommandMeta{
		RequestID: "shared-request", ExpectedRevision: 1, ActorID: "worker",
		Role: "conservator", Fingerprint: "identical-body", EventType: "case.touched",
	}
	if _, _, err = store.Mutate(context.Background(), first.ID, shared, func(*domain.ConservationCase) error { return nil }); err != nil {
		t.Fatal(err)
	}

	got, replayed, err := store.Mutate(context.Background(), second.ID, shared, func(*domain.ConservationCase) error { return nil })
	if errors.Is(err, application.ErrIdempotencyConflict) {
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if replayed || got.ID != second.ID {
		t.Fatalf("TestRequestIDCannotReplayAcrossCases: target=%s replayed=%v returned=%s", second.ID, replayed, got.ID)
	}
}
