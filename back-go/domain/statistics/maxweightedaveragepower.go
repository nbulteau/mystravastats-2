package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type MaxWeightedAveragePowerStatistic struct {
	ActivityStatistic
	maxWeightedAverageWatts *int
}

func NewMaxWeightedAveragePowerStatistic(activities []*strava.Activity) *MaxWeightedAveragePowerStatistic {
	stat := &MaxWeightedAveragePowerStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Weighted average power",
				Activities: activities,
			},
		},
	}

	var maxActivity *strava.Activity
	for _, activity := range activities {
		if maxActivity == nil || activity.WeightedAverageWatts > maxActivity.WeightedAverageWatts {
			maxActivity = activity
		}
	}

	if maxActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   maxActivity.Id,
			Name: maxActivity.Name,
			Type: business.ActivityTypes[maxActivity.Type],
		}
		stat.maxWeightedAverageWatts = &maxActivity.WeightedAverageWatts
	}

	return stat
}

func (stat *MaxWeightedAveragePowerStatistic) Value() string {
	if stat.maxWeightedAverageWatts != nil {
		return fmt.Sprintf("%d W", *stat.maxWeightedAverageWatts)
	}
	return "Not available"
}
