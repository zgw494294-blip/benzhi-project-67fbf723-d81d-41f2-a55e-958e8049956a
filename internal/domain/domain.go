package domain

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type Status string

const (
	Draft              Status = "DRAFT"
	ProtocolLocked     Status = "PROTOCOL_LOCKED"
	Observing          Status = "OBSERVING"
	ReviewPending      Status = "REVIEW_PENDING"
	CorrectionRequired Status = "CORRECTION_REQUIRED"
	Closed             Status = "CLOSED"
)

type Trial struct {
	TrialID           string             `json:"trialId"`
	SpeciesName       string             `json:"speciesName"`
	AccessionCode     string             `json:"accessionCode"`
	CollectionBatch   string             `json:"collectionBatch"`
	ReplicateCount    int                `json:"replicateCount"`
	SeedsPerReplicate int                `json:"seedsPerReplicate"`
	Status            Status             `json:"status"`
	Revision          int64              `json:"revision"`
	CreatedAt         time.Time          `json:"createdAt"`
	ClosedAt          *time.Time         `json:"closedAt,omitempty"`
	Protocol          *TreatmentProtocol `json:"protocol,omitempty"`
	Observations      []DailyObservation `json:"observations"`
	Deviations        []Deviation        `json:"deviations"`
	CorrectionItems   []CorrectionItem   `json:"correctionItems"`
	Events            []Event            `json:"events"`
	ReviewPackages    []ReviewPackage    `json:"reviewPackages"`
	Integrity         IntegrityStatus    `json:"integrity"`
}

type TreatmentProtocol struct {
	ProtocolID                  string     `json:"protocolId"`
	TrialID                     string     `json:"trialId"`
	StratificationDays          int        `json:"stratificationDays"`
	TemperatureCelsius          float64    `json:"temperatureCelsius"`
	LightRegime                 string     `json:"lightRegime"`
	Substrate                   string     `json:"substrate"`
	ObservationDays             int        `json:"observationDays"`
	GerminationThresholdPercent float64    `json:"germinationThresholdPercent"`
	LockedAt                    *time.Time `json:"lockedAt,omitempty"`
	LockedRevision              int64      `json:"lockedRevision"`
	ContentDigest               string     `json:"contentDigest"`
}

type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
type ValidationErrors struct {
	Issues []ValidationIssue `json:"issues"`
}

func (e *ValidationErrors) Error() string { return "字段校验未通过" }

type ProtocolPreview struct {
	Protocol      TreatmentProtocol `json:"protocol"`
	Summary       string            `json:"summary"`
	ContentDigest string            `json:"contentDigest"`
	Issues        []ValidationIssue `json:"issues"`
	Revision      int64             `json:"revision"`
}

type DailyObservation struct {
	ObservationID      string    `json:"observationId"`
	TrialID            string    `json:"trialId"`
	DayIndex           int       `json:"dayIndex"`
	ReplicateIndex     int       `json:"replicateIndex"`
	NewlyGerminated    int       `json:"newlyGerminated"`
	NewlyNonviable     int       `json:"newlyNonviable"`
	TemperatureCelsius float64   `json:"temperatureCelsius"`
	Note               string    `json:"note"`
	RecordedBy         string    `json:"recordedBy"`
	RecordedAt         time.Time `json:"recordedAt"`
	RecordedRevision   int64     `json:"recordedRevision"`
}

type ObservationBatchError struct {
	Message           string            `json:"message"`
	MissingReplicates []int             `json:"missingReplicates,omitempty"`
	RemainingByGroup  map[int]int       `json:"remainingByGroup,omitempty"`
	Issues            []ValidationIssue `json:"issues,omitempty"`
}

func (e *ObservationBatchError) Error() string { return e.Message }

type Deviation struct {
	ID                    string     `json:"id"`
	TrialID               string     `json:"trialId"`
	Kind                  string     `json:"kind"`
	Description           string     `json:"description"`
	CorrectiveAction      string     `json:"correctiveAction"`
	WindowStart           int        `json:"windowStart"`
	WindowEnd             int        `json:"windowEnd"`
	Resolved              bool       `json:"resolved"`
	ResponsiblePerson     string     `json:"responsiblePerson,omitempty"`
	CompletionDescription string     `json:"completionDescription,omitempty"`
	ObservationIDs        []string   `json:"observationIds,omitempty"`
	CreatedAt             *time.Time `json:"createdAt,omitempty"`
	ResolvedAt            *time.Time `json:"resolvedAt,omitempty"`
	ResolvedRevision      int64      `json:"resolvedRevision,omitempty"`
}

type CorrectionItem struct {
	ID                string   `json:"id"`
	Category          string   `json:"category"`
	Description       string   `json:"description"`
	RequiredAction    string   `json:"requiredAction"`
	ObjectType        string   `json:"objectType"`
	ObjectID          string   `json:"objectId"`
	Status            string   `json:"status"`
	CorrectionNote    string   `json:"correctionNote,omitempty"`
	ReferenceIDs      []string `json:"referenceIds,omitempty"`
	ReturnRevision    int64    `json:"returnRevision"`
	SubmittedRevision int64    `json:"submittedRevision,omitempty"`
}

