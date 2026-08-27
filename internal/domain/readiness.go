package domain

import (
	"fmt"
	"sort"
	"strings"
)

type ReviewBlockerCode string

const (
	BlockerStatus             ReviewBlockerCode = "STATUS_NOT_SUBMITTABLE"
	BlockerProtocol           ReviewBlockerCode = "PROTOCOL_MISSING"
	BlockerObservationDay     ReviewBlockerCode = "OBSERVATION_DAY_MISSING"
	BlockerObservationGroup   ReviewBlockerCode = "OBSERVATION_GROUP_MISSING"
	BlockerObservationPending ReviewBlockerCode = "OBSERVATION_PENDING"
	BlockerDeviation          ReviewBlockerCode = "DEVIATION_OPEN"
	BlockerCorrection         ReviewBlockerCode = "CORRECTION_UNCONFIRMED"
	BlockerThreshold          ReviewBlockerCode = "THRESHOLD_NOT_MET"
)

type ReviewBlocker struct {
	Code           ReviewBlockerCode `json:"code"`
	Message        string            `json:"message"`
	DayIndex       int               `json:"dayIndex,omitempty"`
	ReplicateIndex int               `json:"replicateIndex,omitempty"`
	ObjectType     string            `json:"objectType,omitempty"`
	ObjectID       string            `json:"objectId,omitempty"`
}

type ObservationProgress struct {
	ObservationDays    int              `json:"observationDays"`
	ReplicateCount     int              `json:"replicateCount"`
	ExpectedCellCount  int              `json:"expectedCellCount"`
	RecordedCellCount  int              `json:"recordedCellCount"`
	CompletedDays      []int            `json:"completedDays"`
	CurrentDay         int              `json:"currentDay"`
	MissingCellCount   int              `json:"missingCellCount"`
	PendingCellCount   int              `json:"pendingCellCount"`
	CompletionPercent  float64          `json:"completionPercent"`
	RecordedByDay      map[int]int      `json:"recordedByDay"`
	MissingCoordinates []ObservationGap `json:"missingCoordinates"`
	PendingCoordinates []ObservationGap `json:"pendingCoordinates"`
}

type ReviewReadiness struct {
	Eligible               bool                `json:"eligible"`
	StatusAllowed          bool                `json:"statusAllowed"`
	ProtocolLocked         bool                `json:"protocolLocked"`
	ObservationComplete    bool                `json:"observationComplete"`
	ThresholdMet           bool                `json:"thresholdMet"`
	OpenDeviationCount     int                 `json:"openDeviationCount"`
	PendingCorrectionCount int                 `json:"pendingCorrectionCount"`
	Progress               ObservationProgress `json:"progress"`
	Blockers               []ReviewBlocker     `json:"blockers"`
}

func AssessReviewReadiness(t *Trial, metrics Metrics) ReviewReadiness {
	report := ReviewReadiness{
		StatusAllowed:      t.Status == Observing || t.Status == CorrectionRequired,
		ProtocolLocked:     t.Protocol != nil,
		ThresholdMet:       metrics.ThresholdMet,
		Progress:           buildObservationProgress(t, metrics.Gaps),
		Blockers:           []ReviewBlocker{},
		OpenDeviationCount: countOpenDeviations(t.Deviations),
		PendingCorrectionCount: countPendingCorrections(
			t.CorrectionItems,
		),
	}
	report.ObservationComplete = report.Progress.MissingCellCount == 0 && report.Progress.PendingCellCount == 0

	if !report.StatusAllowed {
		report.Blockers = append(report.Blockers, ReviewBlocker{
			Code:       BlockerStatus,
			Message:    fmt.Sprintf("状态 %s 不能送审", t.Status),
			ObjectType: "trial",
			ObjectID:   t.TrialID,
		})
	}
	if !report.ProtocolLocked {
		report.Blockers = append(report.Blockers, ReviewBlocker{
			Code:       BlockerProtocol,
			Message:    "处理方案尚未锁定",
			ObjectType: "protocol",
		})
	}
	report.Blockers = append(report.Blockers, observationBlockers(metrics.Gaps)...)
	report.Blockers = append(report.Blockers, deviationBlockers(t.Deviations)...)
	report.Blockers = append(report.Blockers, correctionBlockers(t.CorrectionItems)...)
	if !report.ThresholdMet {
		report.Blockers = append(report.Blockers, ReviewBlocker{
			Code:       BlockerThreshold,
			Message:    metrics.ThresholdConclusion,
			ObjectType: "metrics",
			ObjectID:   "metrics",
		})
	}

	sortReviewBlockers(report.Blockers)
	report.Eligible = len(report.Blockers) == 0
	return report
}

