package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type MaxAveragePowerStatistic struct {
	ActivityStatistic
	averageWatts *float64
}

func NewMaxAveragePowerStatistic(activities []*strava.Activity) *MaxAveragePowerStatistic {
	stat := &MaxAveragePowerStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Average power",
				Activities: activities,
			},
		},
	}

	var maxActivity *strava.Activity
	for _, activity := range activities {
		if maxActivity == nil || activity.AverageWatts > maxActivity.AverageWatts {
			maxActivity = activity
		}
	}

	if maxActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   maxActivity.Id,
			Name: maxActivity.Name,
			Type: business.ActivityTypes[maxActivity.Type],
		}
		stat.averageWatts = &maxActivity.AverageWatts
	}

	return stat
}

func (stat *MaxAveragePowerStatistic) Value() string {
	if stat.averageWatts != nil {
		return fmt.Sprintf("%0.f W", *stat.averageWatts)
	}
	return "Not available"
}
