package workflow

import (
	"errors"
	"fmt"
	"seed-germination-workbench/internal/domain"
	"seed-germination-workbench/internal/store"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	Store *store.Store
	locks sync.Map
}

type ListFilter struct {
	Species         string
	CollectionBatch string
	Status          domain.Status
}

type TrialSummary struct {
	TrialID         string        `json:"trialId"`
	SpeciesName     string        `json:"speciesName"`
	AccessionCode   string        `json:"accessionCode"`
	CollectionBatch string        `json:"collectionBatch"`
	Status          domain.Status `json:"status"`
	Revision        int64         `json:"revision"`
	CreatedAt       time.Time     `json:"createdAt"`
	NextActions     []string      `json:"nextActions"`
	ReadOnly        bool          `json:"readOnly"`
}

type ListResult struct {
	Items []TrialSummary `json:"items"`
	Total int            `json:"total"`
}

type EventFilter struct {
	Type, Actor string
	From, To    *time.Time
}

func New(s *store.Store) *Service { return &Service{Store: s} }
func (s *Service) lock(id string) func() {
	v, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

func (s *Service) Create(spec domain.Trial) (*domain.Trial, error) {
	return s.CreateBy(spec, "system")
}

func (s *Service) CreateBy(spec domain.Trial, actor string) (*domain.Trial, error) {
	t, err := domain.NewTrial(spec.TrialID, spec.SpeciesName, spec.AccessionCode, spec.CollectionBatch, spec.ReplicateCount, spec.SeedsPerReplicate)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(actor) != "" && actor != "system" {
		t.Events[0].Actor = strings.TrimSpace(actor)
		t.Events[0].Digest = domain.EventDigest(t.Events[0])
	}
	if err = s.Store.Create(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Service) Get(id string) (*domain.Trial, error) { return s.Store.Get(strings.TrimSpace(id)) }
func (s *Service) List() ([]domain.Trial, error)        { return s.Store.List() }

func (s *Service) Search(f ListFilter) (ListResult, error) {
	trials, err := s.Store.List()
	if err != nil {
		return ListResult{}, err
	}
	items := []TrialSummary{}
	for _, t := range trials {
		if f.Species != "" && !strings.Contains(strings.ToLower(t.SpeciesName), strings.ToLower(strings.TrimSpace(f.Species))) {
			continue
		}
		if f.CollectionBatch != "" && !strings.EqualFold(strings.TrimSpace(f.CollectionBatch), t.CollectionBatch) {
			continue
		}
		if f.Status != "" && f.Status != t.Status {
			continue
		}
		items = append(items, TrialSummary{t.TrialID, t.SpeciesName, t.AccessionCode, t.CollectionBatch, t.Status, t.Revision, t.CreatedAt, t.NextActions(), t.Status == domain.Closed})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].TrialID < items[j].TrialID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return ListResult{Items: items, Total: len(items)}, nil
}

// idempotencyActionKey 将动作与幂等键组合，使同一试验的不同动作即使复用相同的
// Idempotency-Key 也能各自独立缓存与重放。空键保持不启用幂等缓存的原有行为。
func idempotencyActionKey(action, key string) string {
	if key == "" {
		return ""
	}
	return action + "\x00" + key
}

func (s *Service) mutate(id, action, key string, expected *int64, fn func(*domain.Trial) error) (*domain.Trial, error) {
	unlock := s.lock(id)
	defer unlock()
	cacheKey := idempotencyActionKey(action, key)
	if out, ok := s.Store.LookupIdempotent(id, cacheKey); ok {
		return out, nil
	}
	t, err := s.Store.Get(id)
	if err != nil {
		return nil, err
	}
	if !t.Integrity.Valid {
		return nil, fmt.Errorf("数据完整性错误：%s", t.Integrity.Message)
	}
	if t.Status == domain.Closed {
		return nil, errors.New("已关闭批次为只读，拒绝后续写操作")
	}
	if expected != nil && *expected != t.Revision {
		return nil, fmt.Errorf("版本冲突：当前 revision=%d", t.Revision)
	}
	old := t.Revision
	if err = fn(t); err != nil {
		return nil, err
	}
	return s.Store.Save(t, old, cacheKey)
}

func (s *Service) PreviewProtocol(id string, p domain.TreatmentProtocol) (domain.ProtocolPreview, error) {
	t, err := s.Store.Get(id)
	if err != nil {
		return domain.ProtocolPreview{}, err
	}
	if !t.Integrity.Valid {
		return domain.ProtocolPreview{}, fmt.Errorf("数据完整性错误：%s", t.Integrity.Message)
	}
	if t.Status != domain.Draft && t.Status != domain.CorrectionRequired {
		return domain.ProtocolPreview{}, errors.New("当前状态不能编辑方案")
	}
	return domain.PreviewProtocol(t.TrialID, t.Revision, p), nil
}

func (s *Service) LockProtocol(id, key string, p domain.TreatmentProtocol) (*domain.Trial, error) {
	return s.mutate(id, "protocol", key, nil, func(t *domain.Trial) error { return t.LockProtocol(p) })
}
func (s *Service) LockProtocolChecked(id, key string, expected int64, digest, actor string, p domain.TreatmentProtocol) (*domain.Trial, error) {
	return s.mutate(id, "protocol", key, &expected, func(t *domain.Trial) error { return t.LockProtocolPreview(p, digest, expected, actor) })
}
func (s *Service) Start(id, key string) (*domain.Trial, error) {
	return s.mutate(id, "start", key, nil, func(t *domain.Trial) error { return t.StartObserving() })
}
func (s *Service) StartChecked(id, key string, expected int64, actor string) (*domain.Trial, error) {
	return s.mutate(id, "start", key, &expected, func(t *domain.Trial) error { return t.StartObservingBy(actor) })
}
func (s *Service) Observe(id, key string, o domain.DailyObservation) (*domain.Trial, error) {
	return s.mutate(id, "observe", key, nil, func(t *domain.Trial) error { return t.AddObservation(o) })
}
func (s *Service) ObserveBatch(id, key string, expected int64, day int, observations []domain.DailyObservation, actor string) (*domain.Trial, error) {
	return s.mutate(id, "observe", key, &expected, func(t *domain.Trial) error { return t.AddObservationBatch(day, observations, actor) })
}
func (s *Service) Deviation(id, key string, d domain.Deviation) (*domain.Trial, error) {
	return s.mutate(id, "deviations", key, nil, func(t *domain.Trial) error { return t.AddDeviation(d) })
}
func (s *Service) DeviationChecked(id, key string, expected int64, d domain.Deviation, actor string) (*domain.Trial, error) {
	return s.mutate(id, "deviations", key, &expected, func(t *domain.Trial) error { return t.AddDeviationBy(d, actor) })
}
func (s *Service) Resolve(id, key, devID string) (*domain.Trial, error) {
	return s.mutate(id, "resolve", key, nil, func(t *domain.Trial) error { return t.ResolveDeviation(devID) })
}
func (s *Service) ResolveChecked(id, key string, expected int64, devID, responsible, completion string, observationIDs []string, actor string) (*domain.Trial, error) {
	return s.mutate(id, "resolve", key, &expected, func(t *domain.Trial) error {
		return t.ResolveDeviationWithEvidence(devID, responsible, completion, observationIDs, actor)
	})
}
func (s *Service) Submit(id, key string) (*domain.Trial, error) {
	return s.mutate(id, "submit", key, nil, func(t *domain.Trial) error { return t.ReadyForReview() })
}
func (s *Service) SubmitChecked(id, key string, expected int64, actor string) (*domain.Trial, error) {
	return s.mutate(id, "submit", key, &expected, func(t *domain.Trial) error { return t.SubmitForReview(actor) })
}
func (s *Service) Review(id, key, decision, reason, reviewer string) (*domain.Trial, error) {
	return s.mutate(id, "review", key, nil, func(t *domain.Trial) error { return t.Review(decision, reason, reviewer) })
}
func (s *Service) ReviewChecked(id, key string, expected int64, decision, conclusion, reviewer, digest string, issues []domain.ReviewIssueInput) (*domain.Trial, error) {
	return s.mutate(id, "review", key, &expected, func(t *domain.Trial) error {
		return t.ReviewWithChecklist(decision, conclusion, reviewer, digest, issues)
	})
}
func (s *Service) Correct(id, key string, expected int64, issueID, note string, refs []string, confirm bool, actor string) (*domain.Trial, error) {
	return s.mutate(id, "corrections", key, &expected, func(t *domain.Trial) error { return t.SubmitCorrection(issueID, note, refs, confirm, actor) })
}

func (s *Service) Details(id string, filters ...EventFilter) (map[string]any, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	m := t.Metrics()
	if t.Status == domain.ReviewPending && len(t.ReviewPackages) > 0 {
		m = t.ReviewPackages[len(t.ReviewPackages)-1].Snapshot.Metrics
	}
	events := append([]domain.Event(nil), t.Events...)
	if len(filters) > 0 {
		f := filters[0]
		filtered := []domain.Event{}
		for _, e := range events {
			if f.Type != "" && e.Type != f.Type {
				continue
			}
			if f.Actor != "" && !strings.EqualFold(e.Actor, f.Actor) {
				continue
			}
			if f.From != nil && e.At.Before(*f.From) {
				continue
			}
			if f.To != nil && e.At.After(*f.To) {
				continue
			}
			filtered = append(filtered, e)
		}
		events = filtered
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Seq < events[j].Seq })
	readiness := domain.AssessReviewReadiness(t, m)
	return map[string]any{
		"trial": t, "metrics": m, "openDeviationCount": readiness.OpenDeviationCount, "canSubmit": readiness.Eligible,
		"reviewReadiness": readiness,
		"nextActions":     t.NextActions(), "readOnly": t.Status == domain.Closed || !t.Integrity.Valid,
		"eventHistory": map[string]any{"items": events, "total": len(events), "integrity": t.Integrity},
	}, nil
}
