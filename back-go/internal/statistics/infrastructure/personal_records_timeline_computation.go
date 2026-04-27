package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/domain/statistics"
	dataqualityInfra "mystravastats/internal/dataquality/infrastructure"
	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strings"
	"time"
)

type personalRecordMetricDefinition struct {
	key                  string
	label                string
	effortExtractor      func(*strava.Activity) *business.ActivityEffort
	score                func(*business.ActivityEffort) float64
	isBetter             func(float64, float64) bool
	valueFormatter       func(*business.ActivityEffort) string
	improvementFormatter func(*business.ActivityEffort, *business.ActivityEffort) string
}

func computePersonalRecordsTimelineByYearMetricAndTypes(year *int, metric *string, activityTypes ...business.ActivityType) []business.PersonalRecordTimelineEntry {
	if len(activityTypes) == 0 {
		return []business.PersonalRecordTimelineEntry{}
	}

	filteredActivities := dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...))
	return buildPersonalRecordsTimeline(filteredActivities, metric, activityTypes)
}

func buildPersonalRecordsTimeline(filteredActivities []*strava.Activity, metric *string, activityTypes []business.ActivityType) []business.PersonalRecordTimelineEntry {
	if len(filteredActivities) == 0 {
		return []business.PersonalRecordTimelineEntry{}
	}

	// Work on a local copy so timeline sorting cannot mutate shared slices.
	activitiesForTimeline := append([]*strava.Activity(nil), filteredActivities...)

	sort.Slice(activitiesForTimeline, func(i, j int) bool {
		left := activitiesForTimeline[i]
		right := activitiesForTimeline[j]

		leftDay := helpers.FirstNonEmpty(helpers.ExtractSortableDay(left.StartDateLocal), helpers.ExtractSortableDay(left.StartDate))
		rightDay := helpers.FirstNonEmpty(helpers.ExtractSortableDay(right.StartDateLocal), helpers.ExtractSortableDay(right.StartDate))
		if leftDay != rightDay {
			return leftDay < rightDay
		}

		leftDateValue := helpers.FirstNonEmpty(left.StartDateLocal, left.StartDate)
		rightDateValue := helpers.FirstNonEmpty(right.StartDateLocal, right.StartDate)
		if leftDateValue != rightDateValue {
			return helpers.IsBeforeActivityDate(leftDateValue, rightDateValue)
		}

		return left.Id < right.Id
	})

	selectedMetrics := getPersonalRecordMetricDefinitions(activityTypes)
	if metric != nil {
		trimmedMetric := strings.TrimSpace(*metric)
		if trimmedMetric != "" {
			filteredMetrics := make([]personalRecordMetricDefinition, 0, len(selectedMetrics))
			for _, definition := range selectedMetrics {
				if definition.key == trimmedMetric {
					filteredMetrics = append(filteredMetrics, definition)
				}
			}
			selectedMetrics = filteredMetrics
		}
	}

	timeline := make([]business.PersonalRecordTimelineEntry, 0)
	for _, definition := range selectedMetrics {
		var bestEffort *business.ActivityEffort
		for _, activity := range activitiesForTimeline {
			effort := definition.effortExtractor(activity)
			if effort == nil {
				continue
			}

			previousBest := bestEffort
			if previousBest == nil || definition.isBetter(definition.score(effort), definition.score(previousBest)) {
				entry := business.PersonalRecordTimelineEntry{
					MetricKey:    definition.key,
					MetricLabel:  definition.label,
					ActivityDate: helpers.FirstNonEmpty(activity.StartDateLocal, activity.StartDate),
					Value:        definition.valueFormatter(effort),
					Activity:     effort.ActivityShort,
				}
				if previousBest != nil {
					entry.PreviousValue = stringPtr(definition.valueFormatter(previousBest))
					entry.Improvement = stringPtr(definition.improvementFormatter(previousBest, effort))
				}
				timeline = append(timeline, entry)
				bestEffort = effort
			}
		}
	}

	sort.Slice(timeline, func(i, j int) bool {
		leftDay := helpers.ExtractSortableDay(timeline[i].ActivityDate)
		rightDay := helpers.ExtractSortableDay(timeline[j].ActivityDate)
		if leftDay != rightDay {
			return leftDay < rightDay
		}
		if timeline[i].ActivityDate != timeline[j].ActivityDate {
			return helpers.IsBeforeActivityDate(timeline[i].ActivityDate, timeline[j].ActivityDate)
		}
		return timeline[i].Activity.Id < timeline[j].Activity.Id
	})

	return timeline
}

func getPersonalRecordMetricDefinitions(activityTypes []business.ActivityType) []personalRecordMetricDefinition {
	switch resolvePrimaryActivityType(activityTypes) {
	case business.Run:
		return buildRunMetricDefinitions()
	case business.InlineSkate:
		return buildInlineSkateMetricDefinitions()
	case business.AlpineSki:
		return buildAlpineSkiMetricDefinitions()
	case business.Hike:
		return buildActivityRecordMetricDefinitions()
	default:
		return buildRideMetricDefinitions()
	}
}

