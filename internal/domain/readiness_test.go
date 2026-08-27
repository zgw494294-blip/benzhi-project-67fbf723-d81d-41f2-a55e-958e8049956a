package domain

import "testing"

func TestReviewReadinessReportsProgressAndBlockers(t *testing.T) {
	trial, err := NewTrial("ready-1", "银杏", "A01", "C01", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err = trial.LockProtocol(TreatmentProtocol{
		ObservationDays:             2,
		TemperatureCelsius:          25,
		LightRegime:                 "16h",
		Substrate:                   "培养基",
		GerminationThresholdPercent: 50,
	}); err != nil {
		t.Fatal(err)
	}
	if err = trial.StartObserving(); err != nil {
		t.Fatal(err)
	}
	if err = trial.AddObservationBatch(1, []DailyObservation{
		{ReplicateIndex: 1, NewlyGerminated: 5},
		{ReplicateIndex: 2, NewlyGerminated: 5},
	}, "观察员"); err != nil {
		t.Fatal(err)
	}

	report := AssessReviewReadiness(trial, trial.Metrics())
	if report.Eligible || report.ObservationComplete {
		t.Fatal("观察周期未完成时不应具备送审资格")
	}
	if report.Progress.ExpectedCellCount != 4 || report.Progress.RecordedCellCount != 2 {
		t.Fatalf("观察进度错误: %+v", report.Progress)
	}
	if report.Progress.PendingCellCount != 2 || report.Progress.MissingCellCount != 0 {
		t.Fatalf("未来观察缺口分类错误: %+v", report.Progress)
	}
	if len(report.Progress.CompletedDays) != 1 || report.Progress.CompletedDays[0] != 1 {
		t.Fatalf("完整日期错误: %+v", report.Progress.CompletedDays)
	}
}

func TestReviewReadinessBecomesEligibleAfterCompleteObservation(t *testing.T) {
	trial, _ := NewTrial("ready-2", "银杏", "A01", "C02", 1, 10)
	_ = trial.LockProtocol(TreatmentProtocol{
		ObservationDays:             1,
		TemperatureCelsius:          25,
		LightRegime:                 "16h",
		Substrate:                   "培养基",
		GerminationThresholdPercent: 50,
	})
	_ = trial.StartObserving()
	_ = trial.AddObservationBatch(1, []DailyObservation{{
		ReplicateIndex:  1,
		NewlyGerminated: 5,
	}}, "观察员")

	report := AssessReviewReadiness(trial, trial.Metrics())
	if !report.Eligible || len(report.Blockers) != 0 {
		t.Fatalf("完整有效观察应允许送审: %+v", report)
	}
	if report.Progress.CompletionPercent != 100 || !report.ThresholdMet {
		t.Fatalf("完成率或阈值结论错误: %+v", report)
	}
}
