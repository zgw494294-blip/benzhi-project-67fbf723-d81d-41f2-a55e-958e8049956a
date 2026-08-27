package domain

import "testing"

func TestTrialLifecycle(t *testing.T) {
	x, err := NewTrial("t1", "银杏", "A", "B", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err = x.LockProtocol(TreatmentProtocol{ObservationDays: 2, TemperatureCelsius: 25, LightRegime: "16h", Substrate: "培养基", GerminationThresholdPercent: 50}); err != nil {
		t.Fatal(err)
	}
	if err = x.StartObserving(); err != nil {
		t.Fatal(err)
	}
	for d := 1; d <= 2; d++ {
		for r := 1; r <= 2; r++ {
			if err = x.AddObservation(DailyObservation{DayIndex: d, ReplicateIndex: r, NewlyGerminated: 5}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err = x.ReadyForReview(); err != nil {
		t.Fatal(err)
	}
	if err = x.Review("APPROVE", "合格", "审定员"); err != nil {
		t.Fatal(err)
	}
	if x.Status != Closed {
		t.Fatalf("status=%s", x.Status)
	}
}

func TestObservationValidation(t *testing.T) {
	x, _ := NewTrial("t2", "植物", "", "", 1, 5)
	if err := x.AddObservation(DailyObservation{DayIndex: 1, ReplicateIndex: 1, NewlyGerminated: 6}); err == nil {
		t.Fatal("expected count validation")
	}
}

func TestProtocolPreviewReturnsAllIssuesWithoutMutation(t *testing.T) {
	x, _ := NewTrial(" t3 ", " 银杏 ", " a01 ", " c01 ", 2, 10)
	preview := PreviewProtocol(x.TrialID, x.Revision, TreatmentProtocol{StratificationDays: -1, LightRegime: "16h", ObservationDays: 2, GerminationThresholdPercent: 101})
	if len(preview.Issues) != 3 {
		t.Fatalf("issues=%+v", preview.Issues)
	}
	if x.Status != Draft || x.Revision != 1 || len(x.Events) != 1 {
		t.Fatal("预检不应改变聚合")
	}
	if x.AccessionCode != "A01" || x.CollectionBatch != "C01" {
		t.Fatal("身份字段未规范化")
	}
}

func TestObservationBatchIsAtomicAndMetricsHaveEvidence(t *testing.T) {
	x, _ := NewTrial("t4", "银杏", "A01", "C01", 3, 20)
	_ = x.LockProtocol(TreatmentProtocol{ObservationDays: 2, TemperatureCelsius: 25, LightRegime: "16h", Substrate: "培养基", GerminationThresholdPercent: 20})
	_ = x.StartObserving()
	before := x.Revision
	err := x.AddObservationBatch(1, []DailyObservation{{ReplicateIndex: 1, NewlyGerminated: 12}, {ReplicateIndex: 2, NewlyGerminated: 5}}, "观察员")
	if err == nil || len(x.Observations) != 0 || x.Revision != before {
		t.Fatal("缺组批次不应部分写入")
	}
	day1 := []DailyObservation{{ReplicateIndex: 1, NewlyGerminated: 12, NewlyNonviable: 6}, {ReplicateIndex: 2, NewlyGerminated: 5}, {ReplicateIndex: 3, NewlyGerminated: 4}}
	if err = x.AddObservationBatch(1, day1, "观察员"); err != nil {
		t.Fatal(err)
	}
	day2 := []DailyObservation{{ReplicateIndex: 1, NewlyGerminated: 3}, {ReplicateIndex: 2}, {ReplicateIndex: 3}}
	err = x.AddObservationBatch(2, day2, "观察员")
	batchErr, ok := err.(*ObservationBatchError)
	if !ok || batchErr.RemainingByGroup[1] != 2 || len(x.Observations) != 3 {
		t.Fatalf("余量校验或原子性错误: %#v", err)
	}
	day2[0].NewlyGerminated = 2
	if err = x.AddObservationBatch(2, day2, "观察员"); err != nil {
		t.Fatal(err)
	}
	m := x.Metrics()
	if m.FormulaInputs.TotalSeeds != 60 || len(m.ReplicateDetails) != 3 || len(m.CumulativeCurve) != 2 {
		t.Fatalf("指标证据不完整: %+v", m)
	}
}

func TestSubmissionFreezesPackageAndRequiresChecklist(t *testing.T) {
	x, _ := NewTrial("t5", "银杏", "A01", "C02", 1, 10)
	_ = x.LockProtocol(TreatmentProtocol{ObservationDays: 1, TemperatureCelsius: 25, LightRegime: "16h", Substrate: "培养基", GerminationThresholdPercent: 50})
	_ = x.StartObserving()
	_ = x.AddObservationBatch(1, []DailyObservation{{ReplicateIndex: 1, NewlyGerminated: 5}}, "观察员")
	if err := x.SubmitForReview("研究员"); err != nil {
		t.Fatal(err)
	}
	if len(x.ReviewPackages) != 1 || domainDigestMismatch(x.ReviewPackages[0]) {
		t.Fatal("送审时未固化有效审定包")
	}
	p := x.ReviewPackages[0]
	if err := x.ReviewWithChecklist("RETURN", "需补证", "审定员", p.SnapshotDigest, nil); err == nil || x.Status != ReviewPending {
		t.Fatal("空退回清单不应改变状态")
	}
	issues := []ReviewIssueInput{{Category: "DATA", Description: "核对指标", RequiredAction: "重新计算", ObjectType: "metrics", ObjectID: "metrics"}}
	if err := x.ReviewWithChecklist("RETURN", "需补证", "审定员", p.SnapshotDigest, issues); err != nil || x.Status != CorrectionRequired {
		t.Fatal(err)
	}
}

func domainDigestMismatch(p ReviewPackage) bool { return Digest(p.Snapshot) != p.SnapshotDigest }
