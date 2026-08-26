package application

import (
	"context"
	"manuscript-conservation-gate/internal/domain"
	"strconv"
)

func (s *Service) Coverage(ctx context.Context, id string) (domain.CoverageReport, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.CoverageReport{}, err
	}
	key := id + ":" + strconv.FormatInt(item.Revision, 10)
	if cached, ok := s.coverageCache[key]; ok {
		return cached, nil
	}
	report := domain.BuildCoverageReport(item)
	s.coverageCache[key] = report
	return report, nil
}

func (s *Service) PlanHistory(ctx context.Context, id string) (domain.PlanHistoryReport, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.PlanHistoryReport{}, err
	}
	return domain.BuildPlanHistory(item), nil
}

func (s *Service) PlanDiff(ctx context.Context, id string, version int) (domain.PlanDiffReport, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.PlanDiffReport{}, err
	}
	return domain.BuildPlanDiff(item, version)
}

func (s *Service) Trials(ctx context.Context, id string) (domain.TrialReport, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.TrialReport{}, err
	}
	return domain.BuildTrialReport(item), nil
}

type TrialDetail struct {
	domain.CompatibilityTrial
	Revision       int64                        `json:"revision"`
	Expired        bool                         `json:"expired"`
	FailureDetails []domain.MetricFailureDetail `json:"failure_details"`
}

func (s *Service) TrialDetail(ctx context.Context, id, trialID string) (TrialDetail, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return TrialDetail{}, err
	}
	trial, err := domain.FindTrial(item, trialID)
	if err != nil {
		return TrialDetail{}, err
	}
	currentVersion := 0
	if current := item.CurrentPlan(); current != nil {
		currentVersion = current.Version
	}
	return TrialDetail{CompatibilityTrial: trial, Revision: item.Revision, Expired: trial.PlanVersion != currentVersion, FailureDetails: domain.ExplainMetricFailures(trial.Thresholds, trial.Measurements)}, nil
}

func (s *Service) ExecutionStatus(ctx context.Context, id string) (domain.ExecutionStatus, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.ExecutionStatus{}, err
	}
	return domain.BuildExecutionStatus(item), nil
}

func (s *Service) StabilityReport(ctx context.Context, id string) (domain.StabilityReport, error) {
	item, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.StabilityReport{}, err
	}
	return domain.BuildStabilityReport(item), nil
}
