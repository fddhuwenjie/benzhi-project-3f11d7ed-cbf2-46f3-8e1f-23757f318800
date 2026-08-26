package httpapi

import (
	"log/slog"
	"manuscript-conservation-gate/internal/application"
	"net/http"
)

type API struct {
	service *application.Service
	log     *slog.Logger
	mux     *http.ServeMux
}

func New(service *application.Service, log *slog.Logger) http.Handler {
	a := &API{service: service, log: log, mux: http.NewServeMux()}
	a.routes()
	return a.recover(a.mux)
}
func (a *API) routes() {
	a.mux.HandleFunc("GET /healthz", a.Health)
	a.mux.HandleFunc("GET /api/v1/conservation-cases", a.ListCases)
	a.mux.HandleFunc("POST /api/v1/conservation-cases", a.CreateCase)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}", a.GetCase)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/conditions", a.AddCondition)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/conditions/coverage", a.GetCoverage)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/baseline-lock", a.LockBaseline)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/plans", a.SavePlan)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/plans", a.GetPlans)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/plans/{version}/diff", a.GetPlanDiff)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/plan-submit", a.SubmitPlan)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/trials", a.RecordTrial)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/trials", a.GetTrials)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/trials/{trial_id}", a.GetTrial)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/ethics-review", a.ReviewEthics)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/checkpoints", a.CompleteCheckpoint)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/deviation-resolution", a.ResolveDeviation)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/execution-status", a.GetExecutionStatus)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/stability", a.RecordStability)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/stability/report", a.GetStabilityReport)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/release", a.ReleaseCase)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/archive", a.ArchiveCase)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/timeline", a.GetTimeline)
	a.mux.HandleFunc("GET /api/v1/conservation-cases/{case_id}/archive", a.GetArchive)
	a.mux.HandleFunc("POST /api/v1/conservation-cases/{case_id}/archive-verification", a.VerifyArchive)
}
