package stale_archive_verification_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"manuscript-conservation-gate/internal/evidence"
	"manuscript-conservation-gate/internal/httpapi"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type archiveStore struct {
	item   *domain.ConservationCase
	events []domain.AuditEvent
}

func (s *archiveStore) Create(context.Context, application.CommandMeta, *domain.ConservationCase) (*domain.ConservationCase, bool, error) {
	panic("not used")
}

func (s *archiveStore) Mutate(context.Context, string, application.CommandMeta, func(*domain.ConservationCase) error) (*domain.ConservationCase, bool, error) {
	panic("not used")
}

func (s *archiveStore) Get(context.Context, string) (*domain.ConservationCase, error) {
	return s.item, nil
}

func (s *archiveStore) List(context.Context) ([]domain.ConservationCase, error) {
	panic("not used")
}

func (s *archiveStore) Events(context.Context, string) ([]domain.AuditEvent, error) {
	return s.events, nil
}

func (s *archiveStore) VerifyAudit(context.Context, string) (string, int64, error) {
	panic("not used")
}

func (s *archiveStore) Close() error { return nil }

func verifyThroughHTTP(t *testing.T, handler http.Handler, id string) (int, application.VerificationResult) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conservation-cases/"+id+"/archive-verification", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	var result application.VerificationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析验证响应: status=%d body=%q err=%v", recorder.Code, recorder.Body.String(), err)
	}
	return recorder.Code, result
}

func TestArchiveVerificationMustRecheckEvidence(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	manager := evidence.New(dir)
	item := &domain.ConservationCase{
		ID:                    "case-cache-1",
		State:                 domain.StateArchived,
		Revision:              7,
		Plans:                 []domain.TreatmentPlan{},
		Trials:                []domain.CompatibilityTrial{},
		Checkpoints:           []domain.TreatmentCheckpoint{},
		StabilityObservations: []domain.StabilityObservation{},
	}
	manifest, err := manager.Generate(ctx, item, nil, "")
	if err != nil {
		t.Fatalf("生成测试证据包: %v", err)
	}
	item.Archive = &manifest
	store := &archiveStore{item: item, events: []domain.AuditEvent{}}
	service := application.NewService(store, manager, application.RealClock{}, func() string { return "unused" })
	handler := httpapi.New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	firstStatus, first := verifyThroughHTTP(t, handler, item.ID)
	if firstStatus != http.StatusOK || !first.Valid {
		t.Fatalf("首次验证应成功: status=%d result=%+v", firstStatus, first)
	}

	path := filepath.Join(dir, item.ID+".evidence.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取证据包: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析证据包: %v", err)
	}
	doc["manifest_digest"] = strings.Repeat("0", 64)
	raw, err = json.Marshal(doc)
	if err != nil {
		t.Fatalf("编码篡改证据包: %v", err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("篡改证据包: %v", err)
	}
	control, err := manager.Verify(ctx, item, nil)
	if err != nil {
		t.Fatalf("直接验证篡改证据包: %v", err)
	}
	if control.Valid {
		t.Fatal("测试前提失败：证据适配器未识别篡改")
	}

	secondStatus, second := verifyThroughHTTP(t, handler, item.ID)
	if secondStatus != http.StatusConflict || second.Valid {
		t.Fatalf("TestArchiveVerificationMustRecheckEvidence: 证据文件已篡改，但再次验证返回 status=%d valid=%v", secondStatus, second.Valid)
	}
}
