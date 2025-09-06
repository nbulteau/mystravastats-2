package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	"time"
)

type MaxElevationInADayStatistic struct {
	BaseStatistic
	mostActiveDay *map[string]float64
}

func NewMaxElevationInADayStatistic(activities []*strava.Activity) *MaxElevationInADayStatistic {
	stat := &MaxElevationInADayStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Max elevation gain in a day",
			Activities: activities,
		},
	}

	activityMap := make(map[string]float64)
	for _, activity := range activities {
		date := activity.StartDateLocal[:10]
		activityMap[date] += activity.TotalElevationGain
	}

	var mostActiveDay *map[string]float64
	var maxElevation float64
	for date, elevation := range activityMap {
		if elevation > maxElevation {
			maxElevation = elevation
			mostActiveDay = &map[string]float64{date: elevation}
		}
	}

	stat.mostActiveDay = mostActiveDay
	return stat
}

func (stat *MaxElevationInADayStatistic) Value() string {
	if stat.mostActiveDay != nil {
		for date, elevation := range *stat.mostActiveDay {
			parsedDate, _ := time.Parse("2006-01-02", date)
			return fmt.Sprintf("%.2f m - %s", elevation, parsedDate.Format(helpers.DateFormatter))
		}
	}
	return "Not available"
}
