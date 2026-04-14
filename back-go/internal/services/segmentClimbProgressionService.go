package services

import (
	"fmt"
	"log"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	"sort"
	"strings"
)

type segmentMetric string

const (
	segmentMetricTime  segmentMetric = "TIME"
	segmentMetricSpeed segmentMetric = "SPEED"
)

type segmentTargetType string

const (
	segmentTargetTypeAll     segmentTargetType = "ALL"
	segmentTargetTypeSegment segmentTargetType = "SEGMENT"
	segmentTargetTypeClimb   segmentTargetType = "CLIMB"
)

type segmentAttemptRaw struct {
	effortId           int64
	targetId           int64
	targetName         string
	targetType         segmentTargetType
	climbCategory      int
	distance           float64
	averageGrade       float64
	elapsedTimeSeconds int
	movingTimeSeconds  int
	speedKph           float64
	averagePowerWatts  float64
	averageHeartRate   float64
	activityDate       string
	prRank             *int
	activity           business.ActivityShort
}

func FetchSegmentClimbProgressionByActivityTypeAndYear(
	year *int,
	metric *string,
	targetType *string,
	targetId *int64,
	activityTypes ...business.ActivityType,
) business.SegmentClimbProgression {
	resolvedMetric := parseSegmentMetric(metric)
	resolvedTargetType := parseSegmentTargetType(targetType)
	effectiveTargetType := resolvedTargetType

	if len(activityTypes) == 0 {
		return emptySegmentClimbProgression(resolvedMetric, resolvedTargetType)
	}

	filteredActivities := getActivityProvider().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	if len(filteredActivities) == 0 && year != nil {
		log.Printf(
			"Segment progression fallback to all years: requestedYear=%v targetType=%s activities=0",
			yearValueForLog(year),
			resolvedTargetType,
		)
		filteredActivities = getActivityProvider().GetActivitiesByYearAndActivityTypes(nil, activityTypes...)
	}
	sort.Slice(filteredActivities, func(i, j int) bool {
		return filteredActivities[i].StartDateLocal < filteredActivities[j].StartDateLocal
	})

	rawFavoriteAttempts := make([]segmentAttemptRaw, 0)
	rawFavoriteAttemptsAllTypes := make([]segmentAttemptRaw, 0)
	rawAllAttempts := make([]segmentAttemptRaw, 0)
	rawAllAttemptsAllTypes := make([]segmentAttemptRaw, 0)
	detailedLoadedCount := 0
	detailedMissingCount := 0
	totalCandidateEfforts := 0
	for _, activity := range filteredActivities {
		detailedActivity := getActivityProvider().GetCachedDetailedActivity(activity.Id)
		if detailedActivity == nil {
			detailedActivity = getActivityProvider().GetDetailedActivity(activity.Id)
		}
		if detailedActivity == nil {
			detailedMissingCount++
			continue
		}
		detailedLoadedCount++

		for _, effort := range detailedActivity.SegmentEfforts {
			if !isCandidateEffort(effort) {
				continue
			}
			totalCandidateEfforts++

			effortTargetType := segmentTargetTypeSegment
			if effort.Segment.ClimbCategory > 0 {
				effortTargetType = segmentTargetTypeClimb
			}
			if resolvedTargetType != segmentTargetTypeAll && effortTargetType != resolvedTargetType {
				continue
			}

			attempt := segmentAttemptRaw{
				effortId:           effort.Id,
				targetId:           effort.Segment.Id,
				targetName:         effort.Segment.Name,
				targetType:         effortTargetType,
				climbCategory:      effort.Segment.ClimbCategory,
				distance:           effort.Distance,
				averageGrade:       effort.Segment.AverageGrade,
				elapsedTimeSeconds: effort.ElapsedTime,
				movingTimeSeconds:  effort.MovingTime,
				speedKph:           computeSpeedKph(effort.Distance, effort.ElapsedTime),
				averagePowerWatts:  effort.AverageWatts,
				averageHeartRate:   effort.AverageHeartRate,
				activityDate:       effort.StartDateLocal,
				prRank:             effort.PrRank,
				activity:           toActivityShort(activity),
			}

			rawAllAttemptsAllTypes = append(rawAllAttemptsAllTypes, attempt)
			if isFavoriteEffort(effort) {
				rawFavoriteAttemptsAllTypes = append(rawFavoriteAttemptsAllTypes, attempt)
			}

			if resolvedTargetType == segmentTargetTypeAll || effortTargetType == resolvedTargetType {
				rawAllAttempts = append(rawAllAttempts, attempt)
				if isFavoriteEffort(effort) {
					rawFavoriteAttempts = append(rawFavoriteAttempts, attempt)
				}
			}
		}
	}

	rawAttempts := rawFavoriteAttempts
	if len(rawAttempts) == 0 && len(rawAllAttempts) > 0 {
		log.Printf(
			"Segment progression fallback to all efforts (favorites empty): year=%v activities=%d detailedLoaded=%d detailedMissing=%d candidateEfforts=%d",
			yearValueForLog(year),
			len(filteredActivities),
			detailedLoadedCount,
			detailedMissingCount,
			totalCandidateEfforts,
		)
		rawAttempts = rawAllAttempts
	} else {
		log.Printf(
			"Segment progression: year=%v activities=%d detailedLoaded=%d detailedMissing=%d candidateEfforts=%d favoriteEfforts=%d",
			yearValueForLog(year),
			len(filteredActivities),
			detailedLoadedCount,
			detailedMissingCount,
			totalCandidateEfforts,
			len(rawFavoriteAttempts),
		)
	}

	if len(rawAttempts) == 0 && resolvedTargetType != segmentTargetTypeAll {
		effectiveTargetType = segmentTargetTypeAll
		rawAttempts = rawFavoriteAttemptsAllTypes
		if len(rawAttempts) == 0 {
			rawAttempts = rawAllAttemptsAllTypes
		}
		log.Printf(
			"Segment progression fallback to targetType=ALL: requestedTargetType=%s fallbackAttempts=%d",
			resolvedTargetType,
			len(rawAttempts),
		)
	}

	if len(rawAttempts) == 0 {
		if year != nil {
			// Fallback for UX: when selected year has no segment data yet, reuse all-years data.
			return FetchSegmentClimbProgressionByActivityTypeAndYear(nil, metric, strPtr(string(effectiveTargetType)), targetId, activityTypes...)
		}
		return emptySegmentClimbProgression(resolvedMetric, resolvedTargetType)
	}

	attemptsByTarget := make(map[int64][]segmentAttemptRaw)
	for _, attempt := range rawAttempts {
		attemptsByTarget[attempt.targetId] = append(attemptsByTarget[attempt.targetId], attempt)
	}

	targetSummaries := make([]business.SegmentClimbTargetSummary, 0, len(attemptsByTarget))
	for _, attempts := range attemptsByTarget {
		targetSummaries = append(targetSummaries, buildTargetSummary(attempts, resolvedMetric))
	}

	sort.Slice(targetSummaries, func(i, j int) bool {
		if targetSummaries[i].AttemptsCount != targetSummaries[j].AttemptsCount {
			return targetSummaries[i].AttemptsCount > targetSummaries[j].AttemptsCount
		}
		return strings.ToLower(targetSummaries[i].TargetName) < strings.ToLower(targetSummaries[j].TargetName)
	})

	selectedTarget := resolveSelectedTarget(targetSummaries, targetId)
	selectedAttempts := []business.SegmentClimbAttempt{}
	var selectedTargetID *int64
	var selectedTargetTypeValue *string
	if selectedTarget != nil {
		selectedAttempts = buildAttempts(attemptsByTarget[selectedTarget.TargetId], resolvedMetric)
		id := selectedTarget.TargetId
		kind := selectedTarget.TargetType
		selectedTargetID = &id
		selectedTargetTypeValue = &kind
	}

	return business.SegmentClimbProgression{
		Metric:                  string(resolvedMetric),
		TargetTypeFilter:        string(effectiveTargetType),
		WeatherContextAvailable: false,
		Targets:                 targetSummaries,
		SelectedTargetId:        selectedTargetID,
		SelectedTargetType:      selectedTargetTypeValue,
		Attempts:                selectedAttempts,
	}
}

