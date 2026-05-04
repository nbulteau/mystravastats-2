package application

import (
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strings"
	"time"
)

const (
	activityComparisonMaxCandidates      = 5
	activityComparisonMaxSimilarityScore = 0.45
	activityComparisonSegmentDetailLimit = 12
	activityComparisonMinDistanceScale   = 1000.0
	activityComparisonMinElevationScale  = 100.0
)

type GetActivityComparisonUseCase struct {
	reader ActivityComparisonReader
}

func NewGetActivityComparisonUseCase(reader ActivityComparisonReader) *GetActivityComparisonUseCase {
	return &GetActivityComparisonUseCase{reader: reader}
}

type ActivityComparison struct {
	Status            string
	Label             string
	Criteria          ActivityComparisonCriteria
	Baseline          ActivityComparisonBaseline
	Deltas            ActivityComparisonDeltas
	SimilarActivities []ActivityComparisonActivity
	CommonSegments    []ActivityComparisonSegment
}

type ActivityComparisonCriteria struct {
	ActivityType string
	Year         int
	SampleSize   int
}

type ActivityComparisonBaseline struct {
	Distance         float64
	ElevationGain    float64
	MovingTime       int
	AverageSpeed     float64
	AverageHeartrate float64
	AverageWatts     float64
	AverageCadence   float64
}

type ActivityComparisonDeltas struct {
	Distance         float64
	ElevationGain    float64
	MovingTime       int
	AverageSpeed     float64
	AverageSpeedPct  float64
	AverageHeartrate float64
	AverageWatts     float64
	AverageCadence   float64
}

type ActivityComparisonActivity struct {
	ID               int64
	Name             string
	Date             string
	Distance         float64
	ElevationGain    float64
	MovingTime       int
	AverageSpeed     float64
	AverageHeartrate float64
	AverageWatts     float64
	AverageCadence   float64
	SimilarityScore  float64
}

type ActivityComparisonSegment struct {
	ID            int64
	Name          string
	MatchCount    int
	ActivityIDs   []int64
	ActivityNames []string
}

func (uc *GetActivityComparisonUseCase) Execute(target *strava.DetailedActivity) *ActivityComparison {
	if uc == nil || uc.reader == nil || target == nil {
		return nil
	}

	activityType, ok := resolveComparisonActivityType(target)
	if !ok {
		return nil
	}
	year, ok := resolveComparisonYear(target)
	if !ok {
		return nil
	}

	candidates := uc.reader.FindActivitiesByYearAndTypes(&year, activityType)
	ranked := rankSimilarActivities(target, candidates)
	selected := topSimilarActivities(ranked, activityComparisonMaxCandidates)

	comparison := &ActivityComparison{
		Status: "insufficient-data",
		Label:  "Not enough similar activities",
		Criteria: ActivityComparisonCriteria{
			ActivityType: activityType.String(),
			Year:         year,
			SampleSize:   len(selected),
		},
		SimilarActivities: selected,
	}

	if len(selected) == 0 {
		return comparison
	}

	comparison.Baseline = buildComparisonBaseline(selected)
	comparison.Deltas = buildComparisonDeltas(target, comparison.Baseline)
	comparison.Status, comparison.Label = classifyComparison(comparison.Deltas)
	comparison.CommonSegments = findCommonSegments(target, selected, uc.reader)
	return comparison
}

func resolveComparisonActivityType(target *strava.DetailedActivity) (business.ActivityType, bool) {
	if target.Commute {
		return business.Commute, true
	}
	for _, value := range []string{target.SportType, target.Type} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if activityType, ok := business.ActivityTypes[value]; ok {
			return activityType, true
		}
	}
	return 0, false
}

func resolveComparisonYear(target *strava.DetailedActivity) (int, bool) {
	for _, value := range []string{target.StartDateLocal, target.StartDate} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) >= len("2006-01-02") {
			if parsed, err := time.Parse("2006-01-02", value[:len("2006-01-02")]); err == nil {
				return parsed.Year(), true
			}
		}
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed.Year(), true
		}
	}
	return 0, false
}

