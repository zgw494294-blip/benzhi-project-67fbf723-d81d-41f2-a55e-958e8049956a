package snapshotcachealiasing_test

import (
	"seed-germination-workbench/internal/domain"
	"seed-germination-workbench/internal/store"
	"seed-germination-workbench/internal/workflow"
	"testing"
)

func TestCachedTrialSnapshotRemainsImmutable(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := workflow.New(st)
	trial, err := svc.Create(domain.Trial{
		TrialID:           "cache-alias-trial",
		SpeciesName:       "银杏",
		AccessionCode:     "A01",
		CollectionBatch:   "CACHE-01",
		ReplicateCount:    1,
		SeedsPerReplicate: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	protocol := domain.TreatmentProtocol{
		ObservationDays:             1,
		TemperatureCelsius:          25,
		LightRegime:                 "16h",
		Substrate:                   "培养基",
		GerminationThresholdPercent: 50,
	}
	preview, err := svc.PreviewProtocol(trial.TrialID, protocol)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := svc.LockProtocolChecked(trial.TrialID, "lock-cache-alias", trial.Revision, preview.ContentDigest, "研究员", protocol)
	if err != nil {
		t.Fatal(err)
	}

	readSnapshot, err := svc.Get(trial.TrialID)
	if err != nil {
		t.Fatal(err)
	}
	wantStatus := readSnapshot.Status
	wantRevision := readSnapshot.Revision
	wantEventCount := len(readSnapshot.Events)

	if _, err = svc.StartChecked(trial.TrialID, "start-cache-alias", locked.Revision, "观察员"); err != nil {
		t.Fatal(err)
	}
	if readSnapshot.Status != wantStatus || readSnapshot.Revision != wantRevision || len(readSnapshot.Events) != wantEventCount {
		t.Fatalf("cached read snapshot was mutated by a later write: status=%s revision=%d events=%d; want status=%s revision=%d events=%d",
			readSnapshot.Status, readSnapshot.Revision, len(readSnapshot.Events), wantStatus, wantRevision, wantEventCount)
	}
}
