package timeline_missing_error_chain

import (
	"io"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/evidence"
	"manuscript-conservation-gate/internal/httpapi"
	"manuscript-conservation-gate/internal/sqlstore"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTimelineUnknownCaseReturnsNotFound(t *testing.T) {
	root := t.TempDir()
	store, err := sqlstore.Open(root + "/cases.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store, evidence.New(root+"/evidence"), application.RealClock{}, func() string { return "unused" })
	handler := httpapi.New(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conservation-cases/does-not-exist/timeline", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown timeline status=%d body=%s", rec.Code, rec.Body.String())
	}
}
