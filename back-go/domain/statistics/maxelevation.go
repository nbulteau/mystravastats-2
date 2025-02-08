package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type MaxElevationStatistic struct {
	ActivityStatistic
	maxTotalElevationGain *float64
}

func NewMaxElevationStatistic(activities []*strava.Activity) *MaxElevationStatistic {
	stat := &MaxElevationStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Max elevation",
				Activities: activities,
			},
		},
	}

	var maxTotalElevationGainActivity *strava.Activity
	for _, activity := range activities {
		if maxTotalElevationGainActivity == nil || activity.TotalElevationGain > maxTotalElevationGainActivity.TotalElevationGain {
			maxTotalElevationGainActivity = activity
		}
	}

	if maxTotalElevationGainActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   maxTotalElevationGainActivity.Id,
			Name: maxTotalElevationGainActivity.Name,
			Type: business.ActivityTypes[maxTotalElevationGainActivity.Type],
		}
		stat.maxTotalElevationGain = &maxTotalElevationGainActivity.TotalElevationGain
	}

	return stat
}

func (stat *MaxElevationStatistic) Value() string {
	if stat.maxTotalElevationGain != nil {
		return fmt.Sprintf("%.2f m", *stat.maxTotalElevationGain)
	}
	return "Not available"
}