func resolvePrimaryActivityType(activityTypes []business.ActivityType) business.ActivityType {
	for _, activityType := range activityTypes {
		if activityType == business.Run || activityType == business.TrailRun {
			return business.Run
		}
	}
	for _, activityType := range activityTypes {
		if activityType == business.InlineSkate {
			return business.InlineSkate
		}
	}
	for _, activityType := range activityTypes {
		if activityType == business.Hike || activityType == business.Walk {
			return business.Hike
		}
	}
	for _, activityType := range activityTypes {
		if activityType == business.AlpineSki {
			return business.AlpineSki
		}
	}
	return business.Ride
}

func buildRunMetricDefinitions() []personalRecordMetricDefinition {
	return append([]personalRecordMetricDefinition{
		bestTimeForDistanceMetric("best-time-200m", "Best 200 m", 200.0),
		bestTimeForDistanceMetric("best-time-400m", "Best 400 m", 400.0),
		bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
		bestTimeForDistanceMetric("best-time-5000m", "Best 5000 m", 5000.0),
		bestTimeForDistanceMetric("best-time-10000m", "Best 10000 m", 10000.0),
		bestTimeForDistanceMetric("best-time-half-marathon", "Best half Marathon", 21097.0),
		bestTimeForDistanceMetric("best-time-marathon", "Best Marathon", 42195.0),
		bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60*60),
		bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2*60*60),
		bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3*60*60),
		bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4*60*60),
		bestDistanceForTimeMetric("best-distance-5h", "Best 5 h", 5*60*60),
		bestDistanceForTimeMetric("best-distance-6h", "Best 6 h", 6*60*60),
	}, buildActivityRecordMetricDefinitions()...)
}

func buildRideMetricDefinitions() []personalRecordMetricDefinition {
	return append([]personalRecordMetricDefinition{
		bestTimeForDistanceMetric("best-time-250m", "Best 250 m", 250.0),
		bestTimeForDistanceMetric("best-time-500m", "Best 500 m", 500.0),
		bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
		bestTimeForDistanceMetric("best-time-5km", "Best 5 km", 5000.0),
		bestTimeForDistanceMetric("best-time-10km", "Best 10 km", 10000.0),
		bestTimeForDistanceMetric("best-time-20km", "Best 20 km", 20000.0),
		bestTimeForDistanceMetric("best-time-50km", "Best 50 km", 50000.0),
		bestTimeForDistanceMetric("best-time-100km", "Best 100 km", 100000.0),
		bestDistanceForTimeMetric("best-distance-30min", "Best 30 min", 30*60),
		bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60*60),
		bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2*60*60),
		bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3*60*60),
		bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4*60*60),
		bestDistanceForTimeMetric("best-distance-5h", "Best 5 h", 5*60*60),
		bestGradientForDistanceMetric("best-gradient-250m", "Max gradient for 250 m", 250.0),
		bestGradientForDistanceMetric("best-gradient-500m", "Max gradient for 500 m", 500.0),
		bestGradientForDistanceMetric("best-gradient-1000m", "Max gradient for 1000 m", 1000.0),
		bestGradientForDistanceMetric("best-gradient-5km", "Max gradient for 5 km", 5000.0),
		bestGradientForDistanceMetric("best-gradient-10km", "Max gradient for 10 km", 10000.0),
		bestGradientForDistanceMetric("best-gradient-20km", "Max gradient for 20 km", 20000.0),
		bestPowerForTimeMetric("best-power-20min", "Best average power for 20 min", 20*60),
		bestPowerForTimeMetric("best-power-1h", "Best average power for 1 h", 60*60),
	}, buildActivityRecordMetricDefinitions()...)
}

func buildAlpineSkiMetricDefinitions() []personalRecordMetricDefinition {
	return append([]personalRecordMetricDefinition{
		bestTimeForDistanceMetric("best-time-250m", "Best 250 m", 250.0),
		bestTimeForDistanceMetric("best-time-500m", "Best 500 m", 500.0),
		bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
		bestTimeForDistanceMetric("best-time-5km", "Best 5 km", 5000.0),
		bestTimeForDistanceMetric("best-time-10km", "Best 10 km", 10000.0),
		bestTimeForDistanceMetric("best-time-20km", "Best 20 km", 20000.0),
		bestTimeForDistanceMetric("best-time-50km", "Best 50 km", 50000.0),
		bestTimeForDistanceMetric("best-time-100km", "Best 100 km", 100000.0),
		bestDistanceForTimeMetric("best-distance-30min", "Best 30 min", 30*60),
		bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60*60),
		bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2*60*60),
		bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3*60*60),
		bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4*60*60),
		bestDistanceForTimeMetric("best-distance-5h", "Best 5 h", 5*60*60),
	}, buildActivityRecordMetricDefinitions()...)
}

