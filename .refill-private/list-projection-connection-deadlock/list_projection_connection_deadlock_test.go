package listprojectionconnectiondeadlock_test

import (
	"context"
	"io"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"manuscript-conservation-gate/internal/httpapi"
	"manuscript-conservation-gate/internal/sqlstore"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestListMustNotDeadlockDuringProjectionVerification(t *testing.T) {
	store, err := sqlstore.Open(filepath.Join(t.TempDir(), "list.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	item, err := domain.CreateCase(domain.NewCase{
		ID:               "case-list-deadlock",
		ManuscriptCode:   "MS-LIST-001",
		Title:            "集合读取连接生命周期复现",
		CustodianID:      "custodian-list",
		SignificanceNote: "用于验证集合读取时的投影完整性",
		TreatmentGoal:    "保持读取接口可用",
		InitialRisk:      "连接被嵌套查询占用",
		RequiredRegions:  []domain.RegionRequirement{{LeafRef: "1r", RegionRef: "center"}},
		CreatedAt:        time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	meta := application.CommandMeta{
		RequestID:   "create-list-deadlock",
		ActorID:     "conservator-list",
		Role:        "conservator",
		Fingerprint: "create-list-deadlock",
		EventType:   "case.created",
	}
	if _, _, err = store.Create(context.Background(), meta, item); err != nil {
		t.Fatal(err)
	}

	service := application.NewService(store, nil, application.RealClock{}, func() string { return "unused" })
	handler := httpapi.New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{Timeout: 500 * time.Millisecond}
	response, err := client.Get(server.URL + "/api/v1/conservation-cases")
	if err != nil {
		t.Fatalf("集合读取不应等待 context 超时: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("集合读取状态码=%d，期望 %d", response.StatusCode, http.StatusOK)
	}
}