func buildObservationProgress(t *Trial, gaps []ObservationGap) ObservationProgress {
	period := 0
	if t.Protocol != nil {
		period = t.Protocol.ObservationDays
	}
	progress := ObservationProgress{
		ObservationDays:    period,
		ReplicateCount:     t.ReplicateCount,
		ExpectedCellCount:  period * t.ReplicateCount,
		CompletedDays:      []int{},
		RecordedByDay:      map[int]int{},
		MissingCoordinates: []ObservationGap{},
		PendingCoordinates: []ObservationGap{},
	}
	seen := map[string]bool{}
	for _, observation := range t.Observations {
		if observation.DayIndex < 1 || observation.DayIndex > period {
			continue
		}
		if observation.ReplicateIndex < 1 || observation.ReplicateIndex > t.ReplicateCount {
			continue
		}
		key := fmt.Sprintf("%d:%d", observation.DayIndex, observation.ReplicateIndex)
		if seen[key] {
			continue
		}
		seen[key] = true
		progress.RecordedCellCount++
		progress.RecordedByDay[observation.DayIndex]++
		if observation.DayIndex > progress.CurrentDay {
			progress.CurrentDay = observation.DayIndex
		}
	}
	for day := 1; day <= period; day++ {
		if progress.RecordedByDay[day] == t.ReplicateCount {
			progress.CompletedDays = append(progress.CompletedDays, day)
		}
	}
	for _, gap := range expandObservationGaps(gaps, t.ReplicateCount) {
		if strings.HasPrefix(gap.Kind, "PENDING") {
			progress.PendingCellCount++
			progress.PendingCoordinates = append(progress.PendingCoordinates, gap)
			continue
		}
		progress.MissingCellCount++
		progress.MissingCoordinates = append(progress.MissingCoordinates, gap)
	}
	if progress.ExpectedCellCount > 0 {
		progress.CompletionPercent = round(float64(progress.RecordedCellCount) * 100 / float64(progress.ExpectedCellCount))
	}
	return progress
}

func expandObservationGaps(gaps []ObservationGap, replicates int) []ObservationGap {
	expanded := make([]ObservationGap, 0, len(gaps))
	for _, gap := range gaps {
		if gap.ReplicateIndex != 0 {
			expanded = append(expanded, gap)
			continue
		}
		kind := "MISSING"
		if strings.HasPrefix(gap.Kind, "PENDING") {
			kind = "PENDING"
		}
		for replicate := 1; replicate <= replicates; replicate++ {
			expanded = append(expanded, ObservationGap{
				DayIndex:       gap.DayIndex,
				ReplicateIndex: replicate,
				Kind:           kind,
			})
		}
	}
	return expanded
}

func observationBlockers(gaps []ObservationGap) []ReviewBlocker {
	blockers := make([]ReviewBlocker, 0, len(gaps))
	for _, gap := range gaps {
		blocker := ReviewBlocker{
			DayIndex:       gap.DayIndex,
			ReplicateIndex: gap.ReplicateIndex,
			ObjectType:     "observation",
			ObjectID:       fmt.Sprintf("day-%d", gap.DayIndex),
		}
		switch gap.Kind {
		case "MISSING_DAY":
			blocker.Code = BlockerObservationDay
			blocker.Message = fmt.Sprintf("第%d日观察全部缺失", gap.DayIndex)
		case "MISSING":
			blocker.Code = BlockerObservationGroup
			blocker.Message = fmt.Sprintf("第%d日重复组%d观察缺失", gap.DayIndex, gap.ReplicateIndex)
		default:
			blocker.Code = BlockerObservationPending
			blocker.Message = fmt.Sprintf("第%d日观察尚待录入", gap.DayIndex)
		}
		blockers = append(blockers, blocker)
	}
	return blockers
}

func deviationBlockers(deviations []Deviation) []ReviewBlocker {
	blockers := []ReviewBlocker{}
	for _, deviation := range deviations {
		if deviation.Resolved {
			continue
		}
		blockers = append(blockers, ReviewBlocker{
			Code:       BlockerDeviation,
			Message:    fmt.Sprintf("偏差 %s 尚未解决", deviation.ID),
			ObjectType: "deviation",
			ObjectID:   deviation.ID,
		})
	}
	return blockers
}

func correctionBlockers(items []CorrectionItem) []ReviewBlocker {
	blockers := []ReviewBlocker{}
	for _, item := range items {
		if item.Status == "CONFIRMED" {
			continue
		}
		blockers = append(blockers, ReviewBlocker{
			Code:       BlockerCorrection,
			Message:    fmt.Sprintf("退回问题 %s 尚未确认", item.ID),
			ObjectType: "correction-item",
			ObjectID:   item.ID,
		})
	}
	return blockers
}

func countOpenDeviations(deviations []Deviation) int {
	count := 0
	for _, deviation := range deviations {
		if !deviation.Resolved {
			count++
		}
	}
	return count
}

func countPendingCorrections(items []CorrectionItem) int {
	count := 0
	for _, item := range items {
		if item.Status != "CONFIRMED" {
			count++
		}
	}
	return count
}

func sortReviewBlockers(blockers []ReviewBlocker) {
	sort.SliceStable(blockers, func(i, j int) bool {
		left, right := blockers[i], blockers[j]
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.DayIndex != right.DayIndex {
			return left.DayIndex < right.DayIndex
		}
		if left.ReplicateIndex != right.ReplicateIndex {
			return left.ReplicateIndex < right.ReplicateIndex
		}
		return left.ObjectID < right.ObjectID
	})
}
