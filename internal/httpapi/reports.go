package httpapi

import (
	"manuscript-conservation-gate/internal/domain"
	"net/http"
	"strconv"
)

func validPathID(r *http.Request, name string) (string, error) {
	value := r.PathValue(name)
	if err := domain.ValidateIdentifier(name, value); err != nil {
		return "", err
	}
	return value, nil
}

func (a *API) GetCoverage(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.service.Coverage(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, report, report.Revision, false)
}

func (a *API) GetPlans(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.service.PlanHistory(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, report, report.Revision, false)
}

func (a *API) GetPlanDiff(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 {
		writeError(w, domain.Invalid("version", "必须为正整数"))
		return
	}
	report, err := a.service.PlanDiff(r.Context(), id, version)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, report, report.Revision, false)
}

func (a *API) GetTrials(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.service.Trials(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, report, report.Revision, false)
}

func (a *API) GetTrial(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	trialID, err := validPathID(r, "trial_id")
	if err != nil {
		writeError(w, err)
		return
	}
	detail, err := a.service.TrialDetail(r.Context(), id, trialID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, detail, detail.Revision, false)
}

func (a *API) GetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.service.ExecutionStatus(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, report, report.Revision, false)
}

func (a *API) GetStabilityReport(w http.ResponseWriter, r *http.Request) {
	id, err := validPathID(r, "case_id")
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.service.StabilityReport(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeCase(w, http.StatusOK, report, report.Revision, false)
}
