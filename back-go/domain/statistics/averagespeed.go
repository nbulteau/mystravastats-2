package statistics

import (
	"fmt"
	"mystravastats/domain/strava"
)

type AverageSpeedStatistic struct {
	BaseStatistic
	averageSpeed *float64
}

func NewAverageSpeedStatistic(activities []*strava.Activity) *AverageSpeedStatistic {
	stat := &AverageSpeedStatistic{
		BaseStatistic: BaseStatistic{
			name:       "Average speed",
			Activities: activities,
		},
	}

	var totalSpeed float64
	var count int

	for _, activity := range activities {
		if activity.AverageSpeed > 0 {
			totalSpeed += activity.AverageSpeed
			count++
		}
	}

	if count > 0 {
		averageSpeed := totalSpeed / float64(count)
		stat.averageSpeed = &averageSpeed
	}

	return stat
}

func (stat *AverageSpeedStatistic) Value() string {
	if stat.averageSpeed != nil {
		return fmt.Sprintf("%.02f km/h", *stat.averageSpeed*3.6)
	}
	return "Not available"
}