type ReviewSnapshot struct {
	TrialID         string             `json:"trialId"`
	Status          Status             `json:"status"`
	StateRevision   int64              `json:"stateRevision"`
	Protocol        *TreatmentProtocol `json:"protocol"`
	Observations    []DailyObservation `json:"observations"`
	Metrics         Metrics            `json:"metrics"`
	Deviations      []Deviation        `json:"deviations"`
	CorrectionItems []CorrectionItem   `json:"correctionItems"`
	ProtocolSummary string             `json:"protocolSummary"`
}
type SnapshotDifference struct {
	AddedObservations   []string `json:"addedObservations"`
	ChangedObservations []string `json:"changedObservations"`
	DeviationChanges    []string `json:"deviationChanges"`
	MetricChanges       []string `json:"metricChanges"`
	ProtocolChanged     bool     `json:"protocolChanged"`
}
type ReviewPackage struct {
	ReviewID           string             `json:"reviewId"`
	TrialID            string             `json:"trialId"`
	SubmissionNumber   int                `json:"submissionNumber"`
	SnapshotDigest     string             `json:"snapshotDigest"`
	Snapshot           ReviewSnapshot     `json:"snapshot"`
	Difference         SnapshotDifference `json:"difference"`
	MetricSummary      string             `json:"metricSummary"`
	OpenDeviationCount int                `json:"openDeviationCount"`
	Decision           string             `json:"decision,omitempty"`
	DecisionReason     string             `json:"decisionReason,omitempty"`
	FinalConclusion    string             `json:"finalConclusion,omitempty"`
	ReviewedBy         string             `json:"reviewedBy,omitempty"`
	ReviewedAt         *time.Time         `json:"reviewedAt,omitempty"`
	CreatedAt          time.Time          `json:"createdAt"`
}

type Event struct {
	Seq            int64     `json:"seq"`
	Type           string    `json:"type"`
	Summary        string    `json:"summary"`
	Actor          string    `json:"actor"`
	At             time.Time `json:"at"`
	RevisionBefore int64     `json:"revisionBefore"`
	RevisionAfter  int64     `json:"revisionAfter"`
	ObjectType     string    `json:"objectType"`
	ObjectID       string    `json:"objectId"`
	PreviousDigest string    `json:"previousDigest"`
	Digest         string    `json:"digest"`
}
type IntegrityStatus struct {
	Valid                bool   `json:"valid"`
	FirstInvalidSequence int64  `json:"firstInvalidSequence,omitempty"`
	LastTrustedRevision  int64  `json:"lastTrustedRevision"`
	Message              string `json:"message,omitempty"`
}

type Metrics struct {
	GerminationRatePercent float64             `json:"germinationRatePercent"`
	MeanGerminationTime    float64             `json:"meanGerminationTime"`
	ReplicateStdDev        float64             `json:"replicateStdDev"`
	CumulativeCurve        []CurvePoint        `json:"cumulativeCurve"`
	MissingDays            []int               `json:"missingDays"`
	Gaps                   []ObservationGap    `json:"gaps"`
	ReplicateDetails       []ReplicateMetric   `json:"replicateDetails"`
	ThresholdPercent       float64             `json:"thresholdPercent"`
	ThresholdMet           bool                `json:"thresholdMet"`
	ThresholdFirstDay      int                 `json:"thresholdFirstDay,omitempty"`
	ThresholdConclusion    string              `json:"thresholdConclusion"`
	FormulaInputs          MetricFormulaInputs `json:"formulaInputs"`
}
type MetricFormulaInputs struct {
	TotalSeeds                     int     `json:"totalSeeds"`
	ValidGerminated                int     `json:"validGerminated"`
	DayWeightedGerminated          int     `json:"dayWeightedGerminated"`
	MeanGerminationTimeNumerator   int     `json:"meanGerminationTimeNumerator"`
	MeanGerminationTimeDenominator int     `json:"meanGerminationTimeDenominator"`
	ReplicateMeanRatePercent       float64 `json:"replicateMeanRatePercent"`
}
type ReplicateMetric struct {
	ReplicateIndex         int     `json:"replicateIndex"`
	Germinated             int     `json:"germinated"`
	RatePercent            float64 `json:"ratePercent"`
	DifferenceFromMean     float64 `json:"differenceFromMean"`
	DispersionContribution float64 `json:"dispersionContribution"`
}
type ObservationGap struct {
	DayIndex       int    `json:"dayIndex"`
	ReplicateIndex int    `json:"replicateIndex,omitempty"`
	Kind           string `json:"kind"`
}
type CurvePoint struct {
	Day         int     `json:"day"`
	Germinated  int     `json:"germinated"`
	RatePercent float64 `json:"ratePercent"`
}

