package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type HighestPointStatistic struct {
	ActivityStatistic
	highestElevation *float64
}

func NewHighestPointStatistic(activities []*strava.Activity) *HighestPointStatistic {
	stat := &HighestPointStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Highest point",
				Activities: activities,
			},
		},
	}

	var highestElevationActivity *strava.Activity
	for _, activity := range activities {
		if highestElevationActivity == nil || activity.ElevHigh > highestElevationActivity.ElevHigh {
			highestElevationActivity = activity
		}
	}

	if highestElevationActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   highestElevationActivity.Id,
			Name: highestElevationActivity.Name,
			Type: business.ActivityTypes[highestElevationActivity.Type],
		}
		stat.highestElevation = &highestElevationActivity.ElevHigh
	}

	return stat
}

func (stat *HighestPointStatistic) Value() string {
	if stat.highestElevation != nil {
		return fmt.Sprintf("%.2f m", *stat.highestElevation)
	}
	return "Not available"
}
