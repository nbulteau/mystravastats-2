package services

import (
	"fmt"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	"sort"
	"strings"
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

func FetchPersonalRecordsTimelineByActivityTypeAndYear(year *int, metric *string, activityTypes ...business.ActivityType) []business.PersonalRecordTimelineEntry {
	if len(activityTypes) == 0 {
		return []business.PersonalRecordTimelineEntry{}
	}

	filteredActivities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	if len(filteredActivities) == 0 {
		return []business.PersonalRecordTimelineEntry{}
	}

	sort.Slice(filteredActivities, func(i, j int) bool {
		return filteredActivities[i].StartDateLocal < filteredActivities[j].StartDateLocal
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
		for _, activity := range filteredActivities {
			effort := definition.effortExtractor(activity)
			if effort == nil {
				continue
			}

			previousBest := bestEffort
			if previousBest == nil || definition.isBetter(definition.score(effort), definition.score(previousBest)) {
				entry := business.PersonalRecordTimelineEntry{
					MetricKey:    definition.key,
					MetricLabel:  definition.label,
					ActivityDate: activity.StartDateLocal,
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
		return timeline[i].ActivityDate < timeline[j].ActivityDate
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
		return []personalRecordMetricDefinition{}
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
		if activityType == business.Hike {
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
	return []personalRecordMetricDefinition{
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
	}
}

func buildRideMetricDefinitions() []personalRecordMetricDefinition {
	return []personalRecordMetricDefinition{
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
	}
}

func buildAlpineSkiMetricDefinitions() []personalRecordMetricDefinition {
	return []personalRecordMetricDefinition{
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
	}
}

func buildInlineSkateMetricDefinitions() []personalRecordMetricDefinition {
	return []personalRecordMetricDefinition{
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

func stringPtr(value string) *string {
	v := value
	return &v
}