func emptySegmentClimbProgression(metric segmentMetric, targetType segmentTargetType) business.SegmentClimbProgression {
	return business.SegmentClimbProgression{
		Metric:                  string(metric),
		TargetTypeFilter:        string(targetType),
		WeatherContextAvailable: false,
		Targets:                 []business.SegmentClimbTargetSummary{},
		SelectedTargetId:        nil,
		SelectedTargetType:      nil,
		Attempts:                []business.SegmentClimbAttempt{},
	}
}

func resolveSelectedTarget(
	targets []business.SegmentClimbTargetSummary,
	targetId *int64,
) *business.SegmentClimbTargetSummary {
	if targetId != nil {
		for i := range targets {
			if targets[i].TargetId == *targetId {
				return &targets[i]
			}
		}
	}
	if len(targets) == 0 {
		return nil
	}
	return &targets[0]
}

func isFavoriteEffort(effort strava.SegmentEffort) bool {
	return effort.Segment.Starred ||
		effort.Segment.ClimbCategory > 0 ||
		(effort.PrRank != nil && *effort.PrRank <= 3)
}

func isCandidateEffort(effort strava.SegmentEffort) bool {
	if effort.Segment.Id == 0 {
		return false
	}
	if strings.TrimSpace(effort.Segment.Name) == "" {
		return false
	}
	if effort.ElapsedTime <= 0 || effort.Distance <= 0 {
		return false
	}

	return true
}

