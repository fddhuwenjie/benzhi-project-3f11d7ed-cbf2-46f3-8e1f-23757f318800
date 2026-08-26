package httpapi

import (
	"bytes"
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStrictJSONAndPanicRecovery(t *testing.T) {
	a := &API{log: slog.Default()}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"request_id":"x","expected_revision":0,"actor_id":"a","role":"conservator","unknown":true}`))
	rec := httptest.NewRecorder()
	var command application.WriteContext
	if err := decode(rec, req, &command); err == nil {
		t.Fatal("未知字段应拒绝")
	}
	wrapped := a.recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("test") }))
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 500 {
		t.Fatalf("status=%d", rec.Code)
	}
}
