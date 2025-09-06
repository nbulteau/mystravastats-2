package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	"time"
)

type MaxDistanceInADayStatistic struct {
	BaseStatistic
	mostActiveDay *map[string]float64
}

func NewMaxDistanceInADayStatistic(activities []*strava.Activity) *MaxDistanceInADayStatistic {
	stat := &MaxDistanceInADayStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Max distance in a day",
			Activities: activities,
		},
	}

	activityMap := make(map[string]float64)
	for _, activity := range activities {
		date := activity.StartDateLocal[:10]
		activityMap[date] += activity.Distance
	}

	var mostActiveDay *map[string]float64
	var maxDistance float64
	for date, distance := range activityMap {
		if distance > maxDistance {
			maxDistance = distance
			mostActiveDay = &map[string]float64{date: distance}
		}
	}

	stat.mostActiveDay = mostActiveDay
	return stat
}

func (stat *MaxDistanceInADayStatistic) Value() string {
	if stat.mostActiveDay != nil {
		for date, distance := range *stat.mostActiveDay {
			parsedDate, _ := time.Parse("2006-01-02", date)
			return fmt.Sprintf("%.2f km - %s", distance/1000, parsedDate.Format(helpers.DateFormatter))
		}
	}
	return "Not available"
}
