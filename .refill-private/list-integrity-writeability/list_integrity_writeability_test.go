package list_integrity_writeability_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"seed-germination-workbench/internal/domain"
	"seed-germination-workbench/internal/store"
	"seed-germination-workbench/internal/workflow"
	"testing"
)

func TestSearchDoesNotAdvertiseTamperedTrialAsWritable(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	svc := workflow.New(s)
	trial, err := svc.Create(domain.Trial{TrialID: "tampered-trial", SpeciesName: "银杏", AccessionCode: "A01", CollectionBatch: "C01", ReplicateCount: 1, SeedsPerReplicate: 10})
	if err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(dir, trial.TrialID+".events")
	b, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatal(err)
	}
	var event domain.Event
	if err = json.Unmarshal(b, &event); err != nil {
		t.Fatal(err)
	}
	event.Summary = "被篡改的事件"
	b, err = json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	if err = os.WriteFile(eventPath, b, 0o644); err != nil {
		t.Fatal(err)
	}

	details, err := svc.Details(trial.TrialID)
	if err != nil {
		t.Fatal(err)
	}
	if readOnly, _ := details["readOnly"].(bool); !readOnly {
		t.Fatal("detail did not detect tampering")
	}
	result, err := svc.Search(workflow.ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items=%d", len(result.Items))
	}
	item := result.Items[0]
	if !item.ReadOnly || len(item.NextActions) != 1 || item.NextActions[0] != "VIEW" {
		t.Fatalf("search advertised tampered trial as writable: readOnly=%v nextActions=%v", item.ReadOnly, item.NextActions)
	}
}
