package sqlstore

import (
	"context"
	"errors"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"path/filepath"
	"testing"
	"time"
)

func storedCase(t *testing.T) *domain.ConservationCase {
	t.Helper()
	c, err := domain.CreateCase(domain.NewCase{ID: "case-db", ManuscriptCode: "DB-1", Title: "测试", CustodianID: "owner", SignificanceNote: "重要", TreatmentGoal: "稳定", InitialRisk: "风险", RequiredRegions: []domain.RegionRequirement{{LeafRef: "1r", RegionRef: "all"}}, CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestIdempotencyRevisionAndAudit(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m := application.CommandMeta{RequestID: "r1", ExpectedRevision: 0, ActorID: "worker", Role: "conservator", Fingerprint: "fp1", EventType: "case.created"}
	created, replay, err := s.Create(ctx, m, storedCase(t))
	if err != nil || replay {
		t.Fatalf("create: %v %v", replay, err)
	}
	again, replay, err := s.Create(ctx, m, storedCase(t))
	if err != nil || !replay || again.ID != created.ID {
		t.Fatalf("replay: %v %v", replay, err)
	}
	m.Fingerprint = "different"
	if _, _, err = s.Create(ctx, m, storedCase(t)); !errors.Is(err, application.ErrIdempotencyConflict) {
		t.Fatalf("err=%v", err)
	}
	mut := application.CommandMeta{RequestID: "r2", ExpectedRevision: 0, ActorID: "worker", Role: "conservator", Fingerprint: "fp2", EventType: "condition.recorded"}
	if _, _, err = s.Mutate(ctx, created.ID, mut, func(*domain.ConservationCase) error { return nil }); !errors.Is(err, application.ErrRevisionConflict) {
		t.Fatalf("err=%v", err)
	}
	head, count, err := s.VerifyAudit(ctx, created.ID)
	if err != nil || head == "" || count != 1 {
		t.Fatalf("audit: %s %d %v", head, count, err)
	}
}
func TestAuditTamperDetected(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	m := application.CommandMeta{RequestID: "r1", ActorID: "worker", Fingerprint: "fp", EventType: "case.created"}
	c, _, err := s.Create(ctx, m, storedCase(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`UPDATE audit_events SET payload='tampered' WHERE case_id=? AND sequence=1`, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.VerifyAudit(ctx, c.ID); !errors.Is(err, application.ErrAuditCorrupt) {
		t.Fatalf("err=%v", err)
	}
	if _, err = s.Events(ctx, c.ID); !errors.Is(err, application.ErrAuditCorrupt) {
		t.Fatalf("read err=%v", err)
	}
}

func TestRejectedReleaseIsAuditedWithoutRevisionChange(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c, _, err := s.Create(ctx, application.CommandMeta{RequestID: "create", ActorID: "worker", Fingerprint: "create", EventType: "case.created"}, storedCase(t))
	if err != nil {
		t.Fatal(err)
	}
	release := application.CommandMeta{RequestID: "release", ExpectedRevision: c.Revision, ActorID: "reviewer", Fingerprint: "release", EventType: "case.released"}
	if _, _, err = s.Mutate(ctx, c.ID, release, func(*domain.ConservationCase) error { return domain.Conflict("稳定性门禁未通过") }); err == nil {
		t.Fatal("放行应被拒绝")
	}
	stored, err := s.Get(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Revision != c.Revision || stored.State != domain.StateDraft {
		t.Fatalf("case=%+v", stored)
	}
	events, err := s.Events(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[1].Revision != c.Revision || events[1].Type != "release.precheck_rejected" {
		t.Fatalf("events=%+v", events)
	}
	next := application.CommandMeta{RequestID: "next", ExpectedRevision: c.Revision, ActorID: "worker", Fingerprint: "next", EventType: "case.touched"}
	if _, _, err = s.Mutate(ctx, c.ID, next, func(*domain.ConservationCase) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, count, err := s.VerifyAudit(ctx, c.ID); err != nil || count != 3 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
