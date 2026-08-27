package rejected_command_idempotency

import (
	"context"
	"errors"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"manuscript-conservation-gate/internal/sqlstore"
	"path/filepath"
	"testing"
)

func TestRejectedCommandRetryMustBeIdempotent(t *testing.T) {
	ctx := context.Background()
	store, err := sqlstore.Open(filepath.Join(t.TempDir(), "cases.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	id := 0
	service := application.NewService(store, nil, application.RealClock{}, func() string {
		id++
		return "case-" + string(rune('0'+id))
	})
	created, _, err := service.CreateCase(ctx, application.CreateCaseCommand{
		WriteContext:   application.WriteContext{RequestID: "create-1", ActorID: "conservator-1", Role: "conservator"},
		ManuscriptCode: "MS-1", Title: "手稿", CustodianID: "custodian-1",
		SignificanceNote: "重要", TreatmentGoal: "稳定", InitialRisk: "中等",
		RequiredRegions: []domain.RegionRequirement{{LeafRef: "leaf-1", RegionRef: "region-a"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	cmd := application.EthicsCommand{
		WriteContext: application.WriteContext{RequestID: "ethics-retry-1", ExpectedRevision: created.Revision, ActorID: "custodian-1", Role: "custodian"},
		Decision:     "approve",
	}
	_, _, firstErr := service.ReviewEthics(ctx, created.ID, cmd)
	if firstErr == nil {
		t.Fatal("未通过试验的伦理审查应被拒绝")
	}
	var domainErr *domain.Error
	if !errors.As(firstErr, &domainErr) {
		t.Fatalf("应返回领域错误，得到 %v", firstErr)
	}

	_, _, secondErr := service.ReviewEthics(ctx, created.ID, cmd)
	if secondErr == nil {
		t.Fatal("重试仍应返回原始拒绝")
	}
	events, err := service.Timeline(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("相同 request_id 的拒绝重试不应重复追加审计事件，事件数=%d", len(events))
	}
	if events[1].Type != "ethics.gate_blocked" || events[1].Revision != created.Revision {
		t.Fatalf("拒绝审计事件不符合预期: %+v", events[1])
	}
}
