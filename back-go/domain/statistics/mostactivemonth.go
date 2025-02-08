package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
	"time"
)

type MostActiveMonthStatistic struct {
	BaseStatistic
	mostActiveMonth *map[string]float64
}

func NewMostActiveMonthStatistic(activities []*strava.Activity) *MostActiveMonthStatistic {
	stat := &MostActiveMonthStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Most active month",
			Activities: activities,
		},
	}

	activityMap := make(map[string]float64)
	for _, activity := range activities {
		date := activity.StartDateLocal[:7]
		activityMap[date] += activity.Distance
	}

	var mostActiveMonth *map[string]float64
	var maxDistance float64
	for date, distance := range activityMap {
		if distance > maxDistance {
			maxDistance = distance
			mostActiveMonth = &map[string]float64{date: distance}
		}
	}

	stat.mostActiveMonth = mostActiveMonth
	return stat
}

func (stat *MostActiveMonthStatistic) Value() string {
	if stat.mostActiveMonth != nil {
		for date, distance := range *stat.mostActiveMonth {
			parsedDate, _ := time.Parse("2006-01", date)
			return fmt.Sprintf("%s with %.2f km", parsedDate.Format("January 2006"), distance/1000)
		}
	}
	return "Not available"
}
