package staleaggregateread_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/evidence"
	"manuscript-conservation-gate/internal/httpapi"
	"manuscript-conservation-gate/internal/sqlstore"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

type caseResponse struct {
	ID         string `json:"id"`
	Revision   int64  `json:"revision"`
	Conditions []struct {
		ID string `json:"id"`
	} `json:"conditions"`
}

func request(t *testing.T, handler http.Handler, method, path string, body any, wantStatus int) caseResponse {
	t.Helper()
	var raw io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, raw)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status=%d body=%s", method, path, rec.Code, rec.Body.String())
	}
	var result caseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func TestAggregateReadMustReflectCommittedMutation(t *testing.T) {
	store, err := sqlstore.Open(filepath.Join(t.TempDir(), "case.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ids := []string{"case-cache", "condition-cache"}
	nextID := func() string {
		id := ids[0]
		ids = ids[1:]
		return id
	}
	service := application.NewService(store, evidence.New(t.TempDir()), application.RealClock{}, nextID)
	handler := httpapi.New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created := request(t, handler, http.MethodPost, "/api/v1/conservation-cases", map[string]any{
		"request_id": "create-cache-case", "expected_revision": 0, "actor_id": "worker-cache", "role": "conservator",
		"manuscript_code": "CACHE-1", "title": "缓存一致性测试", "custodian_id": "owner-cache",
		"significance_note": "重要手稿", "treatment_goal": "稳定保存", "initial_risk": "纸张脆弱",
		"required_regions": []map[string]string{{"leaf_ref": "1r", "region_ref": "all"}},
	}, http.StatusCreated)
	path := "/api/v1/conservation-cases/" + created.ID

	before := request(t, handler, http.MethodGet, path, nil, http.StatusOK)
	if before.Revision != 1 {
		t.Fatalf("initial revision=%d", before.Revision)
	}
	updated := request(t, handler, http.MethodPost, path+"/conditions", map[string]any{
		"request_id": "record-cache-condition", "expected_revision": 1, "actor_id": "worker-cache", "role": "conservator",
		"leaf_ref": "1r", "region_ref": "all", "medium": "纸本", "damage_type": "脆化", "severity": 2,
		"measurement": "边缘轻微脆化", "evidence_ref": "sha256:cache-condition",
	}, http.StatusOK)
	if updated.Revision != 2 || len(updated.Conditions) != 1 {
		t.Fatalf("mutation response revision=%d conditions=%d", updated.Revision, len(updated.Conditions))
	}

	after := request(t, handler, http.MethodGet, path, nil, http.StatusOK)
	if after.Revision != 2 || len(after.Conditions) != 1 {
		t.Fatalf("committed mutation is hidden by aggregate cache: revision=%d conditions=%d", after.Revision, len(after.Conditions))
	}
}