func yearValueForLog(year *int) string {
	if year == nil {
		return "all"
	}
	return fmt.Sprintf("%d", *year)
}

func strPtr(value string) *string {
	return &value
}

func computeSpeedKph(distanceInMeters float64, elapsedTimeSeconds int) float64 {
	if distanceInMeters <= 0 || elapsedTimeSeconds <= 0 {
		return 0
	}
	return (distanceInMeters / float64(elapsedTimeSeconds)) * 3.6
}

func buildAttempts(attempts []segmentAttemptRaw, metric segmentMetric) []business.SegmentClimbAttempt {
	sortedAttempts := append([]segmentAttemptRaw(nil), attempts...)
	sort.Slice(sortedAttempts, func(i, j int) bool {
		return sortedAttempts[i].activityDate < sortedAttempts[j].activityDate
	})

	bestRankedAttempts := append([]segmentAttemptRaw(nil), attempts...)
	sort.Slice(bestRankedAttempts, func(i, j int) bool {
		left := bestRankedAttempts[i]
		right := bestRankedAttempts[j]
		if metric == segmentMetricSpeed {
			if left.speedKph != right.speedKph {
				return left.speedKph > right.speedKph
			}
		} else {
			if left.elapsedTimeSeconds != right.elapsedTimeSeconds {
				return left.elapsedTimeSeconds < right.elapsedTimeSeconds
			}
		}
		if left.activityDate != right.activityDate {
			return left.activityDate < right.activityDate
		}
		return left.effortId < right.effortId
	})

	personalRankByEffortID := make(map[int64]int, len(bestRankedAttempts))
	for index, attempt := range bestRankedAttempts {
		if attempt.effortId <= 0 {
			continue
		}
		personalRankByEffortID[attempt.effortId] = index + 1
	}

	bestValue := 0.0
	switch metric {
	case segmentMetricSpeed:
		bestValue = math.Inf(-1)
		for _, attempt := range sortedAttempts {
			if attempt.speedKph > bestValue {
				bestValue = attempt.speedKph
			}
		}
	default:
		bestValue = math.Inf(1)
		for _, attempt := range sortedAttempts {
			value := float64(attempt.elapsedTimeSeconds)
			if value < bestValue {
				bestValue = value
			}
		}
	}

	bestSoFar := math.Inf(1)
	if metric == segmentMetricSpeed {
		bestSoFar = math.Inf(-1)
	}

	progression := make([]business.SegmentClimbAttempt, 0, len(sortedAttempts))
	for _, attempt := range sortedAttempts {
		currentMetricValue := float64(attempt.elapsedTimeSeconds)
		if metric == segmentMetricSpeed {
			currentMetricValue = attempt.speedKph
		}

		setsNewPr := currentMetricValue < bestSoFar
		if metric == segmentMetricSpeed {
			setsNewPr = currentMetricValue > bestSoFar
		}
		if setsNewPr {
			bestSoFar = currentMetricValue
		}

		closeToPr := false
		switch metric {
		case segmentMetricSpeed:
			closeToPr = !setsNewPr && currentMetricValue >= bestValue*0.97
		default:
			closeToPr = !setsNewPr && currentMetricValue <= bestValue*1.03
		}

		deltaToPr := "PR"
		switch metric {
		case segmentMetricSpeed:
			if bestValue > 0 {
				delta := bestValue - currentMetricValue
				if delta > 0 {
					deltaToPr = fmt.Sprintf("-%.1f%%", (delta/bestValue)*100.0)
				}
			}
		default:
			delta := int(math.Round(currentMetricValue - bestValue))
			if delta > 0 {
				deltaToPr = fmt.Sprintf("+%s", helpers.FormatSeconds(delta))
			}
		}

		var personalRank *int
		if rank, ok := personalRankByEffortID[attempt.effortId]; ok {
			rankCopy := rank
			personalRank = &rankCopy
		}

		progression = append(progression, business.SegmentClimbAttempt{
			TargetId:           attempt.targetId,
			TargetName:         attempt.targetName,
			TargetType:         string(attempt.targetType),
			ActivityDate:       attempt.activityDate,
			ElapsedTimeSeconds: attempt.elapsedTimeSeconds,
			MovingTimeSeconds:  attempt.movingTimeSeconds,
			SpeedKph:           attempt.speedKph,
			Distance:           attempt.distance,
			AverageGrade:       attempt.averageGrade,
			ElevationGain:      (attempt.distance * attempt.averageGrade) / 100.0,
			AveragePowerWatts:  attempt.averagePowerWatts,
			AverageHeartRate:   attempt.averageHeartRate,
			PrRank:             attempt.prRank,
			PersonalRank:       personalRank,
			SetsNewPr:          setsNewPr,
			CloseToPr:          closeToPr,
			DeltaToPr:          deltaToPr,
			WeatherSummary:     nil,
			Activity:           attempt.activity,
		})
	}

	return progression
}

