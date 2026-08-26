package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"net/http"
	"strconv"
)

const maxBody = 1 << 20

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Field   string `json:"field,omitempty"`
		Message string `json:"message"`
	} `json:"error"`
}

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.Invalid("body", "JSON 请求体无效: "+err.Error())
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return domain.Invalid("body", "只能包含一个 JSON 对象")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeCase(w http.ResponseWriter, status int, v any, revision int64, replayed bool) {
	w.Header().Set("ETag", fmt.Sprintf("\"revision-%d\"", revision))
	w.Header().Set("X-Case-Revision", strconv.FormatInt(revision, 10))
	if replayed {
		w.Header().Set("Idempotent-Replay", "true")
	}
	writeJSON(w, status, v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	field := ""
	message := "服务器内部错误"
	var de *domain.Error
	if errors.As(err, &de) {
		code = de.Code
		field = de.Field
		message = de.Message
		switch de.Code {
		case "validation_error":
			status = 422
		case "not_found":
			status = 404
		case "forbidden":
			status = 403
		default:
			status = 409
		}
	} else if errors.Is(err, application.ErrRevisionConflict) {
		status = 409
		code = "revision_conflict"
		message = err.Error()
	} else if errors.Is(err, application.ErrIdempotencyConflict) {
		status = 409
		code = "idempotency_conflict"
		message = err.Error()
	} else if errors.Is(err, application.ErrAuditCorrupt) {
		status = 500
		code = "audit_corrupt"
		message = err.Error()
	}
	var out errorBody
	out.Error.Code = code
	out.Error.Field = field
	out.Error.Message = message
	writeJSON(w, status, out)
}
func (a *API) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				a.log.Error("HTTP panic", "error", v)
				writeError(w, fmt.Errorf("panic: %v", v))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
