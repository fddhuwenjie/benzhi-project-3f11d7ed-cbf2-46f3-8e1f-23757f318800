package httpapi

import (
	"manuscript-conservation-gate/internal/application"
	"manuscript-conservation-gate/internal/domain"
	"net/http"
)

type executionResponse struct {
	*domain.ConservationCase
	ExecutionStatus domain.ExecutionStatus `json:"execution_status"`
}

type releaseResponse struct {
	*domain.ConservationCase
	ReleasePrecheck domain.ReleasePrecheck `json:"release_precheck"`
}

func (a *API) SavePlan(w http.ResponseWriter, r *http.Request) {
	var c application.SavePlanCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.SavePlan(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) SubmitPlan(w http.ResponseWriter, r *http.Request) {
	var c application.WriteContext
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.SubmitPlan(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) RecordTrial(w http.ResponseWriter, r *http.Request) {
	var c application.TrialCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.RecordTrial(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) ReviewEthics(w http.ResponseWriter, r *http.Request) {
	var c application.EthicsCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.ReviewEthics(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) CompleteCheckpoint(w http.ResponseWriter, r *http.Request) {
	var c application.CheckpointCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.CompleteCheckpoint(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, executionResponse{ConservationCase: item, ExecutionStatus: domain.BuildExecutionStatus(item)}, item.Revision, replay)
}
func (a *API) ResolveDeviation(w http.ResponseWriter, r *http.Request) {
	var c application.ResolveDeviationCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.ResolveDeviation(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, executionResponse{ConservationCase: item, ExecutionStatus: domain.BuildExecutionStatus(item)}, item.Revision, replay)
}
func (a *API) RecordStability(w http.ResponseWriter, r *http.Request) {
	var c application.StabilityCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.RecordStability(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) ReleaseCase(w http.ResponseWriter, r *http.Request) {
	var c application.ReleaseCommand
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.Release(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, releaseResponse{ConservationCase: item, ReleasePrecheck: domain.ReleasePrecheck{Eligible: true, Reasons: []domain.ReleaseReason{}}}, item.Revision, replay)
}
func (a *API) ArchiveCase(w http.ResponseWriter, r *http.Request) {
	var c application.WriteContext
	if err := decode(w, r, &c); err != nil {
		writeError(w, err)
		return
	}
	item, replay, err := a.service.Archive(r.Context(), r.PathValue("case_id"), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, 200, item, item.Revision, replay)
}
func (a *API) GetTimeline(w http.ResponseWriter, r *http.Request) {
	events, err := a.service.Timeline(r.Context(), r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": events, "count": len(events)})
}
func (a *API) GetArchive(w http.ResponseWriter, r *http.Request) {
	data, err := a.service.ReadArchive(r.Context(), r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=evidence.json")
	w.WriteHeader(200)
	_, _ = w.Write(data)
}
func (a *API) VerifyArchive(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.VerifyArchive(r.Context(), r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	status := 200
	if !result.Valid {
		status = 409
	}
	writeJSON(w, status, result)
}