func NewTrial(id, species, accession, batch string, reps, seeds int) (*Trial, error) {
	id, species, accession, batch = NormalizeIdentity(id, species, accession, batch)
	if id == "" {
		return nil, errors.New("试验编号不能为空")
	}
	if species == "" || reps < 1 || seeds < 1 {
		return nil, errors.New("物种和重复组规模必须有效")
	}
	now := time.Now().UTC()
	t := &Trial{TrialID: id, SpeciesName: species, AccessionCode: accession, CollectionBatch: batch, ReplicateCount: reps, SeedsPerReplicate: seeds, Status: Draft, Revision: 1, CreatedAt: now, Observations: []DailyObservation{}, Deviations: []Deviation{}, CorrectionItems: []CorrectionItem{}, Events: []Event{}, ReviewPackages: []ReviewPackage{}, Integrity: IntegrityStatus{Valid: true, LastTrustedRevision: 1}}
	t.addEventAt("TRIAL_CREATED", "试验批次已建立", "system", "trial", id, 0, 1, now)
	return t, nil
}

func PreviewProtocol(trialID string, revision int64, p TreatmentProtocol) ProtocolPreview {
	p = NormalizeProtocol(p)
	p.TrialID = trialID
	if p.ProtocolID == "" {
		p.ProtocolID = trialID + "-protocol"
	}
	issues := []ValidationIssue{}
	if p.StratificationDays < 0 {
		issues = append(issues, ValidationIssue{"stratificationDays", "层积天数不能为负数"})
	}
	if p.TemperatureCelsius < -20 || p.TemperatureCelsius > 60 {
		issues = append(issues, ValidationIssue{"temperatureCelsius", "温度必须在 -20 至 60 摄氏度之间"})
	}
	if p.LightRegime == "" {
		issues = append(issues, ValidationIssue{"lightRegime", "光照方案不能为空"})
	}
	if p.Substrate == "" {
		issues = append(issues, ValidationIssue{"substrate", "培养基不能为空"})
	}
	if p.ObservationDays < 1 {
		issues = append(issues, ValidationIssue{"observationDays", "观察周期至少为 1 日"})
	}
	if p.GerminationThresholdPercent <= 0 || p.GerminationThresholdPercent > 100 {
		issues = append(issues, ValidationIssue{"germinationThresholdPercent", "判定阈值必须大于 0 且不超过 100%"})
	}
	digest := ""
	if len(issues) == 0 {
		digest = Digest(p)
	}
	return ProtocolPreview{p, fmt.Sprintf("层积%d日；温度%.2f℃；光照%s；培养基%s；观察%d日；阈值%.2f%%", p.StratificationDays, p.TemperatureCelsius, p.LightRegime, p.Substrate, p.ObservationDays, p.GerminationThresholdPercent), digest, issues, revision}
}
func (t *Trial) LockProtocol(p TreatmentProtocol) error {
	v := PreviewProtocol(t.TrialID, t.Revision, p)
	return t.LockProtocolPreview(v.Protocol, v.ContentDigest, t.Revision, "system")
}
func (t *Trial) LockProtocolPreview(p TreatmentProtocol, digest string, expected int64, actor string) error {
	if t.Status != Draft && t.Status != CorrectionRequired {
		return errors.New("当前状态不能锁定方案")
	}
	if expected != t.Revision {
		return fmt.Errorf("版本冲突：当前 revision=%d，请重新预检", t.Revision)
	}
	v := PreviewProtocol(t.TrialID, t.Revision, p)
	if len(v.Issues) > 0 {
		return &ValidationErrors{v.Issues}
	}
	if digest == "" || digest != v.ContentDigest {
		return errors.New("方案摘要不匹配，请重新预检")
	}
	now := time.Now().UTC()
	p = v.Protocol
	p.LockedAt = &now
	p.LockedRevision = t.Revision + 1
	p.ContentDigest = v.ContentDigest
	t.Protocol = &p
	t.Status = ProtocolLocked
	t.bumpEvent("PROTOCOL_LOCKED", "处理方案已锁定", actor, "protocol", p.ProtocolID)
	return nil
}
func (t *Trial) StartObserving() error { return t.StartObservingBy("system") }
func (t *Trial) StartObservingBy(actor string) error {
	if t.Status != ProtocolLocked {
		return errors.New("方案未锁定")
	}
	t.Status = Observing
	t.bumpEvent("OBSERVING", "试验开始观察", actor, "trial", t.TrialID)
	return nil
}