func buildTargetSummary(attempts []segmentAttemptRaw, metric segmentMetric) business.SegmentClimbTargetSummary {
	progressionAttempts := buildAttempts(attempts, metric)
	latestAttempt := progressionAttempts[len(progressionAttempts)-1]

	bestAttempt := progressionAttempts[0]
	for _, attempt := range progressionAttempts[1:] {
		switch metric {
		case segmentMetricSpeed:
			if attempt.SpeedKph > bestAttempt.SpeedKph {
				bestAttempt = attempt
			}
		default:
			if attempt.ElapsedTimeSeconds < bestAttempt.ElapsedTimeSeconds {
				bestAttempt = attempt
			}
		}
	}

	averageSpeedKph := average(mapAttemptsToSpeeds(progressionAttempts))

	values := mapAttemptsToMetricValues(progressionAttempts, metric)
	lowerIsBetter := metric == segmentMetricTime
	consistency := consistencyLabel(values)
	recentTrend := trendLabel(values, lowerIsBetter, metricUnit(metric))

	bestValue := fmt.Sprintf("%.1f km/h", bestAttempt.SpeedKph)
	latestValue := fmt.Sprintf("%.1f km/h", latestAttempt.SpeedKph)
	if metric == segmentMetricTime {
		bestValue = helpers.FormatSeconds(bestAttempt.ElapsedTimeSeconds)
		latestValue = helpers.FormatSeconds(latestAttempt.ElapsedTimeSeconds)
	}

	firstAttempt := attempts[0]
	closeToPrCount := 0
	for _, attempt := range progressionAttempts {
		if attempt.CloseToPr {
			closeToPrCount++
		}
	}

	return business.SegmentClimbTargetSummary{
		TargetId:       latestAttempt.TargetId,
		TargetName:     latestAttempt.TargetName,
		TargetType:     latestAttempt.TargetType,
		ClimbCategory:  firstAttempt.climbCategory,
		Distance:       firstAttempt.distance,
		AverageGrade:   firstAttempt.averageGrade,
		AttemptsCount:  len(progressionAttempts),
		BestValue:      bestValue,
		LatestValue:    latestValue,
		Consistency:    consistency,
		AveragePacing:  fmt.Sprintf("%.1f km/h", averageSpeedKph),
		CloseToPrCount: closeToPrCount,
		RecentTrend:    recentTrend,
	}
}