func rankSimilarActivities(target *strava.DetailedActivity, candidates []*strava.Activity) []ActivityComparisonActivity {
	ranked := make([]ActivityComparisonActivity, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.Id == target.Id || candidate.Distance <= 0 || candidate.MovingTime <= 0 {
			continue
		}
		score := similarActivityScore(target, candidate)
		if math.IsInf(score, 0) || math.IsNaN(score) {
			continue
		}
		if score > activityComparisonMaxSimilarityScore {
			continue
		}
		ranked = append(ranked, ActivityComparisonActivity{
			ID:               candidate.Id,
			Name:             candidate.Name,
			Date:             firstNonEmpty(candidate.StartDateLocal, candidate.StartDate),
			Distance:         candidate.Distance,
			ElevationGain:    candidate.TotalElevationGain,
			MovingTime:       candidate.MovingTime,
			AverageSpeed:     candidate.AverageSpeed,
			AverageHeartrate: candidate.AverageHeartrate,
			AverageWatts:     candidate.AverageWatts,
			AverageCadence:   candidate.AverageCadence,
			SimilarityScore:  score,
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].SimilarityScore == ranked[j].SimilarityScore {
			return ranked[i].Date > ranked[j].Date
		}
		return ranked[i].SimilarityScore < ranked[j].SimilarityScore
	})
	return ranked
}

func similarActivityScore(target *strava.DetailedActivity, candidate *strava.Activity) float64 {
	distanceScore := ratioDelta(float64(target.Distance), candidate.Distance, activityComparisonMinDistanceScale)
	elevationScore := ratioDelta(float64(target.TotalElevationGain), candidate.TotalElevationGain, activityComparisonMinElevationScale)
	return distanceScore*0.62 + elevationScore*0.38
}

func ratioDelta(target, value, minScale float64) float64 {
	denominator := math.Max(math.Abs(target), minScale)
	return math.Abs(value-target) / denominator
}

func topSimilarActivities(ranked []ActivityComparisonActivity, limit int) []ActivityComparisonActivity {
	if len(ranked) < limit {
		limit = len(ranked)
	}
	if limit <= 0 {
		return []ActivityComparisonActivity{}
	}
	return append([]ActivityComparisonActivity(nil), ranked[:limit]...)
}

func buildComparisonBaseline(activities []ActivityComparisonActivity) ActivityComparisonBaseline {
	return ActivityComparisonBaseline{
		Distance:         averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return activity.Distance }, false),
		ElevationGain:    averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return activity.ElevationGain }, false),
		MovingTime:       int(math.Round(averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return float64(activity.MovingTime) }, false))),
		AverageSpeed:     averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return activity.AverageSpeed }, false),
		AverageHeartrate: averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return activity.AverageHeartrate }, true),
		AverageWatts:     averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return activity.AverageWatts }, true),
		AverageCadence:   averageFloat(activities, func(activity ActivityComparisonActivity) float64 { return activity.AverageCadence }, true),
	}
}

func buildComparisonDeltas(target *strava.DetailedActivity, baseline ActivityComparisonBaseline) ActivityComparisonDeltas {
	speedDelta := finiteDelta(target.AverageSpeed, baseline.AverageSpeed)
	return ActivityComparisonDeltas{
		Distance:         finiteDelta(target.Distance, baseline.Distance),
		ElevationGain:    finiteDelta(target.TotalElevationGain, baseline.ElevationGain),
		MovingTime:       target.MovingTime - baseline.MovingTime,
		AverageSpeed:     speedDelta,
		AverageSpeedPct:  percentageDelta(speedDelta, baseline.AverageSpeed),
		AverageHeartrate: finiteDelta(target.AverageHeartrate, baseline.AverageHeartrate),
		AverageWatts:     finiteDelta(target.AverageWatts, baseline.AverageWatts),
		AverageCadence:   finiteDelta(target.AverageCadence, baseline.AverageCadence),
	}
}

