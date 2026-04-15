package infrastructure

import (
	"hash/fnv"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"mystravastats/internal/platform/activityprovider"
	"sort"
	"strings"
)

type SegmentSummaryResult struct {
	Metric         string
	Segment        business.SegmentClimbTargetSummary
	PersonalRecord *business.SegmentClimbAttempt
	TopEfforts     []business.SegmentClimbAttempt
	Attempts       []business.SegmentClimbAttempt
}

func computeSegmentsByYearMetricQueryRangeAndTypes(
	year *int,
	metric *string,
	query *string,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbTargetSummary {
	if len(activityTypes) == 0 {
		return []business.SegmentClimbTargetSummary{}
	}

	resolvedMetric := parseSegmentMetric(metric)
	queryFilter := strings.ToLower(strings.TrimSpace(valueOrEmpty(query)))

	attemptsByTarget := collectSegmentAttemptsGroupedByTarget(year, from, to, activityTypes...)
	summaries := make([]business.SegmentClimbTargetSummary, 0, len(attemptsByTarget))
	for _, attempts := range attemptsByTarget {
		if len(attempts) < 2 {
			continue
		}
		summary := buildTargetSummary(attempts, resolvedMetric)
		if queryFilter != "" && !strings.Contains(strings.ToLower(summary.TargetName), queryFilter) {
			continue
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].AttemptsCount != summaries[j].AttemptsCount {
			return summaries[i].AttemptsCount > summaries[j].AttemptsCount
		}
		return strings.ToLower(summaries[i].TargetName) < strings.ToLower(summaries[j].TargetName)
	})

	return summaries
}

func computeSegmentEffortsByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	targetId int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbAttempt {
	if len(activityTypes) == 0 {
		return []business.SegmentClimbAttempt{}
	}

	attemptsByTarget := collectSegmentAttemptsGroupedByTarget(year, from, to, activityTypes...)
	attempts := attemptsByTarget[targetId]
	if len(attempts) == 0 {
		return []business.SegmentClimbAttempt{}
	}

	resolvedMetric := parseSegmentMetric(metric)
	return buildAttempts(attempts, resolvedMetric)
}

func computeSegmentSummaryByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	targetId int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) *SegmentSummaryResult {
	if len(activityTypes) == 0 {
		return nil
	}

	attemptsByTarget := collectSegmentAttemptsGroupedByTarget(year, from, to, activityTypes...)
	attempts := attemptsByTarget[targetId]
	if len(attempts) == 0 {
		return nil
	}

	resolvedMetric := parseSegmentMetric(metric)
	summary := buildTargetSummary(attempts, resolvedMetric)
	progression := buildAttempts(attempts, resolvedMetric)
	topEfforts := rankTopEfforts(progression, resolvedMetric, 3)
	var personalRecord *business.SegmentClimbAttempt
	if len(topEfforts) > 0 {
		best := topEfforts[0]
		personalRecord = &best
	}

	return &SegmentSummaryResult{
		Metric:         string(resolvedMetric),
		Segment:        summary,
		PersonalRecord: personalRecord,
		TopEfforts:     topEfforts,
		Attempts:       progression,
	}
}

func collectSegmentAttemptsGroupedByTarget(
	year *int,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) map[int64][]segmentAttemptRaw {
	filteredActivities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	sort.Slice(filteredActivities, func(i, j int) bool {
		return filteredActivities[i].StartDateLocal < filteredActivities[j].StartDateLocal
	})

	activitySignature := computeSegmentActivitiesSignature(filteredActivities)
	cacheKey := buildSegmentAttemptsCacheKey(year, from, to, activityTypes, activitySignature)
	if cachedAttempts, ok := getSegmentAttemptsFromCache(cacheKey); ok {
		return cachedAttempts
	}

	attemptsByTarget := make(map[int64][]segmentAttemptRaw)
	for _, activity := range filteredActivities {
		detailedActivity := activityprovider.Get().GetCachedDetailedActivity(activity.Id)
		if detailedActivity == nil {
			detailedActivity = activityprovider.Get().GetDetailedActivity(activity.Id)
		}
		if detailedActivity == nil {
			continue
		}

		for _, effort := range detailedActivity.SegmentEfforts {
			if !isCandidateEffort(effort) {
				continue
			}

			day := extractDateOnly(effort.StartDateLocal)
			if from != nil && day < *from {
				continue
			}
			if to != nil && day > *to {
				continue
			}

			attempt := toSegmentAttemptRaw(effort, activity, detailedActivity)

			attemptsByTarget[attempt.targetId] = append(attemptsByTarget[attempt.targetId], attempt)
		}
	}

	attemptsByTarget = splitAttemptsByDirection(attemptsByTarget)

	if len(attemptsByTarget) > 0 {
		storeSegmentAttemptsInCache(cacheKey, attemptsByTarget, false)
		return attemptsByTarget
	}

	// Cache-only fallback:
	// When detailed activities are unavailable in cache (thus no segment_efforts),
	// provide route-level progression by grouping repeated activity names.
	fallbackAttempts := collectNameBasedAttemptsByTarget(filteredActivities, from, to)
	storeSegmentAttemptsInCache(cacheKey, fallbackAttempts, true)
	return fallbackAttempts
}

