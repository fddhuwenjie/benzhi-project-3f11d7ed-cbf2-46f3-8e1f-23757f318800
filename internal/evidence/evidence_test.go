package evidence

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
	"os"
	"testing"
	"time"
)

func TestGenerateAndDetectTamper(t *testing.T) {
	dir := t.TempDir()
	m := New(dir)
	now := time.Now().UTC()
	m.now = func() time.Time { return now }
	item := &domain.ConservationCase{ID: "evidence-case", Revision: 7, State: domain.StateReleased, Release: &domain.ReleaseSignature{ReviewerID: "r", Statement: "通过", SignedAt: now}}
	payload := domain.EventPayload(map[string]string{"state": "released"})
	hash := domain.HashEvent("", 1, 1, item.ID, "created", "actor", payload)
	events := []domain.AuditEvent{{Sequence: 1, CaseID: item.ID, Revision: 1, Type: "created", ActorID: "actor", Payload: payload, Hash: hash}}
	manifest, err := m.Generate(context.Background(), item, events, hash)
	if err != nil {
		t.Fatal(err)
	}
	item.State = domain.StateArchived
	item.Archive = &manifest
	item.ArchivedAt = &now
	result, err := m.Verify(context.Background(), item, events)
	if err != nil || !result.Valid {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	raw, err := m.Read(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 4 {
		t.Fatal("证据文件异常短")
	}
	raw[len(raw)-3] = '0'
	if err = os.WriteFile(m.path(item.ID), raw, 0600); err != nil {
		t.Fatal(err)
	}
	result, err = m.Verify(context.Background(), item, events)
	if err == nil && result.Valid {
		t.Fatal("篡改后不应验证通过")
	}
}