func classifyComparison(deltas ActivityComparisonDeltas) (string, string) {
	if math.Abs(deltas.AverageSpeedPct) >= 15 {
		return "atypical", "Atypical pace for similar activities"
	}
	if deltas.AverageSpeedPct >= 5 {
		return "faster", "Faster than similar activities"
	}
	if deltas.AverageSpeedPct <= -5 {
		return "slower", "Slower than similar activities"
	}
	return "typical", "In line with similar activities"
}

func findCommonSegments(target *strava.DetailedActivity, activities []ActivityComparisonActivity, reader ActivityComparisonReader) []ActivityComparisonSegment {
	targetSegments := targetSegmentNames(target)
	if len(targetSegments) == 0 {
		return []ActivityComparisonSegment{}
	}

	commonByID := make(map[int64]*ActivityComparisonSegment)
	for _, activity := range activities {
		detailed := reader.FindCachedDetailedActivityByID(activity.ID)
		if detailed == nil {
			continue
		}
		seenForActivity := map[int64]struct{}{}
		for _, effort := range detailed.SegmentEfforts {
			segmentID := effort.Segment.Id
			name, ok := targetSegments[segmentID]
			if !ok {
				continue
			}
			if _, seen := seenForActivity[segmentID]; seen {
				continue
			}
			seenForActivity[segmentID] = struct{}{}
			common := commonByID[segmentID]
			if common == nil {
				common = &ActivityComparisonSegment{ID: segmentID, Name: name}
				commonByID[segmentID] = common
			}
			common.MatchCount++
			common.ActivityIDs = append(common.ActivityIDs, activity.ID)
			common.ActivityNames = append(common.ActivityNames, activity.Name)
		}
	}

	commonSegments := make([]ActivityComparisonSegment, 0, len(commonByID))
	for _, segment := range commonByID {
		commonSegments = append(commonSegments, *segment)
	}
	sort.SliceStable(commonSegments, func(i, j int) bool {
		if commonSegments[i].MatchCount == commonSegments[j].MatchCount {
			return commonSegments[i].Name < commonSegments[j].Name
		}
		return commonSegments[i].MatchCount > commonSegments[j].MatchCount
	})
	if len(commonSegments) > activityComparisonSegmentDetailLimit {
		commonSegments = commonSegments[:activityComparisonSegmentDetailLimit]
	}
	return commonSegments
}

func targetSegmentNames(target *strava.DetailedActivity) map[int64]string {
	segments := make(map[int64]string)
	for _, effort := range target.SegmentEfforts {
		if effort.Segment.Id == 0 {
			continue
		}
		name := strings.TrimSpace(effort.Segment.Name)
		if name == "" {
			name = strings.TrimSpace(effort.Name)
		}
		if name == "" {
			continue
		}
		segments[effort.Segment.Id] = name
	}
	return segments
}

func averageFloat(activities []ActivityComparisonActivity, getter func(ActivityComparisonActivity) float64, ignoreZero bool) float64 {
	sum := 0.0
	count := 0
	for _, activity := range activities {
		value := getter(activity)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		if ignoreZero && value <= 0 {
			continue
		}
		sum += value
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func finiteDelta(target, baseline float64) float64 {
	if math.IsNaN(target) || math.IsInf(target, 0) || math.IsNaN(baseline) || math.IsInf(baseline, 0) {
		return 0
	}
	if baseline == 0 {
		return 0
	}
	return target - baseline
}

func percentageDelta(delta, baseline float64) float64 {
	if baseline == 0 || math.IsNaN(delta) || math.IsInf(delta, 0) {
		return 0
	}
	return delta / baseline * 100
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
