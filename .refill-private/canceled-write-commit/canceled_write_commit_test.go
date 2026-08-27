package canceledwritecommit_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"manuscript-conservation-gate/internal/evidence"
	"manuscript-conservation-gate/internal/httpapi"
	"manuscript-conservation-gate/internal/sqlstore"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func TestCanceledMutationMustNotCommit(t *testing.T) {
	tmp := t.TempDir()
	store, err := sqlstore.Open(filepath.Join(tmp, "cases.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ids := []string{"case-cancel", "condition-cancel"}
	service := application.NewService(
		store,
		evidence.New(filepath.Join(tmp, "evidence")),
		fixedClock{now: time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)},
		func() string {
			id := ids[0]
			ids = ids[1:]
			return id
		},
	)
	handler := httpapi.New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	createBody := `{
		"request_id":"create-cancel-case",
		"expected_revision":0,
		"actor_id":"conservator-a",
		"role":"conservator",
		"manuscript_code":"MS-CANCEL-1",
		"title":"取消传播复现",
		"custodian_id":"custodian-a",
		"significance_note":"用于验证请求生命周期",
		"treatment_goal":"避免取消后写入",
		"initial_risk":"客户端已经放弃请求",
		"required_regions":[{"leaf_ref":"1r","region_ref":"all"}]
	}`
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/conservation-cases", strings.NewReader(createBody)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var initial domain.ConservationCase
	if err := json.Unmarshal(created.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}

	mutationBody := `{
		"request_id":"condition-after-cancel",
		"expected_revision":1,
		"actor_id":"conservator-a",
		"role":"conservator",
		"leaf_ref":"1r",
		"region_ref":"all",
		"medium":"iron-gall-ink",
		"damage_type":"flaking",
		"severity":3,
		"measurement":"edge loss 2mm",
		"evidence_ref":"sha256:canceled-request"
	}`
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/conservation-cases/"+initial.ID+"/conditions", strings.NewReader(mutationBody)).WithContext(canceled)
	mutation := httptest.NewRecorder()
	handler.ServeHTTP(mutation, request)

	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/conservation-cases/"+initial.ID, nil))
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body.String())
	}
	var stored domain.ConservationCase
	if err := json.Unmarshal(read.Body.Bytes(), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Revision != initial.Revision || len(stored.Conditions) != 0 {
		t.Fatalf("canceled mutation committed: write_status=%d revision=%d conditions=%d", mutation.Code, stored.Revision, len(stored.Conditions))
	}
}
