package stalerejectedaudit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"manuscript-conservation-gate/internal/httpapi"
	"manuscript-conservation-gate/internal/sqlstore"
)

type timelineResponse struct {
	Items []domain.AuditEvent `json:"items"`
	Count int                 `json:"count"`
}

func request(t *testing.T, client *http.Client, method, url string, body any, want int, dst any) {
	t.Helper()
	var raw []byte
	var err error
	if body != nil {
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s 返回 %d，期望 %d: %s", method, url, resp.StatusCode, want, responseBody)
	}
	if dst != nil {
		if err := json.Unmarshal(responseBody, dst); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRejectedAuditMustInvalidateTimeline(t *testing.T) {
	store, err := sqlstore.Open(filepath.Join(t.TempDir(), "case.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := application.NewService(store, nil, application.RealClock{}, func() string { return "case-timeline-cache" })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(httpapi.New(service, logger))
	defer server.Close()

	create := map[string]any{
		"request_id": "request-create-cache-case", "expected_revision": 0,
		"actor_id": "conservator-1", "role": "conservator",
		"manuscript_code": "MS-CACHE-1", "title": "拒绝审计缓存复现",
		"custodian_id": "custodian-1", "significance_note": "用于验证拒绝操作也必须可追溯",
		"treatment_goal": "保持审计时间线完整", "initial_risk": "缓存遗漏拒绝事实",
		"required_regions": []map[string]string{{"leaf_ref": "1r", "region_ref": "center"}},
	}
	var item domain.ConservationCase
	request(t, server.Client(), http.MethodPost, server.URL+"/api/v1/conservation-cases", create, http.StatusCreated, &item)
	timelineURL := server.URL + "/api/v1/conservation-cases/" + item.ID + "/timeline"
	var before timelineResponse
	request(t, server.Client(), http.MethodGet, timelineURL, nil, http.StatusOK, &before)
	if before.Count != 1 || len(before.Items) != 1 {
		t.Fatalf("复现前 timeline=%#v，期望 1 条创建事件", before)
	}

	rejected := map[string]any{
		"request_id": "request-rejected-ethics", "expected_revision": item.Revision,
		"actor_id": "custodian-1", "role": "custodian", "decision": "approve",
		"reason": "试图在材料试验前批准",
	}
	request(t, server.Client(), http.MethodPost, server.URL+"/api/v1/conservation-cases/"+item.ID+"/ethics-review", rejected, http.StatusConflict, nil)
	persisted, err := store.Events(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 2 || persisted[1].Type != "ethics.gate_blocked" {
		t.Fatalf("拒绝审计未按预期持久化: %#v", persisted)
	}

	var after timelineResponse
	request(t, server.Client(), http.MethodGet, timelineURL, nil, http.StatusOK, &after)
	if after.Count != len(persisted) || len(after.Items) != len(persisted) {
		t.Fatalf("Timeline 返回 %d 条事件，持久层已有 %d 条；同 revision 的拒绝审计被缓存遗漏", after.Count, len(persisted))
	}
}