// AddObservation 保留领域层兼容；HTTP 主流程使用整日原子批量录入。
func (t *Trial) AddObservation(o DailyObservation) error {
	if t.Status != Observing && t.Status != CorrectionRequired {
		return errors.New("当前状态不能录入观察")
	}
	if err := t.validateObservation(o); err != nil {
		return err
	}
	for _, x := range t.Observations {
		if x.DayIndex == o.DayIndex && x.ReplicateIndex == o.ReplicateIndex {
			return errors.New("重复提交观察")
		}
	}
	used := 0
	for _, x := range t.Observations {
		if x.ReplicateIndex == o.ReplicateIndex {
			used += x.NewlyGerminated + x.NewlyNonviable
		}
	}
	if used+o.NewlyGerminated+o.NewlyNonviable > t.SeedsPerReplicate {
		return errors.New("累计计数超过种子数")
	}
	o.TrialID = t.TrialID
	if o.ObservationID == "" {
		o.ObservationID = fmt.Sprintf("obs-%d", len(t.Observations)+1)
	}
	if o.RecordedAt.IsZero() {
		o.RecordedAt = time.Now().UTC()
	}
	o.RecordedRevision = t.Revision + 1
	t.Observations = append(t.Observations, o)
	t.sortObservations()
	t.bumpEvent("OBSERVATION_ADDED", fmt.Sprintf("第%d日重复组%d观察", o.DayIndex, o.ReplicateIndex), o.RecordedBy, "observation", o.ObservationID)
	return nil
}
func (t *Trial) validateObservation(o DailyObservation) error {
	if t.Protocol != nil && o.DayIndex > t.Protocol.ObservationDays {
		return errors.New("超出观察周期")
	}
	if o.DayIndex < 1 || o.ReplicateIndex < 1 || o.ReplicateIndex > t.ReplicateCount || o.NewlyGerminated < 0 || o.NewlyNonviable < 0 {
		return errors.New("观察日期、重复组或计数不合法")
	}
	return nil
}
func (t *Trial) AddObservationBatch(day int, observations []DailyObservation, actor string) error {
	if t.Status != Observing && t.Status != CorrectionRequired {
		return errors.New("当前状态不能录入观察")
	}
	if t.Protocol == nil || day < 1 || day > t.Protocol.ObservationDays {
		return errors.New("观察日期超出方案周期")
	}
	byRep := map[int]DailyObservation{}
	issues := []ValidationIssue{}
	for _, o := range observations {
		o.DayIndex = day
		if err := t.validateObservation(o); err != nil {
			issues = append(issues, ValidationIssue{fmt.Sprintf("observations.%d", o.ReplicateIndex), err.Error()})
			continue
		}
		if _, ok := byRep[o.ReplicateIndex]; ok {
			issues = append(issues, ValidationIssue{fmt.Sprintf("observations.%d", o.ReplicateIndex), "重复组编号重复"})
			continue
		}
		byRep[o.ReplicateIndex] = o
	}
	missing := []int{}
	for r := 1; r <= t.ReplicateCount; r++ {
		if _, ok := byRep[r]; !ok {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		return &ObservationBatchError{Message: fmt.Sprintf("当日重复组不完整，缺少组 %v", missing), MissingReplicates: missing, Issues: issues}
	}
	if len(issues) > 0 {
		return &ObservationBatchError{Message: "观察数据校验未通过", Issues: issues}
	}
	for _, x := range t.Observations {
		if x.DayIndex == day {
			return errors.New("该试验日已有观察，不能重复提交")
		}
	}
	next := 1
	for d := 1; d <= t.Protocol.ObservationDays; d++ {
		count := 0
		for _, x := range t.Observations {
			if x.DayIndex == d {
				count++
			}
		}
		if count == t.ReplicateCount {
			next = d + 1
			continue
		}
		if count > 0 && d < day {
			miss := []int{}
			for r := 1; r <= t.ReplicateCount; r++ {
				if !t.hasObservation(d, r) {
					miss = append(miss, r)
				}
			}
			return &ObservationBatchError{Message: fmt.Sprintf("前一日数据不完整：第%d日缺少组 %v", d, miss), MissingReplicates: miss}
		}
		break
	}
	if day != next {
		return fmt.Errorf("待录日期必须为第%d日", next)
	}
	remaining := map[int]int{}
	for r, o := range byRep {
		used := 0
		for _, old := range t.Observations {
			if old.ReplicateIndex == r {
				used += old.NewlyGerminated + old.NewlyNonviable
			}
		}
		left := t.SeedsPerReplicate - used
		if o.NewlyGerminated+o.NewlyNonviable > left {
			remaining[r] = left
		}
	}
	if len(remaining) > 0 {
		return &ObservationBatchError{Message: "累计计数超过种子数", RemainingByGroup: remaining}
	}
	now := time.Now().UTC()
	refs := []string{}
	for r := 1; r <= t.ReplicateCount; r++ {
		o := byRep[r]
		o.DayIndex, o.TrialID = day, t.TrialID
		if o.RecordedBy == "" {
			o.RecordedBy = actor
		}
		o.RecordedAt = now
		o.RecordedRevision = t.Revision + 1
		o.ObservationID = fmt.Sprintf("obs-%d", len(t.Observations)+1)
		t.Observations = append(t.Observations, o)
		refs = append(refs, o.ObservationID)
	}
	t.sortObservations()
	t.bumpEvent("OBSERVATIONS_RECORDED", fmt.Sprintf("第%d日%d个重复组观察：%s", day, len(observations), strings.Join(refs, ",")), actor, "observation-day", fmt.Sprintf("day-%d", day))
	return nil
}
func (t *Trial) hasObservation(day, rep int) bool {
	for _, o := range t.Observations {
		if o.DayIndex == day && o.ReplicateIndex == rep {
			return true
		}
	}
	return false
}
func (t *Trial) sortObservations() {
	sort.Slice(t.Observations, func(i, j int) bool {
		if t.Observations[i].DayIndex == t.Observations[j].DayIndex {
			return t.Observations[i].ReplicateIndex < t.Observations[j].ReplicateIndex
		}
		return t.Observations[i].DayIndex < t.Observations[j].DayIndex
	})
}

func (t *Trial) AddDeviation(d Deviation) error { return t.AddDeviationBy(d, "system") }
func (t *Trial) AddDeviationBy(d Deviation, actor string) error {
	if t.Status != Observing && t.Status != CorrectionRequired {
		return errors.New("当前状态不能登记偏差")
	}
	issues := []ValidationIssue{}
	d.Kind, d.Description, d.CorrectiveAction = strings.TrimSpace(d.Kind), strings.TrimSpace(d.Description), strings.TrimSpace(d.CorrectiveAction)
	if d.Kind == "" {
		issues = append(issues, ValidationIssue{"kind", "偏差类型不能为空"})
	}
	if d.Description == "" {
		issues = append(issues, ValidationIssue{"description", "偏差说明不能为空"})
	}
	if d.CorrectiveAction == "" {
		issues = append(issues, ValidationIssue{"correctiveAction", "纠正措施不能为空"})
	}
	if t.Protocol == nil || d.WindowStart < 1 || d.WindowEnd < d.WindowStart || d.WindowEnd > t.Protocol.ObservationDays {
		issues = append(issues, ValidationIssue{"window", "补录窗口必须按起止顺序落在观察周期内"})
	}
	if len(issues) > 0 {
		return &ValidationErrors{issues}
	}
	d.TrialID, d.ID = t.TrialID, fmt.Sprintf("dev-%d", len(t.Deviations)+1)
	now := time.Now().UTC()
	d.CreatedAt = &now
	t.Deviations = append(t.Deviations, d)
	t.bumpEvent("DEVIATION_ADDED", d.Kind, actor, "deviation", d.ID)
	return nil
}
func (t *Trial) ResolveDeviation(id string) error {
	return t.ResolveDeviationWithEvidence(id, "system", "兼容销项", nil, "system")
}
func (t *Trial) ResolveDeviationWithEvidence(id, responsible, completion string, observationIDs []string, actor string) error {
	if t.Status == Closed {
		return errors.New("已关闭批次为只读")
	}
	for i := range t.Deviations {
		d := &t.Deviations[i]
		if d.ID != id {
			continue
		}
		if d.Resolved {
			return nil
		}
		missing := []string{}
		if strings.TrimSpace(responsible) == "" {
			missing = append(missing, "responsiblePerson")
		}
		if strings.TrimSpace(completion) == "" {
			missing = append(missing, "completionDescription")
		}
		if len(observationIDs) == 0 {
			missing = append(missing, "observationIds")
		}
		for _, oid := range observationIDs {
			found := false
			for _, o := range t.Observations {
				if o.ObservationID == oid && o.TrialID == t.TrialID && o.DayIndex >= d.WindowStart && o.DayIndex <= d.WindowEnd {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, "窗口内观察:"+oid)
			}
		}
		for day := d.WindowStart; day <= d.WindowEnd; day++ {
			for rep := 1; rep <= t.ReplicateCount; rep++ {
				if !t.hasObservation(day, rep) {
					missing = append(missing, fmt.Sprintf("第%d日组%d观察", day, rep))
				}
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("缺少纠正证据：%s", strings.Join(missing, "、"))
		}
		now := time.Now().UTC()
		d.Resolved, d.ResolvedAt = true, &now
		d.ResolvedRevision = t.Revision + 1
		d.ResponsiblePerson, d.CompletionDescription = strings.TrimSpace(responsible), strings.TrimSpace(completion)
		d.ObservationIDs = append([]string(nil), observationIDs...)
		t.bumpEvent("DEVIATION_RESOLVED", id, actor, "deviation", id)
		return nil
	}
	return errors.New("偏差不存在")
}

func (t *Trial) Metrics() Metrics {
	m := Metrics{CumulativeCurve: []CurvePoint{}, MissingDays: []int{}, Gaps: []ObservationGap{}, ReplicateDetails: []ReplicateMetric{}}
	if t.Protocol != nil {
		m.ThresholdPercent = t.Protocol.GerminationThresholdPercent
	}
	total := t.ReplicateCount * t.SeedsPerReplicate
	germ, weighted := 0, 0
	byRep := make([]int, t.ReplicateCount)
	byDay := map[int]int{}
	maxObserved := 0
	for _, o := range t.Observations {
		germ += o.NewlyGerminated
		weighted += o.DayIndex * o.NewlyGerminated
		if o.ReplicateIndex >= 1 && o.ReplicateIndex <= len(byRep) {
			byRep[o.ReplicateIndex-1] += o.NewlyGerminated
		}
		byDay[o.DayIndex] += o.NewlyGerminated
		if o.DayIndex > maxObserved {
			maxObserved = o.DayIndex
		}
	}
	if total > 0 {
		m.GerminationRatePercent = round(float64(germ) * 100 / float64(total))
	}
	if germ > 0 {
		m.MeanGerminationTime = round(float64(weighted) / float64(germ))
	}
	mean := 0.0
	if len(byRep) > 0 {
		for _, n := range byRep {
			mean += float64(n) * 100 / float64(t.SeedsPerReplicate)
		}
		mean /= float64(len(byRep))
	}
	variance := 0.0
	for i, n := range byRep {
		rate := float64(n) * 100 / float64(t.SeedsPerReplicate)
		diff := rate - mean
		contribution := diff * diff / float64(len(byRep))
		variance += contribution
		m.ReplicateDetails = append(m.ReplicateDetails, ReplicateMetric{i + 1, n, round(rate), round(diff), round(contribution)})
	}
	m.ReplicateStdDev = round(math.Sqrt(variance))
	m.FormulaInputs = MetricFormulaInputs{total, germ, weighted, weighted, germ, round(mean)}
	period := maxObserved
	if t.Protocol != nil {
		period = t.Protocol.ObservationDays
	}
	cumulative := 0
	for d := 1; d <= period; d++ {
		cumulative += byDay[d]
		rate := 0.0
		if total > 0 {
			rate = round(float64(cumulative) * 100 / float64(total))
		}
		m.CumulativeCurve = append(m.CumulativeCurve, CurvePoint{d, cumulative, rate})
		if m.ThresholdFirstDay == 0 && rate >= m.ThresholdPercent {
			m.ThresholdFirstDay = d
		}
		missing := []int{}
		for r := 1; r <= t.ReplicateCount; r++ {
			if !t.hasObservation(d, r) {
				missing = append(missing, r)
			}
		}
		if len(missing) == 0 {
			continue
		}
		future := d > maxObserved
		if len(missing) == t.ReplicateCount {
			kind := "MISSING_DAY"
			if future {
				kind = "PENDING_DAY"
			}
			m.Gaps = append(m.Gaps, ObservationGap{DayIndex: d, Kind: kind})
		} else {
			kind := "MISSING"
			if future {
				kind = "PENDING"
			}
			for _, r := range missing {
				m.Gaps = append(m.Gaps, ObservationGap{d, r, kind})
			}
		}
		if !future {
			m.MissingDays = append(m.MissingDays, d)
		}
	}
	m.ThresholdMet = m.GerminationRatePercent >= m.ThresholdPercent
	if m.ThresholdFirstDay > 0 {
		m.ThresholdConclusion = fmt.Sprintf("第%d日首次达到阈值", m.ThresholdFirstDay)
	} else {
		m.ThresholdConclusion = "观察期内尚未达到阈值"
	}
	return m
}

func (t *Trial) ReadyForReview() error { return t.SubmitForReview("system") }
func (t *Trial) SubmitForReview(actor string) error {
	if t.Status != Observing && t.Status != CorrectionRequired {
		return errors.New("当前状态不能送审")
	}
	if t.Protocol == nil {
		return errors.New("缺少方案")
	}
	m := t.Metrics()
	if len(m.Gaps) > 0 {
		return errors.New("观察周期不完整")
	}
	for _, d := range t.Deviations {
		if !d.Resolved {
			return fmt.Errorf("存在未解决偏差：%s", d.ID)
		}
	}
	pending := []string{}
	for _, c := range t.CorrectionItems {
		if c.Status != "CONFIRMED" {
			pending = append(pending, c.ID)
		}
	}
	if len(pending) > 0 {
		return fmt.Errorf("退回问题尚未全部确认：%s", strings.Join(pending, ","))
	}
	if !m.ThresholdMet {
		return errors.New("未达到发芽率阈值")
	}
	now := time.Now().UTC()
	snapshot := t.buildSnapshot(t.Revision + 1)
	p := ReviewPackage{ReviewID: fmt.Sprintf("review-%d", len(t.ReviewPackages)+1), TrialID: t.TrialID, SubmissionNumber: len(t.ReviewPackages) + 1, SnapshotDigest: Digest(snapshot), Snapshot: snapshot, MetricSummary: fmt.Sprintf("发芽率 %.2f%%，平均发芽时间 %.2f日", m.GerminationRatePercent, m.MeanGerminationTime), OpenDeviationCount: openDeviations(t), CreatedAt: now}
	if len(t.ReviewPackages) > 0 {
		p.Difference = compareSnapshots(t.ReviewPackages[len(t.ReviewPackages)-1].Snapshot, snapshot)
	}
	t.ReviewPackages = append(t.ReviewPackages, p)
	t.Status = ReviewPending
	t.bumpEvent("REVIEW_SUBMITTED", fmt.Sprintf("审定包 #%d 已固化", p.SubmissionNumber), actor, "review-package", p.ReviewID)
	return nil
}
func (t *Trial) buildSnapshot(revision int64) ReviewSnapshot {
	return ReviewSnapshot{t.TrialID, ReviewPending, revision, t.Protocol, append([]DailyObservation(nil), t.Observations...), t.Metrics(), append([]Deviation(nil), t.Deviations...), append([]CorrectionItem(nil), t.CorrectionItems...), protocolSummary(t.Protocol)}
}
func protocolSummary(p *TreatmentProtocol) string {
	if p == nil {
		return ""
	}
	return PreviewProtocol(p.TrialID, 0, *p).Summary
}
func compareSnapshots(a, b ReviewSnapshot) SnapshotDifference {
	d := SnapshotDifference{AddedObservations: []string{}, ChangedObservations: []string{}, DeviationChanges: []string{}, MetricChanges: []string{}}
	am := map[string]DailyObservation{}
	for _, o := range a.Observations {
		am[o.ObservationID] = o
	}
	for _, o := range b.Observations {
		if old, ok := am[o.ObservationID]; !ok {
			d.AddedObservations = append(d.AddedObservations, o.ObservationID)
		} else if Digest(old) != Digest(o) {
			d.ChangedObservations = append(d.ChangedObservations, o.ObservationID)
		}
	}
	adm := map[string]Deviation{}
	for _, x := range a.Deviations {
		adm[x.ID] = x
	}
	for _, x := range b.Deviations {
		if old, ok := adm[x.ID]; !ok || Digest(old) != Digest(x) {
			d.DeviationChanges = append(d.DeviationChanges, x.ID)
		}
	}
	if a.Metrics.GerminationRatePercent != b.Metrics.GerminationRatePercent {
		d.MetricChanges = append(d.MetricChanges, "germinationRatePercent")
	}
	if a.Metrics.MeanGerminationTime != b.Metrics.MeanGerminationTime {
		d.MetricChanges = append(d.MetricChanges, "meanGerminationTime")
	}
	if a.Metrics.ReplicateStdDev != b.Metrics.ReplicateStdDev {
		d.MetricChanges = append(d.MetricChanges, "replicateStdDev")
	}
	d.ProtocolChanged = protocolSummary(a.Protocol) != protocolSummary(b.Protocol)
	return d
}

type ReviewIssueInput struct{ Category, Description, RequiredAction, ObjectType, ObjectID string }

func (t *Trial) Review(decision, reason, reviewer string) error {
	digest := ""
	if len(t.ReviewPackages) > 0 {
		digest = t.ReviewPackages[len(t.ReviewPackages)-1].SnapshotDigest
	}
	issues := []ReviewIssueInput{}
	if decision == "RETURN" && strings.TrimSpace(reason) != "" {
		issues = append(issues, ReviewIssueInput{"GENERAL", reason, reason, "trial", t.TrialID})
	}
	return t.ReviewWithChecklist(decision, reason, reviewer, digest, issues)
}
func (t *Trial) ReviewWithChecklist(decision, conclusion, reviewer, snapshotDigest string, issues []ReviewIssueInput) error {
	if t.Status != ReviewPending {
		return errors.New("当前状态不在待审定")
	}
	if strings.TrimSpace(reviewer) == "" {
		return errors.New("审定员不能为空")
	}
	if len(t.ReviewPackages) == 0 {
		return errors.New("缺少审定包")
	}
	p := &t.ReviewPackages[len(t.ReviewPackages)-1]
	if p.SnapshotDigest != snapshotDigest || Digest(p.Snapshot) != p.SnapshotDigest {
		return errors.New("审定包摘要不匹配")
	}
	now := time.Now().UTC()
	switch decision {
	case "RETURN":
		if len(issues) == 0 {
			return errors.New("退回决定至少包含一项结构化问题")
		}
		items := make([]CorrectionItem, 0, len(issues))
		for i, x := range issues {
			if strings.TrimSpace(x.Category) == "" || strings.TrimSpace(x.Description) == "" || strings.TrimSpace(x.RequiredAction) == "" || !t.validObjectReference(x.ObjectType, x.ObjectID) {
				return fmt.Errorf("退回问题 #%d 信息或关联对象无效", i+1)
			}
			items = append(items, CorrectionItem{ID: fmt.Sprintf("issue-%d-%d", p.SubmissionNumber, i+1), Category: x.Category, Description: x.Description, RequiredAction: x.RequiredAction, ObjectType: x.ObjectType, ObjectID: x.ObjectID, Status: "PENDING", ReturnRevision: t.Revision + 1})
		}
		t.CorrectionItems = items
		t.Status = CorrectionRequired
		p.Decision = "RETURN"
		p.DecisionReason = conclusion
	case "APPROVE":
		if strings.TrimSpace(conclusion) == "" {
			return errors.New("审定通过必须填写最终结论")
		}
		t.Status = Closed
		t.ClosedAt = &now
		p.Decision = "APPROVE"
		p.FinalConclusion = strings.TrimSpace(conclusion)
	default:
		return errors.New("未知审定决定")
	}
	p.ReviewedBy, p.ReviewedAt = reviewer, &now
	t.bumpEvent("REVIEW_"+decision, conclusion, reviewer, "review-package", p.ReviewID)
	return nil
}
func (t *Trial) validObjectReference(kind, id string) bool {
	switch kind {
	case "trial":
		return id == t.TrialID
	case "protocol":
		return t.Protocol != nil && id == t.Protocol.ProtocolID
	case "observation":
		for _, x := range t.Observations {
			if x.ObservationID == id {
				return true
			}
		}
	case "deviation":
		for _, x := range t.Deviations {
			if x.ID == id {
				return true
			}
		}
	case "metrics":
		return id == "metrics"
	}
	return false
}
func (t *Trial) SubmitCorrection(issueID, note string, refs []string, confirm bool, actor string) error {
	if t.Status != CorrectionRequired && t.Status != ProtocolLocked && t.Status != Observing {
		return errors.New("当前状态不能处理退回问题")
	}
	for i := range t.CorrectionItems {
		c := &t.CorrectionItems[i]
		if c.ID != issueID {
			continue
		}
		if strings.TrimSpace(note) == "" || len(refs) == 0 {
			return errors.New("纠正说明和关联业务记录不能为空")
		}
		for _, ref := range refs {
			if !t.referenceAfterReturn(ref, c.ReturnRevision) {
				return fmt.Errorf("关联记录 %s 不属于当前试验或未产生于本次退回之后", ref)
			}
		}
		c.CorrectionNote = strings.TrimSpace(note)
		c.ReferenceIDs = append([]string(nil), refs...)
		c.Status = "SUBMITTED"
		if confirm {
			c.Status = "CONFIRMED"
		}
		c.SubmittedRevision = t.Revision + 1
		t.bumpEvent("CORRECTION_"+c.Status, issueID, actor, "correction-item", issueID)
		return nil
	}
	return errors.New("退回问题不存在")
}
func (t *Trial) referenceAfterReturn(ref string, revision int64) bool {
	for _, o := range t.Observations {
		if o.ObservationID == ref && o.RecordedRevision > revision {
			return true
		}
	}
	for _, d := range t.Deviations {
		if d.ID == ref && d.Resolved && d.ResolvedRevision > revision {
			return true
		}
	}
	if ref == "protocol" && t.Protocol != nil && t.Protocol.LockedRevision > revision {
		return true
	}
	return ref == "metrics" && t.Revision >= revision
}
func (t *Trial) bumpEvent(typ, summary, actor, objectType, objectID string) {
	before := t.Revision
	t.Revision++
	t.addEventAt(typ, summary, actor, objectType, objectID, before, t.Revision, time.Now().UTC())
}
func (t *Trial) addEventAt(typ, summary, actor, objectType, objectID string, before, after int64, at time.Time) {
	prev := ""
	if len(t.Events) > 0 {
		prev = t.Events[len(t.Events)-1].Digest
	}
	e := Event{Seq: int64(len(t.Events) + 1), Type: typ, Summary: summary, Actor: strings.TrimSpace(actor), At: at, RevisionBefore: before, RevisionAfter: after, ObjectType: objectType, ObjectID: objectID, PreviousDigest: prev}
	if e.Actor == "" {
		e.Actor = "system"
	}
	e.Digest = EventDigest(e)
	t.Events = append(t.Events, e)
	t.Integrity = IntegrityStatus{Valid: true, LastTrustedRevision: after}
}
func VerifyEvents(events []Event) IntegrityStatus {
	lastRev := int64(0)
	prev := ""
	for i, e := range events {
		seq := int64(i + 1)
		if e.Seq != seq || e.PreviousDigest != prev || e.RevisionBefore != lastRev || e.RevisionAfter != e.RevisionBefore+1 || EventDigest(e) != e.Digest {
			return IntegrityStatus{Valid: false, FirstInvalidSequence: seq, LastTrustedRevision: lastRev, Message: "事件序号、revision 推进或摘要链异常"}
		}
		prev = e.Digest
		lastRev = e.RevisionAfter
	}
	return IntegrityStatus{Valid: true, LastTrustedRevision: lastRev}
}
func (t *Trial) NextActions() []string {
	if !t.Integrity.Valid {
		return []string{"VIEW"}
	}
	switch t.Status {
	case Draft:
		return []string{"PREVIEW_PROTOCOL", "LOCK_PROTOCOL"}
	case ProtocolLocked:
		return []string{"START"}
	case Observing:
		return []string{"OBSERVE", "ADD_DEVIATION", "SUBMIT"}
	case CorrectionRequired:
		return []string{"OBSERVE", "ADD_DEVIATION", "RESOLVE_DEVIATION", "CORRECT", "SUBMIT"}
	case ReviewPending:
		return []string{"REVIEW"}
	default:
		return []string{"VIEW"}
	}
}
