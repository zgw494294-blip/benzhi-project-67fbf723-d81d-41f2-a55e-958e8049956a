package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"seed-germination-workbench/internal/domain"
	"sort"
	"strings"
	"sync"
)

type ConflictError struct {
	Kind    string        `json:"kind"`
	Field   string        `json:"field"`
	TrialID string        `json:"trialId"`
	Status  domain.Status `json:"status"`
	Message string        `json:"message"`
}

func (e *ConflictError) Error() string { return e.Message }

type Store struct {
	Dir  string
	mu   sync.RWMutex
	idem map[string][]byte
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir, idem: map[string][]byte{}}, nil
}

func (s *Store) Get(id string) (*domain.Trial, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getLocked(id)
}

func (s *Store) getLocked(id string) (*domain.Trial, error) {
	b, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var t domain.Trial
	if json.Unmarshal(b, &t) != nil {
		return nil, errors.New("快照损坏")
	}
	s.verifyIntegrityLocked(&t)
	return &t, nil
}

// verifyIntegrityLocked restores live integrity information for a trial
// snapshot by reading its event chain and review package files. This mirrors
// the checks performed in getLocked so that list-level reads expose the same
// read-only state as detail-level reads.
func (s *Store) verifyIntegrityLocked(t *domain.Trial) {
	events, readErr := readEvents(s.eventPath(t.TrialID))
	if readErr == nil && len(events) > 0 {
		t.Events = events
	}
	t.Integrity = domain.VerifyEvents(t.Events)
	if t.Integrity.Valid && t.Integrity.LastTrustedRevision != t.Revision {
		t.Integrity.Valid = false
		t.Integrity.FirstInvalidSequence = int64(len(t.Events) + 1)
		t.Integrity.Message = "事件历史与当前快照 revision 不一致"
	}
	if readErr != nil {
		t.Integrity.Valid = false
		t.Integrity.FirstInvalidSequence = int64(len(events) + 1)
		t.Integrity.Message = "事件文件无法完整解析：" + readErr.Error()
	}
	for _, p := range t.ReviewPackages {
		if domain.Digest(p.Snapshot) != p.SnapshotDigest {
			t.Integrity.Valid = false
			t.Integrity.Message = "审定包摘要异常：" + p.ReviewID
			break
		}
		packagePath := filepath.Join(s.Dir, t.TrialID+".reviews", fmt.Sprintf("submission-%04d.json", p.SubmissionNumber))
		packageBytes, packageErr := os.ReadFile(packagePath)
		var frozen domain.ReviewPackage
		if packageErr != nil || json.Unmarshal(packageBytes, &frozen) != nil || frozen.SnapshotDigest != p.SnapshotDigest || domain.Digest(frozen.Snapshot) != frozen.SnapshotDigest {
			t.Integrity.Valid = false
			t.Integrity.Message = "审定包文件缺失或摘要异常：" + p.ReviewID
			break
		}
	}
}

func (s *Store) Create(t *domain.Trial) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, err := s.getLocked(t.TrialID); err == nil {
		return &ConflictError{Kind: "TRIAL_ID", Field: "trialId", TrialID: current.TrialID, Status: current.Status, Message: "试验编号已存在"}
	} else if !os.IsNotExist(err) {
		return err
	}
	trials, err := s.listLocked()
	if err != nil {
		return err
	}
	for _, x := range trials {
		if strings.EqualFold(x.SpeciesName, t.SpeciesName) && strings.EqualFold(x.AccessionCode, t.AccessionCode) && strings.EqualFold(x.CollectionBatch, t.CollectionBatch) {
			return &ConflictError{Kind: "BATCH_IDENTITY", Field: "collectionBatch", TrialID: x.TrialID, Status: x.Status, Message: fmt.Sprintf("重复批次：已有试验 %s（%s）", x.TrialID, x.Status)}
		}
	}
	return s.writeAllLocked(t)
}

func (s *Store) LookupIdempotent(id, key string) (*domain.Trial, bool) {
	if key == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheKey := id + "\x00" + key
	b, ok := s.idem[cacheKey]
	if !ok {
		m := s.loadIdemLocked(id)
		b, ok = m[key]
	}
	if !ok {
		return nil, false
	}
	var t domain.Trial
	if json.Unmarshal(b, &t) != nil {
		return nil, false
	}
	return &t, true
}

func (s *Store) Save(t *domain.Trial, expected int64, key string) (*domain.Trial, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheKey := t.TrialID + "\x00" + key
	if key != "" {
		if b, ok := s.idem[cacheKey]; ok {
			var out domain.Trial
			_ = json.Unmarshal(b, &out)
			return &out, nil
		}
		if b, ok := s.loadIdemLocked(t.TrialID)[key]; ok {
			var out domain.Trial
			_ = json.Unmarshal(b, &out)
			s.idem[cacheKey] = b
			return &out, nil
		}
	}
	cur, err := s.getLocked(t.TrialID)
	if err != nil {
		return nil, err
	}
	if !cur.Integrity.Valid {
		return nil, fmt.Errorf("数据完整性错误：%s", cur.Integrity.Message)
	}
	if cur.Revision != expected {
		return nil, fmt.Errorf("版本冲突：当前 revision=%d", cur.Revision)
	}
	if t.Revision != expected+1 {
		return nil, errors.New("业务命令必须且只能推进一个 revision")
	}
	if err := s.writeAllLocked(t); err != nil {
		return nil, err
	}
	out, _ := json.Marshal(t)
	if key != "" {
		m := s.loadIdemLocked(t.TrialID)
		m[key] = out
		if err := writeJSONAtomic(s.idemPath(t.TrialID), m); err != nil {
			return nil, err
		}
		s.idem[cacheKey] = out
	}
	return t, nil
}

func (s *Store) writeAllLocked(t *domain.Trial) error {
	if status := domain.VerifyEvents(t.Events); !status.Valid {
		return errors.New("拒绝写入摘要链异常的事件")
	}
	if err := writeEventsAtomic(s.eventPath(t.TrialID), t.Events); err != nil {
		return err
	}
	if err := writeJSONAtomic(s.path(t.TrialID), t); err != nil {
		return err
	}
	if len(t.ReviewPackages) > 0 {
		dir := filepath.Join(s.Dir, t.TrialID+".reviews")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, p := range t.ReviewPackages {
			if domain.Digest(p.Snapshot) != p.SnapshotDigest {
				return errors.New("拒绝保存摘要异常的审定包")
			}
			path := filepath.Join(dir, fmt.Sprintf("submission-%04d.json", p.SubmissionNumber))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				if err := writeJSONAtomic(path, p); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeEventsAtomic(path string, events []domain.Event) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readEvents(path string) ([]domain.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	events := []domain.Event{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e domain.Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return events, err
		}
		events = append(events, e)
	}
	return events, scanner.Err()
}

func (s *Store) loadIdemLocked(id string) map[string][]byte {
	m := map[string][]byte{}
	b, err := os.ReadFile(s.idemPath(id))
	if err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func (s *Store) List() ([]domain.Trial, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listLocked()
}

func (s *Store) listLocked() ([]domain.Trial, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, err
	}
	out := []domain.Trial{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.Dir, entry.Name()))
		if err != nil {
			continue
		}
		var t domain.Trial
		if json.Unmarshal(b, &t) == nil {
			s.verifyIntegrityLocked(&t)
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].TrialID < out[j].TrialID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
