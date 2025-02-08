package statistics

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

type MaxDistanceStatistic struct {
	ActivityStatistic
	maxDistance *float64
}

func NewMaxDistanceStatistic(activities []*strava.Activity) *MaxDistanceStatistic {
	stat := &MaxDistanceStatistic{
		ActivityStatistic: ActivityStatistic{
			BaseStatistic: BaseStatistic{
				name:       "Max distance",
				Activities: activities,
			},
		},
	}

	var maxDistanceActivity *strava.Activity
	for _, activity := range activities {
		if maxDistanceActivity == nil || activity.Distance > maxDistanceActivity.Distance {
			maxDistanceActivity = activity
		}
	}

	if maxDistanceActivity != nil {
		stat.activity = &business.ActivityShort{
			Id:   maxDistanceActivity.Id,
			Name: maxDistanceActivity.Name,
			Type: business.ActivityTypes[maxDistanceActivity.Type],
		}
		stat.maxDistance = &maxDistanceActivity.Distance
	}

	return stat
}

func (stat *MaxDistanceStatistic) Value() string {
	if stat.maxDistance != nil {
		return fmt.Sprintf("%.2f km", *stat.maxDistance/1000)
	}
	return "Not available"
}