func buildInlineSkateMetricDefinitions() []personalRecordMetricDefinition {
	return append([]personalRecordMetricDefinition{
		bestTimeForDistanceMetric("best-time-200m", "Best 200 m", 200.0),
		bestTimeForDistanceMetric("best-time-400m", "Best 400 m", 400.0),
		bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
		bestTimeForDistanceMetric("best-time-10000m", "Best 10000 m", 10000.0),
		bestTimeForDistanceMetric("best-time-half-marathon", "Best half Marathon", 21097.0),
		bestTimeForDistanceMetric("best-time-marathon", "Best Marathon", 42195.0),
		bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60*60),
		bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2*60*60),
		bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3*60*60),
		bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4*60*60),
	}, buildActivityRecordMetricDefinitions()...)
}

func buildActivityRecordMetricDefinitions() []personalRecordMetricDefinition {
	return []personalRecordMetricDefinition{
		maxDistanceActivityMetric("max-distance-activity", "Max distance"),
		maxSpeedActivityMetric("max-speed-activity", "Max speed"),
		maxMovingTimeActivityMetric("max-moving-time-activity", "Max moving time"),
		maxDistanceInDayMetric("max-distance-in-a-day", "Max distance in a day"),
		maxElevationActivityMetric("max-elevation-activity", "Max elevation"),
		maxElevationInDayMetric("max-elevation-in-a-day", "Max elevation gain in a day"),
		highestPointActivityMetric("highest-point-activity", "Highest point"),
	}
}

func maxDistanceActivityMetric(key, label string) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.Distance <= 0 {
				return nil
			}
			return activityToEffort(activity, activity.Distance, maxInt(activity.MovingTime, 1))
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return formatDistanceInKm(effort.Distance)
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("%s farther", formatDistance(current.Distance-previous.Distance))
		},
	}
}

func maxSpeedActivityMetric(key, label string) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.MaxSpeed <= 0 {
				return nil
			}
			return activityToEffort(activity, activity.MaxSpeed, maxInt(activity.MovingTime, 1))
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return formatActivitySpeedFromMSSpeed(effort.Distance, effort.ActivityShort.Type)
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("%+.2f km/h", (current.Distance-previous.Distance)*3.6)
		},
	}
}

func maxMovingTimeActivityMetric(key, label string) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.MovingTime <= 0 {
				return nil
			}
			return activityToEffort(activity, activity.Distance, activity.MovingTime)
		},
		score:    func(effort *business.ActivityEffort) float64 { return float64(effort.Seconds) },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return helpers.FormatSeconds(effort.Seconds)
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			gain := current.Seconds - previous.Seconds
			if gain < 0 {
				gain = 0
			}
			return fmt.Sprintf("%s longer", helpers.FormatSeconds(gain))
		},
	}
}

func maxDistanceInDayMetric(key, label string) personalRecordMetricDefinition {
	distanceByDay := make(map[string]float64)
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.Distance <= 0 {
				return nil
			}
			day := activityDay(activity.StartDateLocal)
			distanceByDay[day] += activity.Distance
			return activityToEffortWithLabel(activity, distanceByDay[day], maxInt(activity.MovingTime, 1), day)
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return fmt.Sprintf("%s - %s", formatDistanceInKm(effort.Distance), formatRecordDay(effort.Label))
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("%s farther", formatDistance(current.Distance-previous.Distance))
		},
	}
}

func maxElevationActivityMetric(key, label string) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.TotalElevationGain <= 0 {
				return nil
			}
			return activityToEffort(activity, activity.TotalElevationGain, maxInt(activity.MovingTime, 1))
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return fmt.Sprintf("%.2f m", effort.Distance)
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("+%.2f m", current.Distance-previous.Distance)
		},
	}
}

func maxElevationInDayMetric(key, label string) personalRecordMetricDefinition {
	elevationByDay := make(map[string]float64)
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.TotalElevationGain <= 0 {
				return nil
			}
			day := activityDay(activity.StartDateLocal)
			elevationByDay[day] += activity.TotalElevationGain
			return activityToEffortWithLabel(activity, elevationByDay[day], maxInt(activity.MovingTime, 1), day)
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return fmt.Sprintf("%.2f m - %s", effort.Distance, formatRecordDay(effort.Label))
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("+%.2f m", current.Distance-previous.Distance)
		},
	}
}

func highestPointActivityMetric(key, label string) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			if activity == nil || activity.ElevHigh <= 0 {
				return nil
			}
			return activityToEffort(activity, activity.ElevHigh, maxInt(activity.MovingTime, 1))
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return fmt.Sprintf("%.2f m", effort.Distance)
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("+%.2f m", current.Distance-previous.Distance)
		},
	}
}