func collectNameBasedAttemptsByTarget(
	activities []*strava.Activity,
	from *string,
	to *string,
) map[int64][]segmentAttemptRaw {
	groupedActivities := make(map[string][]*strava.Activity)
	displayNameByKey := make(map[string]string)

	for _, activity := range activities {
		if activity == nil {
			continue
		}
		day := extractDateOnly(activity.StartDateLocal)
		if from != nil && day < *from {
			continue
		}
		if to != nil && day > *to {
			continue
		}

		name := strings.TrimSpace(activity.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		groupedActivities[key] = append(groupedActivities[key], activity)
		if displayNameByKey[key] == "" {
			displayNameByKey[key] = name
		}
	}

	attemptsByTarget := make(map[int64][]segmentAttemptRaw)
	for key, grouped := range groupedActivities {
		if len(grouped) < 2 {
			continue
		}

		targetId := nameBasedTargetID(key)
		targetName := displayNameByKey[key]

		for _, activity := range grouped {
			if activity == nil {
				continue
			}

			elapsedSeconds := activity.ElapsedTime
			movingSeconds := activity.MovingTime
			if elapsedSeconds <= 0 {
				continue
			}
			if movingSeconds <= 0 {
				movingSeconds = elapsedSeconds
			}

			averageGrade := 0.0
			if activity.Distance > 0 {
				averageGrade = (activity.TotalElevationGain / activity.Distance) * 100.0
			}

			attempt := segmentAttemptRaw{
				effortId:           activity.Id,
				targetId:           targetId,
				targetName:         targetName,
				targetType:         segmentTargetTypeSegment,
				direction:          segmentDirectionUnknown,
				climbCategory:      0,
				distance:           activity.Distance,
				averageGrade:       averageGrade,
				elapsedTimeSeconds: elapsedSeconds,
				movingTimeSeconds:  movingSeconds,
				speedKph:           computeSpeedKph(activity.Distance, movingSeconds),
				averagePowerWatts:  activity.AverageWatts,
				averageHeartRate:   activity.AverageHeartrate,
				activityDate:       activity.StartDateLocal,
				prRank:             nil,
				activity:           toActivityShort(activity),
			}
			attemptsByTarget[targetId] = append(attemptsByTarget[targetId], attempt)
		}
	}

	return attemptsByTarget
}

func rankTopEfforts(
	attempts []business.SegmentClimbAttempt,
	metric segmentMetric,
	limit int,
) []business.SegmentClimbAttempt {
	if limit <= 0 || len(attempts) == 0 {
		return []business.SegmentClimbAttempt{}
	}

	ranked := append([]business.SegmentClimbAttempt(nil), attempts...)
	sort.Slice(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if metric == segmentMetricSpeed {
			if left.SpeedKph != right.SpeedKph {
				return left.SpeedKph > right.SpeedKph
			}
		} else {
			if left.ElapsedTimeSeconds != right.ElapsedTimeSeconds {
				return left.ElapsedTimeSeconds < right.ElapsedTimeSeconds
			}
		}
		if left.ActivityDate != right.ActivityDate {
			return left.ActivityDate < right.ActivityDate
		}
		return left.Activity.Id < right.Activity.Id
	})

	if limit > len(ranked) {
		limit = len(ranked)
	}
	return ranked[:limit]
}

func extractDateOnly(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toSegmentAttemptRaw(
	effort strava.SegmentEffort,
	activity *strava.Activity,
	detailedActivity *strava.DetailedActivity,
) segmentAttemptRaw {
	effortTargetType := segmentTargetTypeSegment
	if effort.Segment.ClimbCategory > 0 {
		effortTargetType = segmentTargetTypeClimb
	}

	return segmentAttemptRaw{
		effortId:           effort.Id,
		targetId:           effort.Segment.Id,
		targetName:         effort.Segment.Name,
		targetType:         effortTargetType,
		direction:          resolveSegmentDirection(effort, activity, detailedActivity),
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
}

func nameBasedTargetID(key string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(key))
	value := hasher.Sum64()
	// Keep fallback target IDs negative to avoid collisions with real Strava segment IDs.
	return -int64(value%9_000_000_000_000_000 + 1)
}
