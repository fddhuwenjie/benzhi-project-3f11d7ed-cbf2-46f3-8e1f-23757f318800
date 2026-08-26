package httpapi

import (
	"manuscript-conservation-gate/internal/application"
	"net/http"
)

func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *API) CreateCase(w http.ResponseWriter, r *http.Request) {
	var c application.CreateCaseCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.CreateCase(r.Context(), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 201, item, item.Revision, replay)
}
func (a *API) ListCases(w http.ResponseWriter, r *http.Request) {
	items, err := a.service.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "count": len(items)})
}
func (a *API) GetCase(w http.ResponseWriter, r *http.Request) {
	item, err := a.service.Get(r.Context(), r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, false)
}
func (a *API) AddCondition(w http.ResponseWriter, r *http.Request) {
	var c application.AddConditionCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.AddCondition(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) LockBaseline(w http.ResponseWriter, r *http.Request) {
	var c application.WriteContext
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.LockBaseline(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