func bestTimeForDistanceMetric(key, label string, distance float64) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			return statistics.BestTimeEffort(*activity, distance)
		},
		score:    func(effort *business.ActivityEffort) float64 { return float64(effort.Seconds) },
		isBetter: func(score, previousScore float64) bool { return score < previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return fmt.Sprintf("%s (%s)", helpers.FormatSeconds(effort.Seconds), effort.GetFormattedSpeed())
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			gainedSeconds := previous.Seconds - current.Seconds
			if gainedSeconds < 0 {
				gainedSeconds = 0
			}
			return fmt.Sprintf("%s faster", helpers.FormatSeconds(gainedSeconds))
		},
	}
}

func bestDistanceForTimeMetric(key, label string, seconds int) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			return statistics.BestDistanceEffort(*activity, seconds)
		},
		score:    func(effort *business.ActivityEffort) float64 { return effort.Distance },
		isBetter: func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string {
			return fmt.Sprintf("%s (%s)", formatDistance(effort.Distance), effort.GetFormattedSpeed())
		},
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("%s farther", formatDistance(current.Distance-previous.Distance))
		},
	}
}

func bestPowerForTimeMetric(key, label string, seconds int) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			return statistics.BestPowerForTime(*activity, seconds)
		},
		score: func(effort *business.ActivityEffort) float64 {
			if effort.AveragePower == nil {
				return math.Inf(-1)
			}
			return *effort.AveragePower
		},
		isBetter:       func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string { return effort.GetFormattedPower() },
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			delta := int(math.Round(powerValue(current) - powerValue(previous)))
			return fmt.Sprintf("%+d W", delta)
		},
	}
}

func bestGradientForDistanceMetric(key, label string, distance float64) personalRecordMetricDefinition {
	return personalRecordMetricDefinition{
		key:   key,
		label: label,
		effortExtractor: func(activity *strava.Activity) *business.ActivityEffort {
			return statistics.BestElevationEffort(*activity, distance)
		},
		score:          func(effort *business.ActivityEffort) float64 { return effort.GetGradient() },
		isBetter:       func(score, previousScore float64) bool { return score > previousScore },
		valueFormatter: func(effort *business.ActivityEffort) string { return effort.GetFormattedGradient() },
		improvementFormatter: func(previous, current *business.ActivityEffort) string {
			return fmt.Sprintf("+%.2f %%", current.GetGradient()-previous.GetGradient())
		},
	}
}

func formatDistance(distanceInMeters float64) string {
	if distanceInMeters >= 1000 {
		return fmt.Sprintf("%.2f km", distanceInMeters/1000)
	}
	return fmt.Sprintf("%.0f m", distanceInMeters)
}

func powerValue(effort *business.ActivityEffort) float64 {
	if effort == nil || effort.AveragePower == nil {
		return 0
	}
	return *effort.AveragePower
}

func activityToEffort(activity *strava.Activity, scoreValue float64, seconds int) *business.ActivityEffort {
	if activity == nil {
		return nil
	}
	return activityToEffortWithLabel(activity, scoreValue, seconds, activity.Name)
}

func activityToEffortWithLabel(activity *strava.Activity, scoreValue float64, seconds int, label string) *business.ActivityEffort {
	if activity == nil {
		return nil
	}
	return &business.ActivityEffort{
		Distance:      scoreValue,
		Seconds:       seconds,
		DeltaAltitude: activity.TotalElevationGain,
		IdxStart:      0,
		IdxEnd:        0,
		Label:         label,
		ActivityShort: business.ActivityShort{
			Id:   activity.Id,
			Name: activity.Name,
			Type: business.ActivityTypes[activity.Type],
		},
	}
}

func formatActivitySpeedFromMSSpeed(speedMS float64, activityType business.ActivityType) string {
	if speedMS <= 0 {
		return "Not available"
	}
	if activityType == business.Run || activityType == business.TrailRun {
		paceSeconds := int(math.Round(1000.0 / speedMS))
		return fmt.Sprintf("%s/km", helpers.FormatSeconds(paceSeconds))
	}
	return fmt.Sprintf("%.02f km/h", speedMS*3.6)
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func activityDay(startDateLocal string) string {
	if len(startDateLocal) >= 10 {
		return startDateLocal[:10]
	}
	return strings.TrimSpace(startDateLocal)
}

func formatRecordDay(day string) string {
	parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(day))
	if err != nil {
		return day
	}
	return parsedDate.Format(helpers.DateFormatter)
}

func formatDistanceInKm(distanceMeters float64) string {
	return fmt.Sprintf("%.2f km", distanceMeters/1000.0)
}

func stringPtr(value string) *string {
	v := value
	return &v
}
