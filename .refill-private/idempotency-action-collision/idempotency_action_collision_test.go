package idempotency_action_collision_test

import (
	"seed-germination-workbench/internal/domain"
	"seed-germination-workbench/internal/store"
	"seed-germination-workbench/internal/workflow"
	"strings"
	"testing"
)

func TestIdempotencyKeyCannotReplayDifferentAction(t *testing.T) {
	s, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := workflow.New(s)
	trial, err := svc.Create(domain.Trial{TrialID: "idem-trial", SpeciesName: "银杏", AccessionCode: "A01", CollectionBatch: "C01", ReplicateCount: 1, SeedsPerReplicate: 10})
	if err != nil {
		t.Fatal(err)
	}
	protocol := domain.TreatmentProtocol{ObservationDays: 1, TemperatureCelsius: 25, LightRegime: "16h", Substrate: "培养基", GerminationThresholdPercent: 50}
	preview, err := svc.PreviewProtocol(trial.TrialID, protocol)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := svc.LockProtocolChecked(trial.TrialID, "shared-key", trial.Revision, preview.ContentDigest, "研究员", protocol)
	if err != nil {
		t.Fatal(err)
	}

	started, startErr := svc.StartChecked(trial.TrialID, "shared-key", locked.Revision, "观察员")
	if startErr != nil {
		if !strings.Contains(startErr.Error(), "幂等") && !strings.Contains(startErr.Error(), "冲突") {
			t.Fatalf("different action returned unrelated error: %v", startErr)
		}
		return
	}
	persisted, err := svc.Get(trial.TrialID)
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != domain.Observing || persisted.Status != domain.Observing {
		t.Fatalf("different action falsely replayed cached protocol result: response=%s persisted=%s", started.Status, persisted.Status)
	}
}
