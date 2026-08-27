package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"seed-germination-workbench/internal/domain"
	"testing"
)

func TestDuplicateIdentityAndEventTamperBecomeReadOnly(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, _ := domain.NewTrial("trial-1", "银杏", "A01", "C01", 2, 10)
	if err = s.Create(first); err != nil {
		t.Fatal(err)
	}
	duplicate, _ := domain.NewTrial("trial-2", "银杏", " a01 ", " c01 ", 2, 10)
	if err = s.Create(duplicate); err == nil {
		t.Fatal("相同物种、种质和采集批号应被拒绝")
	}

	eventFile := filepath.Join(s.Dir, "trial-1.events")
	b, err := os.ReadFile(eventFile)
	if err != nil {
		t.Fatal(err)
	}
	var event domain.Event
	if err = json.Unmarshal(b, &event); err != nil {
		t.Fatal(err)
	}
	event.Summary = "被改动的事件"
	tampered, _ := json.Marshal(event)
	tampered = append(tampered, '\n')
	if err = os.WriteFile(eventFile, tampered, 0644); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Get("trial-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Integrity.Valid || loaded.Integrity.FirstInvalidSequence != 1 || loaded.Integrity.LastTrustedRevision != 0 {
		t.Fatalf("未识别事件摘要异常: %+v", loaded.Integrity)
	}
}