func mapAttemptsToMetricValues(attempts []business.SegmentClimbAttempt, metric segmentMetric) []float64 {
	values := make([]float64, 0, len(attempts))
	for _, attempt := range attempts {
		if metric == segmentMetricSpeed {
			values = append(values, attempt.SpeedKph)
		} else {
			values = append(values, float64(attempt.ElapsedTimeSeconds))
		}
	}
	return values
}

func mapAttemptsToSpeeds(attempts []business.SegmentClimbAttempt) []float64 {
	speeds := make([]float64, 0, len(attempts))
	for _, attempt := range attempts {
		speeds = append(speeds, attempt.SpeedKph)
	}
	return speeds
}

func consistencyLabel(values []float64) string {
	if len(values) < 3 {
		return "-"
	}
	mean := average(values)
	if mean == 0 {
		return "-"
	}

	var varianceSum float64
	for _, value := range values {
		diff := value - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(values))
	coefficientOfVariation := math.Sqrt(variance) / mean * 100.0

	return fmt.Sprintf("CV %.1f%%", coefficientOfVariation)
}

func trendLabel(values []float64, lowerIsBetter bool, unit string) string {
	if len(values) < 6 {
		return "Not enough data"
	}

	recentValues := values[len(values)-3:]
	previousValues := values[len(values)-6 : len(values)-3]
	recentAverage := average(recentValues)
	previousAverage := average(previousValues)
	if previousAverage == 0 {
		return "Stable"
	}

	ratio := (recentAverage - previousAverage) / previousAverage
	isImproving := ratio > 0
	if lowerIsBetter {
		isImproving = ratio < 0
	}

	percentage := math.Abs(ratio * 100.0)
	switch {
	case percentage < 1.0:
		return "Stable"
	case isImproving:
		return fmt.Sprintf("Improving %.1f%% (%s)", percentage, unit)
	default:
		return fmt.Sprintf("Declining %.1f%% (%s)", percentage, unit)
	}
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func metricUnit(metric segmentMetric) string {
	if metric == segmentMetricSpeed {
		return "speed"
	}
	return "time"
}

func parseSegmentMetric(metric *string) segmentMetric {
	if metric != nil && strings.EqualFold(strings.TrimSpace(*metric), string(segmentMetricSpeed)) {
		return segmentMetricSpeed
	}
	return segmentMetricTime
}

func parseSegmentTargetType(targetType *string) segmentTargetType {
	if targetType == nil {
		return segmentTargetTypeAll
	}

	switch strings.ToUpper(strings.TrimSpace(*targetType)) {
	case string(segmentTargetTypeSegment):
		return segmentTargetTypeSegment
	case string(segmentTargetTypeClimb):
		return segmentTargetTypeClimb
	default:
		return segmentTargetTypeAll
	}
}

func toActivityShort(activity *strava.Activity) business.ActivityShort {
	return business.ActivityShort{
		Id:   activity.Id,
		Name: activity.Name,
		Type: resolveActivityType(activity),
	}
}

func resolveActivityType(activity *strava.Activity) business.ActivityType {
	sportType := activity.SportType
	if sportType == "" {
		sportType = activity.Type
	}

	if activity.Commute && strings.EqualFold(sportType, business.Ride.String()) {
		return business.Commute
	}

	if activityType, ok := business.ActivityTypes[sportType]; ok {
		return activityType
	}
	if activityType, ok := business.ActivityTypes[activity.Type]; ok {
		return activityType
	}

	return business.Ride
}
